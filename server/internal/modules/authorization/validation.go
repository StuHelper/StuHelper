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
	if input.Source == "" {
		input.Source = GrantSourceManual
	}
	if input.Source != GrantSourceManual {
		return CreateGrantInput{}, ErrInvalidGrant
	}

	switch input.Role {
	case RoleSuperAdmin:
		return CreateGrantInput{}, ErrProviderManagedRole
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

func normalizeReconcileGrantInput(input ReconcileGrantInput) (ReconcileGrantInput, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if input.GrantID <= 0 || input.ActorUserID <= 0 {
		return ReconcileGrantInput{}, ErrInvalidGrant
	}
	if err := validateReason(input.Reason); err != nil {
		return ReconcileGrantInput{}, err
	}
	return input, nil
}

func normalizeReconcileAllInput(input ReconcileAllInput) (ReconcileAllInput, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if input.ActorUserID <= 0 {
		return ReconcileAllInput{}, ErrInvalidGrant
	}
	if err := validateReason(input.Reason); err != nil {
		return ReconcileAllInput{}, err
	}
	return input, nil
}

func normalizeCasdoorOrganizationAdminSyncInput(
	input CasdoorOrganizationAdminSyncInput,
) (CasdoorOrganizationAdminSyncInput, error) {
	if input.SubjectUserID <= 0 {
		return CasdoorOrganizationAdminSyncInput{}, ErrInvalidGrant
	}
	return input, nil
}

func normalizeListGrantsFilter(filter ListGrantsFilter) (ListGrantsFilter, error) {
	if filter.SubjectUserID != nil && *filter.SubjectUserID <= 0 {
		return ListGrantsFilter{}, ErrInvalidGrant
	}
	if filter.Role != nil && !isSupportedRole(*filter.Role) {
		return ListGrantsFilter{}, ErrInvalidGrant
	}
	if filter.DesiredState != nil &&
		*filter.DesiredState != DesiredGranted &&
		*filter.DesiredState != DesiredRevoked {
		return ListGrantsFilter{}, ErrInvalidGrant
	}
	if filter.Projection != nil &&
		*filter.Projection != ProjectionPending &&
		*filter.Projection != ProjectionApplied &&
		*filter.Projection != ProjectionFailed {
		return ListGrantsFilter{}, ErrInvalidGrant
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		return ListGrantsFilter{}, ErrInvalidGrant
	}
	return filter, nil
}

func isSupportedRole(role Role) bool {
	switch role {
	case RoleSuperAdmin, RoleSchoolAdmin, RoleSectionAdmin, RoleSectionModerator, RoleSectionReviewer:
		return true
	default:
		return false
	}
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
