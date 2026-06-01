package systemconfig

import (
	"encoding/json"
	"fmt"
	"strings"
)

const EmailDeliveryPolicyKey = "email.delivery_policy"

type EmailDeliveryPolicy struct {
	Mode        string                        `json:"mode"`
	MaxAttempts int                           `json:"maxAttempts"`
	Providers   []EmailDeliveryPolicyProvider `json:"providers"`
}

type EmailDeliveryPolicyProvider struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
}

func ValidateEmailDeliveryPolicy(value string) error {
	var policy EmailDeliveryPolicy
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &policy); err != nil {
		return fmt.Errorf("must be a JSON object")
	}
	mode := strings.TrimSpace(policy.Mode)
	if mode != "priority" && mode != "weighted" {
		return fmt.Errorf("mode must be priority or weighted")
	}
	if policy.MaxAttempts <= 0 || policy.MaxAttempts > 10 {
		return fmt.Errorf("maxAttempts must be between 1 and 10")
	}
	if len(policy.Providers) == 0 {
		return fmt.Errorf("providers must not be empty")
	}
	seen := make(map[string]struct{}, len(policy.Providers))
	for _, provider := range policy.Providers {
		name := strings.TrimSpace(provider.Name)
		switch name {
		case "tencent_ses", "resend":
		default:
			return fmt.Errorf("provider %q is not supported", name)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("provider %q must not be duplicated", name)
		}
		seen[name] = struct{}{}
		if provider.Priority < 0 || provider.Priority > 999 {
			return fmt.Errorf("provider %q priority must be between 0 and 999", name)
		}
		if provider.Weight <= 0 || provider.Weight > 1000 {
			return fmt.Errorf("provider %q weight must be between 1 and 1000", name)
		}
	}
	return nil
}
