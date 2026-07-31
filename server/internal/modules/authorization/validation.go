package authorization

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
)

func normalizeCreateGrantInput(input CreateGrantInput) (CreateGrantInput, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if input.SubjectUserID <= 0 || input.ActorUserID <= 0 {
		return CreateGrantInput{}, ErrInvalidGrant
	}
	if err := validateReason(input.Reason); err != nil {
		return CreateGrantInput{}, err
	}

	switch input.Role {
	case RoleSuperAdmin:
		if input.SchoolID != nil || input.SectionID != nil {
			return CreateGrantInput{}, fmt.Errorf("%w: super_admin must be global", ErrInvalidGrant)
		}
	case RoleSchoolAdmin:
		if !validPositiveID(input.SchoolID) || input.SectionID != nil {
			return CreateGrantInput{}, fmt.Errorf("%w: school_admin requires exactly one school", ErrInvalidGrant)
		}
	case RoleSectionAdmin, RoleSectionModerator, RoleSectionReviewer:
		if !validPositiveID(input.SchoolID) || input.SectionID == nil {
			return CreateGrantInput{}, fmt.Errorf("%w: section role requires school and section", ErrInvalidGrant)
		}
		sectionID := strings.TrimSpace(*input.SectionID)
		schoolID, ok := fga.ParseReviewModerationSectionID(sectionID)
		if !ok || schoolID != strconv.FormatInt(*input.SchoolID, 10) {
			return CreateGrantInput{}, fmt.Errorf("%w: unsupported or mismatched section", ErrInvalidGrant)
		}
		input.SectionID = &sectionID
	default:
		return CreateGrantInput{}, fmt.Errorf("%w: unsupported role", ErrInvalidGrant)
	}
	return input, nil
}

func normalizeRevokeGrantInput(input RevokeGrantInput) (RevokeGrantInput, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if input.GrantID <= 0 || input.ActorUserID <= 0 {
		return RevokeGrantInput{}, ErrInvalidGrant
	}
	if err := validateReason(input.Reason); err != nil {
		return RevokeGrantInput{}, err
	}
	return input, nil
}

func validateReason(reason string) error {
	if reason == "" || len([]rune(reason)) > maxReasonLength {
		return fmt.Errorf("%w: reason must contain 1-%d characters", ErrInvalidGrant, maxReasonLength)
	}
	return nil
}

func validPositiveID(value *int64) bool {
	return value != nil && *value > 0
}

func projectionDedupeKey(grantID int64) string {
	return "authorization-grant:" + strconv.FormatInt(grantID, 10)
}
