package systemconfig

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ReviewAccessSchoolIDsKey       = "review_access_school_ids"
	ReviewPreviewTitleCharsKey     = "review_preview_title_chars"
	ReviewPreviewContentCharsKey   = "review_preview_content_chars"
	ReviewPreviewContentPercentKey = "review_preview_content_percent"

	LegacyReviewPreviewCharsKey   = "review_preview_chars"
	LegacyReviewPreviewPercentKey = "review_preview_percent"

	defaultReviewPreviewTitleRunes   = 24
	defaultReviewPreviewContentRunes = 120
	defaultReviewPreviewContentPct   = 100
)

type ReviewAccessPolicySnapshot struct {
	AllowedSchoolIDs    []string
	PreviewTitleRunes   int
	PreviewContentRunes int
	PreviewContentPct   int
	LoadedAt            time.Time
}

var reviewAccessPolicyCache = struct {
	mu       sync.RWMutex
	snapshot ReviewAccessPolicySnapshot
}{
	snapshot: DefaultReviewAccessPolicySnapshot(),
}

func DefaultReviewAccessPolicySnapshot() ReviewAccessPolicySnapshot {
	return ReviewAccessPolicySnapshot{
		PreviewTitleRunes:   defaultReviewPreviewTitleRunes,
		PreviewContentRunes: defaultReviewPreviewContentRunes,
		PreviewContentPct:   defaultReviewPreviewContentPct,
	}
}

func (s ReviewAccessPolicySnapshot) AllowsSchool(schoolID string) bool {
	trimmed := strings.TrimSpace(schoolID)
	if trimmed == "" || len(s.AllowedSchoolIDs) == 0 {
		return false
	}
	for _, allowedSchoolID := range s.AllowedSchoolIDs {
		if trimmed == allowedSchoolID {
			return true
		}
	}
	return false
}

func GetReviewAccessPolicySnapshot() ReviewAccessPolicySnapshot {
	reviewAccessPolicyCache.mu.RLock()
	defer reviewAccessPolicyCache.mu.RUnlock()

	snapshot := reviewAccessPolicyCache.snapshot
	snapshot.AllowedSchoolIDs = append([]string(nil), snapshot.AllowedSchoolIDs...)
	return snapshot
}

func SetReviewAccessPolicySnapshot(snapshot ReviewAccessPolicySnapshot) {
	reviewAccessPolicyCache.mu.Lock()
	defer reviewAccessPolicyCache.mu.Unlock()

	reviewAccessPolicyCache.snapshot = normalizeReviewAccessPolicySnapshot(snapshot)
}

func InvalidateReviewAccessPolicySnapshot() {
	SetReviewAccessPolicySnapshot(ReviewAccessPolicySnapshot{})
}

func AffectsReviewAccessPolicy(key string) bool {
	switch strings.TrimSpace(key) {
	case ReviewAccessSchoolIDsKey,
		ReviewPreviewTitleCharsKey,
		ReviewPreviewContentCharsKey,
		ReviewPreviewContentPercentKey,
		LegacyReviewPreviewCharsKey,
		LegacyReviewPreviewPercentKey:
		return true
	default:
		return false
	}
}

func normalizeReviewAccessPolicySnapshot(snapshot ReviewAccessPolicySnapshot) ReviewAccessPolicySnapshot {
	snapshot.AllowedSchoolIDs = NormalizeStringList(snapshot.AllowedSchoolIDs)
	if snapshot.PreviewTitleRunes <= 0 {
		snapshot.PreviewTitleRunes = defaultReviewPreviewTitleRunes
	}
	if snapshot.PreviewContentRunes <= 0 {
		snapshot.PreviewContentRunes = defaultReviewPreviewContentRunes
	}
	if snapshot.PreviewContentPct <= 0 {
		snapshot.PreviewContentPct = defaultReviewPreviewContentPct
	}
	return snapshot
}

func ParseStringList(value string) ([]string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var raw []string
		if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
			return nil, fmt.Errorf("must be a JSON string array or comma-separated list")
		}
		return NormalizeStringList(raw), nil
	}
	if strings.HasPrefix(trimmed, "{") {
		return nil, fmt.Errorf("must be a JSON string array or comma-separated list")
	}

	parts := strings.Split(trimmed, ",")
	return NormalizeStringList(parts), nil
}

func ParseBoundedInt(value string, minValue, maxValue int) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("value must not be empty")
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("must be an integer")
	}
	if parsed < minValue || parsed > maxValue {
		return 0, fmt.Errorf("must be between %d and %d", minValue, maxValue)
	}
	return parsed, nil
}

func NormalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
