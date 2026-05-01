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

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/reviewaccess"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/singleflightx"
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
	CanEditOwn          bool
	CanDeleteOwn        bool
	IdentityVerified    bool
	StudentVerified     bool
	SchoolID            *string
	PreviewTitleRunes   int
	PreviewContentRunes int
	PreviewContentPct   int
}

func (s *Service) getReviewAccessPolicy(ctx context.Context) (systemconfig.ReviewAccessPolicySnapshot, error) {
	cached := s.readReviewAccessPolicy()
	if policyFresh(cached) {
		return cached, nil
	}

	policy, err := singleflightx.DoValue(&s.accessPolicySF, "access-policy", func() (systemconfig.ReviewAccessPolicySnapshot, error) {
		cached := s.readReviewAccessPolicy()
		if policyFresh(cached) {
			return cached, nil
		}

		refreshCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		policy, err := s.loadReviewAccessPolicy(refreshCtx)
		if err != nil {
			return systemconfig.ReviewAccessPolicySnapshot{}, err
		}

		systemconfig.SetReviewAccessPolicySnapshot(policy)
		return policy, nil
	})
	if err != nil {
		return systemconfig.ReviewAccessPolicySnapshot{}, err
	}
	return policy, nil
}

func (s *Service) readReviewAccessPolicy() systemconfig.ReviewAccessPolicySnapshot {
	return systemconfig.GetReviewAccessPolicySnapshot()
}

func policyFresh(policy systemconfig.ReviewAccessPolicySnapshot) bool {
	if policy.LoadedAt.IsZero() {
		return false
	}
	return time.Since(policy.LoadedAt) <= reviewAccessPolicyTTL
}

func (s *Service) loadReviewAccessPolicy(ctx context.Context) (systemconfig.ReviewAccessPolicySnapshot, error) {
	schools, err := s.accessReader.ListReviewAccessSchoolConfigs(ctx)
	if err != nil {
		return systemconfig.ReviewAccessPolicySnapshot{}, fmt.Errorf("load enabled schools: %w", err)
	}
	configs, err := s.accessReader.ListReviewAccessSystemConfigs(ctx)
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

func buildReviewAccessPolicy(schools []reviewaccess.SchoolConfig, configs []reviewaccess.SystemConfig) (systemconfig.ReviewAccessPolicySnapshot, error) {
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

// ResolveAccessFacts 解析当前用户的评课访问事实。
// 由 Service 统一生产，Handler 不再自行拼装。
func (s *Service) ResolveAccessFacts(ctx context.Context, casdoorSubject string, capabilities []string) (ReviewAccessFacts, error) {
	policy, err := s.getReviewAccessPolicy(ctx)
	if err != nil {
		return ReviewAccessFacts{}, err
	}

	facts := ReviewAccessFacts{
		PreviewTitleRunes:   policy.PreviewTitleRunes,
		PreviewContentRunes: policy.PreviewContentRunes,
		PreviewContentPct:   policy.PreviewContentPct,
	}
	if casdoorSubject == "" {
		return facts, nil
	}

	facts.Authenticated = true

	// 从 Token 角色展开的能力中检查管理权限（零 DB 查询）
	facts.CanManageReviews = capability.Has(capabilities, capability.AdminReviewsManage)
	canViewFull := capability.Has(capabilities, capability.ReviewListFull)
	canCreate := capability.Has(capabilities, capability.ReviewCreate)
	canEditOwn := capability.Has(capabilities, capability.ReviewEditOwn)
	canDeleteOwn := capability.Has(capabilities, capability.ReviewDeleteOwn)

	subject, err := s.accessReader.GetReviewAccessSubject(ctx, casdoorSubject)
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
	facts.CanViewFull = facts.CanManageReviews || (canViewFull && facts.StudentVerified)
	facts.CanPostReview = canCreate && facts.StudentVerified && facts.IdentityVerified
	facts.CanEditOwn = canEditOwn && facts.StudentVerified && facts.IdentityVerified
	facts.CanDeleteOwn = canDeleteOwn && facts.StudentVerified && facts.IdentityVerified

	return facts, nil
}

func (h *Handler) resolveReviewAccessFactsForRequest(c *gin.Context) (ReviewAccessFacts, bool) {
	casdoorSubject := middleware.GetUserID(c)
	capabilities := middleware.GetCapabilities(c)
	facts, err := h.service.ResolveAccessFacts(c.Request.Context(), casdoorSubject, capabilities)
	if err == nil {
		return facts, true
	}

	logger.FromGin(c).Warn("failed to resolve review access facts", zap.Error(err))
	response.ServiceUnavailable(c, "review access policy temporarily unavailable")
	return ReviewAccessFacts{}, false
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
