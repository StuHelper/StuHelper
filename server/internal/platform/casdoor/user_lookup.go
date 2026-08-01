package casdoor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

var (
	ErrUserOwnerMismatch     = errors.New("casdoor: user owner mismatch")
	ErrUserLookupUnavailable = errors.New("casdoor: user lookup unavailable")
)

type SubjectIdentity struct {
	Organization      string
	OrganizationAdmin bool
}

type UserLookupClient struct {
	credential Credential
	users      userAPI
}

func NewUserLookupClient(credential Credential) (*UserLookupClient, error) {
	return newUserLookupClient(credential, sdkUserAPI{})
}

func newUserLookupClient(credential Credential, users userAPI) (*UserLookupClient, error) {
	normalized, err := validateCredentialForPurpose(credential, PurposeUserLookup)
	if err != nil {
		return nil, err
	}
	if users == nil {
		return nil, errors.New("casdoor: user API is required")
	}
	return &UserLookupClient{credential: normalized, users: users}, nil
}

func (c *UserLookupClient) ValidateSubjectOwner(ctx context.Context, subject, expectedOwner string) error {
	_, err := c.ResolveSubject(ctx, subject, expectedOwner)
	return err
}

func (c *UserLookupClient) ResolveSubject(
	ctx context.Context,
	subject,
	expectedOwner string,
) (SubjectIdentity, error) {
	subject = strings.TrimSpace(subject)
	expectedOwner = strings.TrimSpace(expectedOwner)
	if subject == "" {
		return SubjectIdentity{}, errors.New("casdoor: user subject is required")
	}
	if expectedOwner == "" {
		return SubjectIdentity{}, errors.New("casdoor: expected user owner is required")
	}
	user, err := c.lookupUser(ctx, subject)
	if err != nil {
		return SubjectIdentity{}, err
	}
	owner := strings.TrimSpace(user.Owner)
	if owner != expectedOwner {
		return SubjectIdentity{}, fmt.Errorf("%w: got %q, want %q", ErrUserOwnerMismatch, owner, expectedOwner)
	}
	return SubjectIdentity{
		Organization:      owner,
		OrganizationAdmin: user.IsAdmin && !user.IsForbidden && !user.IsDeleted,
	}, nil
}

func (c *UserLookupClient) lookupUser(ctx context.Context, subject string) (*casdoorsdk.User, error) {
	var user *casdoorsdk.User
	err := withSDKConfig(ctx, c.credential, "lookup user "+subject, func() error {
		var lookupErr error
		user, lookupErr = c.users.GetUserByUserId(subject)
		return lookupErr
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserLookupUnavailable, err)
	}
	if user == nil || strings.TrimSpace(user.Name) == "" {
		return nil, fmt.Errorf("casdoor: user subject %q not found", subject)
	}
	return user, nil
}
