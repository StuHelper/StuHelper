package review

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/systemconfig"
)

const (
	reviewAccessPolicyTTL = 5 * time.Minute
)

type ReviewAccessFacts struct {
	Authenticated       bool
	CanManageReviews    bool
	CanViewFull         bool
	CanPostReview       bool
	IdentityVerified    bool
	StudentVerified     bool
	SchoolID            *string
	PreviewTitleRunes   int
	PreviewContentRunes int
	PreviewContentPct   int
}

func (h *Handler) getReviewAccessPolicy(ctx context.Context) (systemconfig.ReviewAccessPolicySnapshot, error) {
	cached := h.readReviewAccessPolicy()
	if policyFresh(cached) {
		return cached, nil
	}

	result, err, _ := h.accessPolicySF.Do("access-policy", func() (interface{}, error) {
		cached := h.readReviewAccessPolicy()
		if policyFresh(cached) {
			return cached, nil
		}

		refreshCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		policy, err := h.loadReviewAccessPolicy(refreshCtx)
		if err != nil {
			if !cached.LoadedAt.IsZero() {
				logger.L().Warn("failed to refresh review access policy, using stale policy", zap.Error(err))
				return cached, nil
			}
			return systemconfig.ReviewAccessPolicySnapshot{}, err
		}

		systemconfig.SetReviewAccessPolicySnapshot(policy)
		return policy, nil
	})
	if err != nil {
		return systemconfig.ReviewAccessPolicySnapshot{}, err
	}
	policy, ok := result.(systemconfig.ReviewAccessPolicySnapshot)
	if !ok {
		return systemconfig.ReviewAccessPolicySnapshot{}, fmt.Errorf("unexpected access policy result type %T", result)
	}
	return policy, nil
}

func (h *Handler) readReviewAccessPolicy() systemconfig.ReviewAccessPolicySnapshot {
	return systemconfig.GetReviewAccessPolicySnapshot()
}

func policyFresh(policy systemconfig.ReviewAccessPolicySnapshot) bool {
	if policy.LoadedAt.IsZero() {
		return false
	}
	return time.Since(policy.LoadedAt) <= reviewAccessPolicyTTL
}

func (h *Handler) loadReviewAccessPolicy(ctx context.Context) (systemconfig.ReviewAccessPolicySnapshot, error) {
	schools, err := h.userRepo.ListSchoolConfigs(ctx)
	if err != nil {
		return systemconfig.ReviewAccessPolicySnapshot{}, fmt.Errorf("load enabled schools: %w", err)
	}
	configs, err := h.userRepo.ListSystemConfigs(ctx)
	if err != nil {
		return systemconfig.ReviewAccessPolicySnapshot{}, fmt.Errorf("load system configs: %w", err)
	}

	policy, err := buildReviewAccessPolicy(schools, configs)
	if err != nil {
		return systemconfig.ReviewAccessPolicySnapshot{}, err
	}
	policy.LoadedAt = time.Now()
	return policy, nil
}

func buildReviewAccessPolicy(schools []user.SchoolConfig, configs []user.SystemConfig) (systemconfig.ReviewAccessPolicySnapshot, error) {
	policy := systemconfig.DefaultReviewAccessPolicySnapshot()

	enabledSchoolIDs := make([]string, 0, len(schools))
	for _, school := range schools {
		if school.SchoolID != 0 {
			enabledSchoolIDs = append(enabledSchoolIDs, strconv.FormatInt(school.SchoolID, 10))
		}
	}
	configMap := make(map[string]string, len(configs))
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	allowedSchoolIDs, err := parseReviewAccessSchoolIDs(configMap, enabledSchoolIDs)
	if err != nil {
		return systemconfig.ReviewAccessPolicySnapshot{}, err
	}
	policy.AllowedSchoolIDs = allowedSchoolIDs

	if value, ok := firstNonEmptyConfig(configMap, systemconfig.ReviewPreviewTitleCharsKey); ok {
		policy.PreviewTitleRunes, err = systemconfig.ParseBoundedInt(value, 1, 200)
		if err != nil {
			return systemconfig.ReviewAccessPolicySnapshot{}, fmt.Errorf("%s %w", systemconfig.ReviewPreviewTitleCharsKey, err)
		}
	}
	if value, ok := firstNonEmptyConfig(configMap, systemconfig.ReviewPreviewContentCharsKey); ok {
		policy.PreviewContentRunes, err = systemconfig.ParseBoundedInt(value, 1, 5000)
		if err != nil {
			return systemconfig.ReviewAccessPolicySnapshot{}, fmt.Errorf("%s %w", systemconfig.ReviewPreviewContentCharsKey, err)
		}
	}
	if value, ok := firstNonEmptyConfig(configMap, systemconfig.ReviewPreviewContentPercentKey); ok {
		policy.PreviewContentPct, err = systemconfig.ParseBoundedInt(value, 1, 100)
		if err != nil {
			return systemconfig.ReviewAccessPolicySnapshot{}, fmt.Errorf("%s %w", systemconfig.ReviewPreviewContentPercentKey, err)
		}
	}

	return policy, nil
}

func parseReviewAccessSchoolIDs(configs map[string]string, enabledSchoolIDs []string) ([]string, error) {
	if raw, ok := firstNonEmptyConfig(configs, systemconfig.ReviewAccessSchoolIDsKey); ok {
		return systemconfig.ParseStringList(raw)
	}
	return systemconfig.NormalizeStringList(enabledSchoolIDs), nil
}

func firstNonEmptyConfig(configs map[string]string, keys ...string) (string, bool) {
	for _, key := range keys {
		value := strings.TrimSpace(configs[key])
		if value == "" {
			continue
		}
		return value, true
	}
	return "", false
}

func (h *Handler) resolveReviewAccessFacts(ctx context.Context, externalID string, capabilities []string) (ReviewAccessFacts, error) {
	policy, err := h.getReviewAccessPolicy(ctx)
	if err != nil {
		return ReviewAccessFacts{}, err
	}

	facts := ReviewAccessFacts{
		PreviewTitleRunes:   policy.PreviewTitleRunes,
		PreviewContentRunes: policy.PreviewContentRunes,
		PreviewContentPct:   policy.PreviewContentPct,
	}
	if externalID == "" {
		return facts, nil
	}

	facts.Authenticated = true

	// 从 Token 角色展开的能力中检查管理权限（零 DB 查询）
	facts.CanManageReviews = capability.Has(capabilities, capability.AdminReviewsManage)

	subject, err := h.userRepo.GetReviewAccessSubjectByExternalID(ctx, externalID)
	if err != nil {
		return facts, err
	}
	if subject == nil {
		return facts, nil
	}

	facts.SchoolID = int64PtrToStringPtr(subject.SchoolID)
	facts.StudentVerified = subject.StudentVerified &&
		subject.SchoolID != nil &&
		policy.AllowsSchool(strconv.FormatInt(*subject.SchoolID, 10))
	facts.IdentityVerified = subject.IdentityVerified
	facts.CanViewFull = facts.CanManageReviews || facts.StudentVerified
	facts.CanPostReview = facts.StudentVerified && facts.IdentityVerified

	return facts, nil
}

func (h *Handler) resolveReviewAccessFactsForRequest(c *gin.Context) ReviewAccessFacts {
	externalID := middleware.GetUserID(c)
	capabilities := middleware.GetCapabilities(c)
	facts, err := h.resolveReviewAccessFacts(c.Request.Context(), externalID, capabilities)
	if err == nil {
		return facts
	}

	logger.FromGin(c).Warn("failed to resolve review access facts", zap.Error(err))
	policy := systemconfig.DefaultReviewAccessPolicySnapshot()
	return ReviewAccessFacts{
		Authenticated:       externalID != "",
		PreviewTitleRunes:   policy.PreviewTitleRunes,
		PreviewContentRunes: policy.PreviewContentRunes,
		PreviewContentPct:   policy.PreviewContentPct,
	}
}

func stripReviewsForResponse(reviews []Review, facts ReviewAccessFacts) []Review {
	result := make([]Review, len(reviews))
	copy(result, reviews)

	for i := range result {
		switch {
		case result[i].Status == StatusHidden && !facts.CanManageReviews:
			result[i].Content = ""
			result[i].Title = ""
		case !facts.Authenticated:
			result[i].Content = ""
			result[i].Title = ""
		case !facts.CanViewFull:
			result[i].Title = previewText(result[i].Title, facts.PreviewTitleRunes, 100)
			result[i].Content = previewText(result[i].Content, facts.PreviewContentRunes, facts.PreviewContentPct)
		}
	}

	return result
}

func previewText(value string, maxRunes int, percent int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	limit := maxRunes
	if percent > 0 && percent < 100 {
		percentLimit := int(math.Ceil(float64(len(runes)) * float64(percent) / 100))
		if percentLimit < 1 {
			percentLimit = 1
		}
		if percentLimit < limit {
			limit = percentLimit
		}
	}
	if limit <= 0 || len(runes) <= limit {
		return trimmed
	}
	return string(runes[:limit]) + "..."
}

func int64PtrToStringPtr(v *int64) *string {
	if v == nil {
		return nil
	}
	s := strconv.FormatInt(*v, 10)
	return &s
}
