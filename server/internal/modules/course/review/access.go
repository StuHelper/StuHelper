package review

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

const (
	reviewAccessSchoolID    = "10006"
	reviewPreviewTitleRunes = 24
	reviewPreviewTextRunes  = 120
)

type ReviewAccessFacts struct {
	Authenticated    bool
	CanManageReviews bool
	CanViewFull      bool
	CanPostReview    bool
	IdentityVerified bool
	StudentVerified  bool
	SchoolID         *string
}

func (h *Handler) resolveReviewAccessFacts(ctx context.Context, externalID string) (ReviewAccessFacts, error) {
	if externalID == "" {
		return ReviewAccessFacts{}, nil
	}

	facts := ReviewAccessFacts{Authenticated: true}

	if h.permissionSvc == nil {
		return facts, nil
	}

	internalUserID, err := h.permissionSvc.GetInternalUserID(ctx, externalID)
	if err != nil {
		return facts, err
	}

	canManageReviews, err := h.permissionSvc.CheckPermission(ctx, internalUserID, capability.AdminReviewsManage, nil)
	if err != nil {
		return facts, err
	}
	facts.CanManageReviews = canManageReviews

	profile, err := h.userRepo.GetProfileByUserID(ctx, internalUserID)
	if err != nil {
		return facts, err
	}
	identity, err := h.userRepo.GetIdentityStatusByUserID(ctx, internalUserID)
	if err != nil {
		return facts, err
	}

	if profile != nil {
		facts.SchoolID = profile.SchoolID
		facts.StudentVerified = profile.VerificationStatus == user.StatusVerified &&
			profile.SchoolID != nil &&
			*profile.SchoolID == reviewAccessSchoolID
	}
	facts.IdentityVerified = identity != nil && identity.Verified
	facts.CanViewFull = facts.CanManageReviews || facts.StudentVerified
	facts.CanPostReview = facts.StudentVerified && facts.IdentityVerified

	return facts, nil
}

func (h *Handler) resolveReviewAccessFactsForRequest(c *gin.Context) ReviewAccessFacts {
	externalID := middleware.GetUserID(c)
	facts, err := h.resolveReviewAccessFacts(c.Request.Context(), externalID)
	if err == nil {
		return facts
	}

	logger.FromGin(c).Warn("failed to resolve review access facts", zap.Error(err))
	return ReviewAccessFacts{Authenticated: externalID != ""}
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
			result[i].Title = previewText(result[i].Title, reviewPreviewTitleRunes)
			result[i].Content = previewText(result[i].Content, reviewPreviewTextRunes)
		}
	}

	return result
}

func previewText(value string, maxRunes int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return string(runes[:maxRunes]) + "..."
}
