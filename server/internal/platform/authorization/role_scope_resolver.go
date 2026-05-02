package authorization

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	roleSchoolAdmin            = "school_admin"
	openFGASchoolType          = "school"
	openFGAUserType            = "user"
	openFGASchoolAdminRelation = "effective_admin"
)

type ObjectLister interface {
	ListObjects(ctx context.Context, user, relation, objectType string) ([]string, error)
}

type InternalUserIDResolver func(ctx context.Context, casdoorSubject string) (int64, error)

type RoleScopeResolver struct {
	fga            ObjectLister
	internalUserID InternalUserIDResolver
}

func NewRoleScopeResolver(fga ObjectLister, internalUserID InternalUserIDResolver) (*RoleScopeResolver, error) {
	if fga == nil {
		return nil, errors.New("authorization: fga object lister is required")
	}
	if internalUserID == nil {
		return nil, errors.New("authorization: internal user id resolver is required")
	}
	return &RoleScopeResolver{fga: fga, internalUserID: internalUserID}, nil
}

func (r *RoleScopeResolver) ResolveRoleScopes(
	ctx context.Context,
	casdoorSubject string,
	roles []string,
) (map[string][]string, error) {
	if !containsRole(roles, roleSchoolAdmin) {
		return nil, nil
	}
	userID, err := r.internalUserID(ctx, casdoorSubject)
	if err != nil {
		return nil, fmt.Errorf("resolve internal user id: %w", err)
	}
	objects, err := r.fga.ListObjects(ctx, fgaUser(userID), openFGASchoolAdminRelation, openFGASchoolType)
	if err != nil {
		return nil, fmt.Errorf("list school admin scopes: %w", err)
	}
	schoolIDs := objectIDs(objects, openFGASchoolType)
	if len(schoolIDs) == 0 {
		return nil, nil
	}
	return map[string][]string{roleSchoolAdmin: schoolIDs}, nil
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
