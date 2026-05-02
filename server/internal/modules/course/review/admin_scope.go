package review

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/errs"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

const (
	roleSuperAdmin       = "super_admin"
	roleSchoolAdmin      = "school_admin"
	roleSectionAdmin     = "section_admin"
	roleSectionModerator = "section_moderator"
)

type moderationScope struct {
	superAdmin        bool
	schoolAdmins      map[int64]struct{}
	moderatorSections map[string]struct{}
}

func requireModerationRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !resolveModerationScope(c).hasModerationAccess() {
			response.Forbidden(c, "insufficient permissions", errs.ErrAccessDenied)
			c.Abort()
			return
		}
		c.Next()
	}
}

func requireSchoolAdminRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !resolveModerationScope(c).hasContentEditAccess() {
			response.Forbidden(c, "insufficient permissions", errs.ErrAccessDenied)
			c.Abort()
			return
		}
		c.Next()
	}
}

func resolveModerationScope(c *gin.Context) moderationScope {
	scope := moderationScope{
		superAdmin:        hasRole(middleware.GetRoles(c), roleSuperAdmin),
		schoolAdmins:      scopedSchoolSet(c, roleSchoolAdmin),
		moderatorSections: scopedModerationSections(c),
	}
	if scope.superAdmin {
		scope.schoolAdmins = nil
		scope.moderatorSections = nil
	}
	return scope
}

func scopedModerationSections(c *gin.Context) map[string]struct{} {
	roles := []string{roleSectionAdmin, roleSectionModerator}
	merged := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		for sectionID := range scopedStringSet(c, role) {
			merged[sectionID] = struct{}{}
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func scopedSchoolSet(c *gin.Context, role string) map[int64]struct{} {
	rawValues := scopedStringSet(c, role)
	if len(rawValues) == 0 {
		return nil
	}
	schoolSet := make(map[int64]struct{}, len(rawValues))
	for rawSchoolID := range rawValues {
		schoolID, err := strconv.ParseInt(rawSchoolID, 10, 64)
		if err != nil {
			continue
		}
		schoolSet[schoolID] = struct{}{}
	}
	if len(schoolSet) == 0 {
		return nil
	}
	return schoolSet
}

func scopedStringSet(c *gin.Context, role string) map[string]struct{} {
	value, exists := c.Get(middleware.CtxKeyOrgScopedRoles)
	if !exists {
		return nil
	}
	scopedRoles, ok := value.(map[string][]string)
	if !ok {
		return nil
	}
	rawSchoolIDs := scopedRoles[role]
	if len(rawSchoolIDs) == 0 {
		return nil
	}

	scopeSet := make(map[string]struct{}, len(rawSchoolIDs))
	for _, rawID := range rawSchoolIDs {
		if rawID == "" {
			continue
		}
		scopeSet[rawID] = struct{}{}
	}
	if len(scopeSet) == 0 {
		return nil
	}
	return scopeSet
}

func hasRole(roles []string, target string) bool {
	for _, role := range roles {
		if role == target {
			return true
		}
	}
	return false
}

func (s moderationScope) hasModerationAccess() bool {
	return s.superAdmin || len(s.schoolIDs()) > 0
}

func (s moderationScope) hasContentEditAccess() bool {
	return s.superAdmin || len(s.schoolAdmins) > 0
}

func (s moderationScope) canModerateSchool(schoolID int64) bool {
	if s.superAdmin {
		return true
	}
	_, schoolAdmin := s.schoolAdmins[schoolID]
	_, moderator := s.moderatorSections[reviewModerationSectionID(schoolID)]
	return schoolAdmin || moderator
}

func (s moderationScope) canEditSchool(schoolID int64) bool {
	if s.superAdmin {
		return true
	}
	_, schoolAdmin := s.schoolAdmins[schoolID]
	return schoolAdmin
}

func (s moderationScope) schoolIDs() []int64 {
	if s.superAdmin {
		return nil
	}
	seen := make(map[int64]struct{}, len(s.schoolAdmins)+len(s.moderatorSections))
	for schoolID := range s.schoolAdmins {
		seen[schoolID] = struct{}{}
	}
	for sectionID := range s.moderatorSections {
		schoolID, ok := schoolIDFromReviewModerationSectionID(sectionID)
		if ok {
			seen[schoolID] = struct{}{}
		}
	}

	ids := make([]int64, 0, len(seen))
	for schoolID := range seen {
		ids = append(ids, schoolID)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func reviewModerationSectionID(schoolID int64) string {
	return fga.ReviewModerationSectionID(strconv.FormatInt(schoolID, 10))
}

func schoolIDFromReviewModerationSectionID(sectionID string) (int64, bool) {
	const prefix = "school_"
	const suffix = "_review_moderation"
	raw, ok := strings.CutPrefix(sectionID, prefix)
	if !ok {
		return 0, false
	}
	raw, ok = strings.CutSuffix(raw, suffix)
	if !ok {
		return 0, false
	}
	schoolID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || schoolID <= 0 {
		return 0, false
	}
	return schoolID, true
}

func (h *Handler) authorizeReviewModeration(c *gin.Context, reviewID string) bool {
	if h.service == nil {
		response.Forbidden(c, "insufficient permission for this review", errs.ErrAccessDenied)
		return false
	}
	scope := resolveModerationScope(c)
	schoolID, err := h.service.GetReviewSchoolID(c.Request.Context(), reviewID)
	if err != nil {
		return respondModerationLookupError(c, err, "review not found", "failed to resolve review scope")
	}
	if !scope.canModerateSchool(schoolID) {
		response.Forbidden(c, "insufficient permission for this review", errs.ErrAccessDenied)
		return false
	}
	return true
}

func (h *Handler) authorizeReviewContentEdit(c *gin.Context, reviewID string) bool {
	if h.service == nil {
		response.Forbidden(c, "insufficient permission for this review", errs.ErrAccessDenied)
		return false
	}
	scope := resolveModerationScope(c)
	schoolID, err := h.service.GetReviewSchoolID(c.Request.Context(), reviewID)
	if err != nil {
		return respondModerationLookupError(c, err, "review not found", "failed to resolve review scope")
	}
	if !scope.canEditSchool(schoolID) {
		response.Forbidden(c, "insufficient permission for this review", errs.ErrAccessDenied)
		return false
	}
	return true
}

func (h *Handler) authorizeBatchReviewModeration(c *gin.Context, reviewIDs []string) bool {
	if h.service == nil {
		response.Forbidden(c, "insufficient permission for one or more reviews", errs.ErrAccessDenied)
		return false
	}
	scope := resolveModerationScope(c)
	schoolIDs, err := h.service.ListReviewSchoolIDs(c.Request.Context(), reviewIDs)
	if err != nil {
		return respondModerationLookupError(c, err, "review not found", "failed to resolve review scope")
	}
	for _, reviewID := range reviewIDs {
		schoolID, ok := schoolIDs[reviewID]
		if !ok {
			response.NotFound(c, "review not found")
			return false
		}
		if !scope.canModerateSchool(schoolID) {
			response.Forbidden(c, "insufficient permission for one or more reviews", errs.ErrAccessDenied)
			return false
		}
	}
	return true
}

func (h *Handler) authorizeReportModeration(c *gin.Context, reportID string) bool {
	if h.service == nil {
		response.Forbidden(c, "insufficient permission for this report", errs.ErrAccessDenied)
		return false
	}
	scope := resolveModerationScope(c)
	schoolID, err := h.service.GetReportSchoolID(c.Request.Context(), reportID)
	if err != nil {
		return respondModerationLookupError(c, err, "report not found", "failed to resolve report scope")
	}
	if !scope.canModerateSchool(schoolID) {
		response.Forbidden(c, "insufficient permission for this report", errs.ErrAccessDenied)
		return false
	}
	return true
}

func respondModerationLookupError(c *gin.Context, err error, notFoundMessage, internalMessage string) bool {
	switch {
	case errors.Is(err, ErrReviewNotFound), errors.Is(err, ErrReportNotFound), errors.Is(err, pgx.ErrNoRows):
		response.NotFound(c, notFoundMessage)
	default:
		response.InternalError(c, internalMessage)
	}
	return false
}
