package casdoor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

var ErrRoleSyncCredentialNotConfigured = errors.New("casdoor role sync credential is not configured")

type RoleSyncClient struct {
	roleCredential       Credential
	userLookupCredential Credential
	roles                roleAPI
	users                userAPI
}

func NewRoleSyncClient(roleCredential, userLookupCredential Credential) (*RoleSyncClient, error) {
	return newRoleSyncClient(roleCredential, userLookupCredential, sdkRoleAPI{}, sdkUserAPI{})
}

func newRoleSyncClient(roleCredential, userLookupCredential Credential, roles roleAPI, users userAPI) (*RoleSyncClient, error) {
	roleCredential, err := validateCredentialForPurpose(roleCredential, PurposeRoleSync)
	if err != nil {
		return nil, err
	}
	userLookupCredential, err = validateCredentialForPurpose(userLookupCredential, PurposeUserLookup)
	if err != nil {
		return nil, err
	}
	if roles == nil || users == nil {
		return nil, errors.New("casdoor: role and user APIs are required")
	}
	return &RoleSyncClient{roleCredential: roleCredential, userLookupCredential: userLookupCredential, roles: roles, users: users}, nil
}

func (c *RoleSyncClient) SyncRole(ctx context.Context, subject, role string, approved bool) error {
	subject, role, err := normalizeRoleSyncInput(subject, role)
	if err != nil {
		return err
	}
	roleUser, legacyUser, err := c.lookupRoleUser(ctx, subject)
	if err != nil {
		return err
	}
	roleObj, err := c.getRole(ctx, role)
	if err != nil {
		return err
	}
	users, changed := roleUsersAfterApproval(roleObj.Users, roleUser, legacyUser, approved)
	if !changed {
		return nil
	}
	roleObj.Users = users
	return c.updateRoleUsers(ctx, roleObj)
}

func (c *RoleSyncClient) UserHasRole(ctx context.Context, subject, role string) (bool, error) {
	subject, role, err := normalizeRoleSyncInput(subject, role)
	if err != nil {
		return false, err
	}
	roleUser, legacyUser, err := c.lookupRoleUser(ctx, subject)
	if err != nil {
		return false, err
	}
	roleObj, err := c.getRole(ctx, role)
	if err != nil {
		return false, err
	}
	users := uniqueNonBlank(roleObj.Users)
	return containsString(users, roleUser) || containsString(users, legacyUser), nil
}

func normalizeRoleSyncInput(subject, role string) (string, string, error) {
	subject = strings.TrimSpace(subject)
	role = strings.TrimSpace(role)
	if subject == "" {
		return "", "", errors.New("casdoor: user subject is required")
	}
	if err := validateName("role name", role); err != nil {
		return "", "", err
	}
	return subject, role, nil
}

func (c *RoleSyncClient) lookupRoleUser(ctx context.Context, subject string) (string, string, error) {
	var user *casdoorsdk.User
	err := withSDKConfig(ctx, c.userLookupCredential, "lookup user "+subject, func() error {
		var lookupErr error
		user, lookupErr = c.users.GetUserByUserId(subject)
		return lookupErr
	})
	if err != nil {
		return "", "", err
	}
	if user == nil || strings.TrimSpace(user.Name) == "" {
		return "", "", fmt.Errorf("casdoor: user subject %q not found", subject)
	}
	owner := strings.TrimSpace(user.Owner)
	if owner == "" {
		owner = strings.TrimSpace(c.userLookupCredential.Organization)
	}
	if owner == "" {
		return "", "", fmt.Errorf("casdoor: user subject %q owner not found", subject)
	}
	name := strings.TrimSpace(user.Name)
	return owner + "/" + name, name, nil
}

func (c *RoleSyncClient) getRole(ctx context.Context, role string) (*casdoorsdk.Role, error) {
	var roleObj *casdoorsdk.Role
	err := withSDKConfig(ctx, c.roleCredential, "get role "+role, func() error {
		var getErr error
		roleObj, getErr = c.roles.GetRole(role)
		return getErr
	})
	if err != nil {
		return nil, err
	}
	if roleObj == nil {
		return nil, fmt.Errorf("casdoor: role %q not found", role)
	}
	return roleObj, nil
}

func (c *RoleSyncClient) updateRoleUsers(ctx context.Context, role *casdoorsdk.Role) error {
	return callWithCredential(ctx, c.roleCredential, "update role users "+role.Name, func() (bool, error) {
		return c.roles.UpdateRoleForColumns(role, []string{"users"})
	})
}

func roleUsersAfterApproval(users []string, roleUser, legacyUser string, approved bool) ([]string, bool) {
	normalized := removeString(uniqueNonBlank(users), legacyUser)
	contains := containsString(normalized, roleUser)
	switch {
	case approved && contains:
		return normalized, !sameStrings(users, normalized)
	case approved:
		return append(normalized, roleUser), true
	case !contains:
		return normalized, !sameStrings(users, normalized)
	default:
		return removeString(normalized, roleUser), true
	}
}

func uniqueNonBlank(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func removeString(values []string, target string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func BuildRoleSyncFunc(client *RoleSyncClient, getUserSubject func(context.Context, int64) (string, error)) func(context.Context, int64, string, bool) error {
	if client == nil {
		return func(_ context.Context, userID int64, role string, approved bool) error {
			logger.L().Info("role sync skipped (Casdoor role sync credentials not configured)",
				zap.Int64("user_id", userID),
				zap.String("role", role),
				zap.Bool("approved", approved),
			)
			return fmt.Errorf("%w: role=%s approved=%t user_id=%d", ErrRoleSyncCredentialNotConfigured, role, approved, userID)
		}
	}

	return func(ctx context.Context, userID int64, role string, approved bool) error {
		subject, err := getUserSubject(ctx, userID)
		if err != nil {
			return fmt.Errorf("get Casdoor subject: %w", err)
		}
		if err := client.SyncRole(ctx, subject, role, approved); err != nil {
			return fmt.Errorf("sync role via Casdoor: %w", err)
		}
		return nil
	}
}

func BuildRoleMembershipFunc(
	client *RoleSyncClient,
	getUserSubject func(context.Context, int64) (string, error),
) func(context.Context, int64, string) (bool, error) {
	if client == nil {
		return func(_ context.Context, userID int64, role string) (bool, error) {
			logger.L().Info("role membership check skipped (Casdoor role sync credentials not configured)",
				zap.Int64("user_id", userID),
				zap.String("role", role),
			)
			return false, fmt.Errorf("%w: role=%s user_id=%d", ErrRoleSyncCredentialNotConfigured, role, userID)
		}
	}

	return func(ctx context.Context, userID int64, role string) (bool, error) {
		subject, err := getUserSubject(ctx, userID)
		if err != nil {
			return false, fmt.Errorf("get Casdoor subject: %w", err)
		}
		allowed, err := client.UserHasRole(ctx, subject, role)
		if err != nil {
			return false, fmt.Errorf("check role via Casdoor: %w", err)
		}
		return allowed, nil
	}
}
