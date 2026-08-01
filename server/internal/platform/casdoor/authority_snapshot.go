package casdoor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

type AuthoritySnapshotUser struct {
	ID                 string
	Owner              string
	Name               string
	OrganizationAdmin  bool
	ForbiddenOrDeleted bool
}

type AuthoritySnapshot struct {
	Organization string
	Users        []AuthoritySnapshotUser
	RoleMembers  map[string][]string
}

type AuthoritySnapshotClient struct {
	credential Credential
	api        authoritySnapshotAPI
}

type authoritySnapshotAPI interface {
	GetUsers() ([]*casdoorsdk.User, error)
	GetRoles() ([]*casdoorsdk.Role, error)
}

type sdkAuthoritySnapshotAPI struct{}

func (sdkAuthoritySnapshotAPI) GetUsers() ([]*casdoorsdk.User, error) {
	return casdoorsdk.GetUsers()
}

func (sdkAuthoritySnapshotAPI) GetRoles() ([]*casdoorsdk.Role, error) {
	return casdoorsdk.GetRoles()
}

func NewAuthoritySnapshotClient(credential Credential) (*AuthoritySnapshotClient, error) {
	return newAuthoritySnapshotClient(credential, sdkAuthoritySnapshotAPI{})
}

func newAuthoritySnapshotClient(
	credential Credential,
	api authoritySnapshotAPI,
) (*AuthoritySnapshotClient, error) {
	normalized, err := validateCredentialForPurpose(credential, PurposeAuthorityCutover)
	if err != nil {
		return nil, err
	}
	if api == nil {
		return nil, errors.New("casdoor: authority snapshot API is required")
	}
	return &AuthoritySnapshotClient{credential: normalized, api: api}, nil
}

func (c *AuthoritySnapshotClient) Snapshot(
	ctx context.Context,
	supportedRoles []string,
) (AuthoritySnapshot, error) {
	roleNames, err := normalizeNonEmptyList("authority cutover role", supportedRoles)
	if err != nil {
		return AuthoritySnapshot{}, err
	}
	var (
		users []*casdoorsdk.User
		roles []*casdoorsdk.Role
	)
	if err := withSDKConfig(ctx, c.credential, "read authority cutover snapshot", func() error {
		var snapshotErr error
		users, snapshotErr = c.api.GetUsers()
		if snapshotErr != nil {
			return snapshotErr
		}
		roles, snapshotErr = c.api.GetRoles()
		return snapshotErr
	}); err != nil {
		return AuthoritySnapshot{}, fmt.Errorf("casdoor: read authority cutover snapshot: %w", err)
	}

	organization := c.credential.Organization
	result := AuthoritySnapshot{
		Organization: organization,
		Users:        make([]AuthoritySnapshotUser, 0, len(users)),
		RoleMembers:  make(map[string][]string, len(roleNames)),
	}
	for _, user := range users {
		if user == nil || strings.TrimSpace(user.Owner) != organization || strings.TrimSpace(user.Name) == "" {
			continue
		}
		inactive := user.IsForbidden || user.IsDeleted
		result.Users = append(result.Users, AuthoritySnapshotUser{
			ID:                 strings.TrimSpace(user.Id),
			Owner:              strings.TrimSpace(user.Owner),
			Name:               strings.TrimSpace(user.Name),
			OrganizationAdmin:  user.IsAdmin && !inactive,
			ForbiddenOrDeleted: inactive,
		})
	}

	wanted := make(map[string]struct{}, len(roleNames))
	for _, role := range roleNames {
		wanted[role] = struct{}{}
		result.RoleMembers[role] = []string{}
	}
	seen := make(map[string]struct{}, len(roleNames))
	for _, role := range roles {
		if role == nil || strings.TrimSpace(role.Owner) != organization {
			continue
		}
		name := strings.TrimSpace(role.Name)
		if _, ok := wanted[name]; !ok {
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			return AuthoritySnapshot{}, fmt.Errorf("casdoor: duplicate authority cutover role %q", name)
		}
		seen[name] = struct{}{}
		if !role.IsEnabled {
			continue
		}
		if len(role.Groups) > 0 || len(role.Roles) > 0 || len(role.Domains) > 0 {
			return AuthoritySnapshot{}, fmt.Errorf(
				"casdoor: authority cutover role %q uses nested groups, roles, or domains; flatten membership before cutover",
				name,
			)
		}
		members, err := normalizeList("authority cutover role member", role.Users)
		if err != nil {
			return AuthoritySnapshot{}, fmt.Errorf("casdoor: normalize role %q members: %w", name, err)
		}
		result.RoleMembers[name] = members
	}
	return result, nil
}
