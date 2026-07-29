package email

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

const (
	ProviderTencentSES = "tencent_ses"
	ProviderResend     = "resend"
)

type OTPSender interface {
	SendOTP(ctx context.Context, to string, subject string, code string, purpose string, schoolName string, expireMinutes int) error
}

type OTPProvider struct {
	Name   string
	Sender OTPSender
}

type DeliveryPolicy struct {
	Mode        string                 `json:"mode"`
	MaxAttempts int                    `json:"maxAttempts"`
	Providers   []DeliveryPolicyEntry  `json:"providers"`
	Extra       map[string]interface{} `json:"-"`
}

type DeliveryPolicyEntry struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
}

type PolicyProvider interface {
	GetEmailDeliveryPolicy(ctx context.Context) (DeliveryPolicy, error)
}

type FailoverOTPSender struct {
	providers      map[string]OTPSender
	defaultPolicy  DeliveryPolicy
	policyProvider PolicyProvider
	mu             sync.Mutex
	next           map[int]int
}

func NewFailoverOTPSender(providers []OTPProvider, defaultPolicy DeliveryPolicy, policyProvider PolicyProvider) (*FailoverOTPSender, error) {
	lookup := make(map[string]OTPSender, len(providers))
	for _, provider := range providers {
		name := NormalizeProviderName(provider.Name)
		if name == "" {
			return nil, errors.New("email provider name is required")
		}
		if provider.Sender == nil {
			return nil, fmt.Errorf("email provider %s sender is nil", name)
		}
		lookup[name] = provider.Sender
	}
	if len(lookup) == 0 {
		return nil, errors.New("at least one email provider is required")
	}
	defaultPolicy = NormalizeDeliveryPolicy(defaultPolicy, lookup)
	order := ResolveProviderOrder(defaultPolicy, lookup, nil)
	if len(order) == 0 {
		return nil, errors.New("email delivery policy has no enabled provider")
	}
	return &FailoverOTPSender{
		providers:      lookup,
		defaultPolicy:  defaultPolicy,
		policyProvider: policyProvider,
		next:           make(map[int]int),
	}, nil
}

func NormalizeProviderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func ParseDeliveryPolicy(raw string) (DeliveryPolicy, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DeliveryPolicy{}, nil
	}
	var policy DeliveryPolicy
	if err := json.Unmarshal([]byte(trimmed), &policy); err != nil {
		return DeliveryPolicy{}, fmt.Errorf("parse email delivery policy: %w", err)
	}
	return policy, nil
}

func NormalizeDeliveryPolicy(policy DeliveryPolicy, available map[string]OTPSender) DeliveryPolicy {
	policy.Mode = strings.ToLower(strings.TrimSpace(policy.Mode))
	if policy.Mode == "" {
		policy.Mode = "priority"
	}
	if policy.Mode != "priority" && policy.Mode != "weighted" {
		policy.Mode = "priority"
	}
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = len(available)
	}
	seen := make(map[string]struct{}, len(policy.Providers))
	normalized := make([]DeliveryPolicyEntry, 0, len(policy.Providers)+len(available))
	for _, entry := range policy.Providers {
		name := NormalizeProviderName(entry.Name)
		if name == "" {
			continue
		}
		if _, ok := available[name]; !ok {
			continue
		}
		if entry.Weight <= 0 {
			entry.Weight = 1
		}
		entry.Name = name
		normalized = append(normalized, entry)
		seen[name] = struct{}{}
	}
	for name := range available {
		if _, ok := seen[name]; ok {
			continue
		}
		normalized = append(normalized, DeliveryPolicyEntry{
			Name:     name,
			Enabled:  true,
			Priority: 100,
			Weight:   1,
		})
	}
	policy.Providers = normalized
	return policy
}

func ResolveProviderOrder(policy DeliveryPolicy, available map[string]OTPSender, rotate func(priority int, totalWeight int) int) []string {
	policy = NormalizeDeliveryPolicy(policy, available)
	entries := make([]DeliveryPolicyEntry, 0, len(policy.Providers))
	for _, entry := range policy.Providers {
		if !entry.Enabled {
			continue
		}
		if _, ok := available[entry.Name]; !ok {
			continue
		}
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Priority != entries[j].Priority {
			return entries[i].Priority < entries[j].Priority
		}
		if entries[i].Weight != entries[j].Weight {
			return entries[i].Weight > entries[j].Weight
		}
		return entries[i].Name < entries[j].Name
	})
	if policy.Mode != "weighted" {
		return limitProviderOrder(entries, policy.MaxAttempts)
	}
	ordered := make([]DeliveryPolicyEntry, 0, len(entries))
	for start := 0; start < len(entries); {
		end := start + 1
		for end < len(entries) && entries[end].Priority == entries[start].Priority {
			end++
		}
		group := append([]DeliveryPolicyEntry(nil), entries[start:end]...)
		totalWeight := 0
		for _, entry := range group {
			if entry.Weight > 0 {
				totalWeight += entry.Weight
			}
		}
		if totalWeight > 0 && rotate != nil && len(group) > 1 {
			offset := rotate(entries[start].Priority, totalWeight)
			group = weightedGroupOrder(group, offset)
		}
		ordered = append(ordered, group...)
		start = end
	}
	return limitProviderOrder(ordered, policy.MaxAttempts)
}

func weightedGroupOrder(entries []DeliveryPolicyEntry, offset int) []DeliveryPolicyEntry {
	if len(entries) <= 1 {
		return entries
	}
	total := 0
	for _, entry := range entries {
		total += max(entry.Weight, 1)
	}
	if total <= 0 {
		return entries
	}
	offset %= total
	selected := 0
	cursor := 0
	for i, entry := range entries {
		cursor += max(entry.Weight, 1)
		if offset < cursor {
			selected = i
			break
		}
	}
	ordered := make([]DeliveryPolicyEntry, 0, len(entries))
	ordered = append(ordered, entries[selected])
	ordered = append(ordered, entries[:selected]...)
	ordered = append(ordered, entries[selected+1:]...)
	return ordered
}

func limitProviderOrder(entries []DeliveryPolicyEntry, maxAttempts int) []string {
	if maxAttempts <= 0 || maxAttempts > len(entries) {
		maxAttempts = len(entries)
	}
	out := make([]string, 0, maxAttempts)
	for i := 0; i < maxAttempts; i++ {
		out = append(out, entries[i].Name)
	}
	return out
}

func (s *FailoverOTPSender) SendOTP(ctx context.Context, to string, subject string, code string, purpose string, schoolName string, expireMinutes int) error {
	if s == nil {
		return errors.New("email failover sender is nil")
	}
	policy := s.defaultPolicy
	if s.policyProvider != nil {
		loaded, err := s.policyProvider.GetEmailDeliveryPolicy(ctx)
		if err != nil {
			logger.L().Warn("email delivery policy unavailable; falling back to startup policy", zap.Error(err))
		} else {
			policy = loaded
		}
	}
	order := ResolveProviderOrder(policy, s.providers, s.rotate)
	if len(order) == 0 {
		return errors.New("email delivery policy has no enabled provider")
	}
	attempts := make([]string, 0, len(order))
	for _, name := range order {
		sender := s.providers[name]
		if sender == nil {
			continue
		}
		if err := sender.SendOTP(ctx, to, subject, code, purpose, schoolName, expireMinutes); err != nil {
			attempts = append(attempts, fmt.Sprintf("%s: %v", name, err))
			logger.L().Warn("email provider failed; trying next provider",
				zap.String("provider", name),
				zap.Error(err),
			)
			continue
		}
		return nil
	}
	return fmt.Errorf("all email providers failed: %s", strings.Join(attempts, "; "))
}

func (s *FailoverOTPSender) rotate(priority int, totalWeight int) int {
	if totalWeight <= 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.next[priority]
	s.next[priority] = (current + 1) % totalWeight
	return current
}
