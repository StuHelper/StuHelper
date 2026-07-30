package authorization

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
)

const (
	roleSchoolAdmin            = "school_admin"
	roleSectionAdmin           = "section_admin"
	roleSectionModerator       = "section_moderator"
	openFGASchoolType          = "school"
	openFGASectionType         = "section"
	openFGAUserType            = "user"
	openFGASchoolAdminRelation = "effective_admin"
)

type ScopeReader interface {
	ListObjects(ctx context.Context, user, relation, objectType string) ([]string, error)
}

type InternalUserIDResolver func(ctx context.Context, casdoorSubject string) (int64, error)

type RoleScopeResolver struct {
	scopeReader    ScopeReader
	internalUserID InternalUserIDResolver
}

func NewRoleScopeResolver(scopeReader ScopeReader, internalUserID InternalUserIDResolver) (*RoleScopeResolver, error) {
	if scopeReader == nil {
		return nil, errors.New("authorization: fga scope reader is required")
	}
	if internalUserID == nil {
		return nil, errors.New("authorization: internal user id resolver is required")
	}
	return &RoleScopeResolver{scopeReader: scopeReader, internalUserID: internalUserID}, nil
}

func (r *RoleScopeResolver) ResolveRoleScopes(
	ctx context.Context,
	casdoorSubject string,
	roles []string,
) (map[string][]string, error) {
	if !needsResolvedScopes(roles) {
		return nil, nil
	}
	userID, err := r.internalUserID(ctx, casdoorSubject)
	if err != nil {
		return nil, fmt.Errorf("resolve internal user id: %w", err)
	}
	subject := fgaUser(userID)
	scopes := map[string][]string{}
	if err := r.resolveSchoolAdminScopes(ctx, subject, roles, scopes); err != nil {
		return nil, err
	}
	if err := r.resolveSectionRoleScopes(ctx, subject, roleSectionAdmin, roles, scopes); err != nil {
		return nil, err
	}
	if err := r.resolveSectionRoleScopes(ctx, subject, roleSectionModerator, roles, scopes); err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		return nil, nil
	}
	return scopes, nil
}

func needsResolvedScopes(roles []string) bool {
	return containsRole(roles, roleSchoolAdmin) ||
		containsRole(roles, roleSectionAdmin) ||
		containsRole(roles, roleSectionModerator)
}

func (r *RoleScopeResolver) resolveSchoolAdminScopes(
	ctx context.Context,
	subject string,
	roles []string,
	scopes map[string][]string,
) error {
	if !containsRole(roles, roleSchoolAdmin) {
		return nil
	}
	objects, err := r.scopeReader.ListObjects(ctx, subject, openFGASchoolAdminRelation, openFGASchoolType)
	if err != nil {
		return fmt.Errorf("list school admin scopes: %w", err)
	}
	schoolIDs := objectIDs(objects, openFGASchoolType)
	if len(schoolIDs) > 0 {
		scopes[roleSchoolAdmin] = schoolIDs
	}
	return nil
}

func (r *RoleScopeResolver) resolveSectionRoleScopes(
	ctx context.Context,
	subject string,
	role string,
	roles []string,
	scopes map[string][]string,
) error {
	if !containsRole(roles, role) {
		return nil
	}
	sections, err := r.scopeReader.ListObjects(ctx, subject, role, openFGASectionType)
	if err != nil {
		return fmt.Errorf("list %s scopes: %w", role, err)
	}
	sectionIDs := objectIDs(sections, openFGASectionType)
	sectionIDs = validReviewModerationSections(subject, role, sectionIDs)
	if len(sectionIDs) > 0 {
		scopes[role] = sectionIDs
	}
	return nil
}

func validReviewModerationSections(subject, role string, sectionIDs []string) []string {
	valid := make([]string, 0, len(sectionIDs))
	for _, sectionID := range sectionIDs {
		if _, ok := fga.ParseReviewModerationSectionID(sectionID); !ok {
			metrics.ObserveIAMInvalidRoleScope()
			logger.L().Warn(
				"ignoring invalid OpenFGA role scope",
				zap.String("fga_user", subject),
				zap.String("role", role),
				zap.String("section_id", sectionID),
			)
			continue
		}
		valid = append(valid, sectionID)
	}
	return valid
}

func fgaUser(userID int64) string {
	return openFGAUserType + ":" + strconv.FormatInt(userID, 10)
}

func containsRole(roles []string, expected string) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}

func objectIDs(objects []string, objectType string) []string {
	prefix := objectType + ":"
	seen := make(map[string]struct{}, len(objects))
	result := make([]string, 0, len(objects))
	for _, object := range objects {
		id, ok := strings.CutPrefix(object, prefix)
		if !ok || id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
