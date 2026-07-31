package review

import (
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
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
		if !hasAnyModerationRole(middleware.GetRoles(c)) {
			response.Forbidden(c, "insufficient permissions", errs.ErrAccessDenied)
			c.Abort()
			return
		}
		c.Next()
	}
}

func requireSchoolAdminRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !hasAnyContentEditRole(middleware.GetRoles(c)) {
			response.Forbidden(c, "insufficient permissions", errs.ErrAccessDenied)
			c.Abort()
			return
		}
		c.Next()
	}
}

func hasAnyModerationRole(roles []string) bool {
	return hasRole(roles, roleSuperAdmin) ||
		hasRole(roles, roleSchoolAdmin) ||
		hasRole(roles, roleSectionAdmin) ||
		hasRole(roles, roleSectionModerator)
}

func hasAnyContentEditRole(roles []string) bool {
	return hasRole(roles, roleSuperAdmin) || hasRole(roles, roleSchoolAdmin)
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
	scopedRoles := middleware.GetScopedRoleGrants(c)
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
	raw, ok := fga.ParseReviewModerationSectionID(sectionID)
	if !ok {
		return 0, false
	}
	schoolID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || schoolID <= 0 {
		return 0, false
	}
	return schoolID, true
}
