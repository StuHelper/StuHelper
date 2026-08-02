package review

import (
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
)

type moderationScope struct {
	global            bool
	schoolAdmins      map[int64]struct{}
	moderatorSections map[string]struct{}
}

func resolveModerationScope(c *gin.Context, capabilityName string) moderationScope {
	return moderationScopeFromCapabilityGrants(middleware.GetCapabilityGrants(c), capabilityName)
}

func moderationScopeFromCapabilityGrants(grants []capability.Grant, capabilityName string) moderationScope {
	scope := moderationScope{
		schoolAdmins:      make(map[int64]struct{}),
		moderatorSections: make(map[string]struct{}),
	}
	for _, grant := range grants {
		if grant.Name != capabilityName {
			continue
		}
		if grant.Global {
			scope.global = true
			break
		}
		for _, rawSchoolID := range grant.ScopeSchoolIDs {
			schoolID, err := strconv.ParseInt(rawSchoolID, 10, 64)
			if err == nil && schoolID > 0 {
				scope.schoolAdmins[schoolID] = struct{}{}
			}
		}
		for _, sectionID := range grant.ScopeSectionIDs {
			if _, ok := schoolIDFromReviewModerationSectionID(sectionID); ok {
				scope.moderatorSections[sectionID] = struct{}{}
			}
		}
	}
	if scope.global {
		scope.schoolAdmins = nil
		scope.moderatorSections = nil
	} else {
		if len(scope.schoolAdmins) == 0 {
			scope.schoolAdmins = nil
		}
		if len(scope.moderatorSections) == 0 {
			scope.moderatorSections = nil
		}
	}
	return scope
}

func (s moderationScope) schoolIDs() []int64 {
	if s.global {
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
