package casdoor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

const mainlandChinaCountryCode = "CN"

type UserProfileClient struct {
	credential Credential
	users      userAPI
	sms        smsAPI
}

type UserPhoneUpdate struct {
	Subject string
	Phone   string
}

func NewUserProfileClient(credential Credential) (*UserProfileClient, error) {
	return newUserProfileClient(credential, sdkUserAPI{}, sdkSMSAPI{})
}

func newUserProfileClient(credential Credential, users userAPI, sms smsAPI) (*UserProfileClient, error) {
	normalized, err := validateCredentialForPurpose(credential, PurposeUserProfile)
	if err != nil {
		return nil, err
	}
	if users == nil {
		return nil, errors.New("casdoor: user API is required")
	}
	if sms == nil {
		return nil, errors.New("casdoor: sms API is required")
	}
	return &UserProfileClient{credential: normalized, users: users, sms: sms}, nil
}

func (c *UserProfileClient) GetPhone(ctx context.Context, subject string) (string, error) {
	trimmed := strings.TrimSpace(subject)
	if trimmed == "" {
		return "", errors.New("casdoor: user subject is required")
	}
	user, err := c.lookupUser(ctx, trimmed)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(user.Phone), nil
}

func (c *UserProfileClient) UpdatePhone(ctx context.Context, input UserPhoneUpdate) error {
	subject := strings.TrimSpace(input.Subject)
	phone := strings.TrimSpace(input.Phone)
	if subject == "" {
		return errors.New("casdoor: user subject is required")
	}
	if phone == "" {
		return errors.New("casdoor: phone is required")
	}

	user, err := c.lookupUser(ctx, subject)
	if err != nil {
		return err
	}
	user.Phone = phone
	user.CountryCode = mainlandChinaCountryCode
	return callWithCredential(ctx, c.credential, "update user phone "+subject, func() (bool, error) {
		return c.users.UpdateUserForColumns(user, []string{"phone", "countryCode"})
	})
}

func (c *UserProfileClient) Send(ctx context.Context, phone, content string) error {
	phone = strings.TrimSpace(phone)
	content = strings.TrimSpace(content)
	if phone == "" {
		return errors.New("casdoor: sms phone is required")
	}
	if content == "" {
		return errors.New("casdoor: sms content is required")
	}
	return withSDKConfig(ctx, c.credential, "send sms", func() error {
		return c.sms.SendSms(content, phone)
	})
}

func (c *UserProfileClient) lookupUser(ctx context.Context, subject string) (*casdoorsdk.User, error) {
	var user *casdoorsdk.User
	err := withSDKConfig(ctx, c.credential, "lookup user "+subject, func() error {
		var lookupErr error
		user, lookupErr = c.users.GetUserByUserId(subject)
		return lookupErr
	})
	if err != nil {
		return nil, err
	}
	if user == nil || strings.TrimSpace(user.Name) == "" {
		return nil, fmt.Errorf("casdoor: user subject %q not found", subject)
	}
	return user, nil
}
