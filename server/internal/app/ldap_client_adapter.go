package app

import (
	"context"
	"errors"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
)

type ldapClientAdapter struct {
	client *ldap.Client
}

func newLDAPAuthClient(cfg user.LDAPConfig) (user.LDAPAuthClient, error) {
	client, err := ldap.NewClient(ldap.Config{
		URL:                cfg.URL,
		BaseDN:             cfg.BaseDN,
		SystemBindDN:       cfg.SystemBindDN,
		SystemBindPassword: cfg.SystemBindPassword,
		UseTLS:             cfg.UseTLS,
	})
	if err != nil {
		return nil, err
	}
	return ldapClientAdapter{client: client}, nil
}

func (a ldapClientAdapter) Login(ctx context.Context, uid, password string) (*user.LDAPLoginResult, error) {
	result, err := a.client.Login(ctx, uid, password)
	if err != nil {
		return nil, normalizeLDAPClientError(err)
	}
	return &user.LDAPLoginResult{Authenticated: result.Authenticated}, nil
}

func (a ldapClientAdapter) QueryUserByUID(ctx context.Context, uid string) (*user.LDAPUserInfo, error) {
	info, err := a.client.QueryUserByUID(ctx, uid)
	if err != nil {
		return nil, normalizeLDAPClientError(err)
	}
	return &user.LDAPUserInfo{
		UID:              info.UID,
		CN:               info.CN,
		SN:               info.SN,
		EmployeeNumber:   info.EmployeeNumber,
		DepartmentNumber: info.DepartmentNumber,
		Mail:             info.Mail,
		Mobile:           info.Mobile,
		EmployeeType:     info.EmployeeType,
	}, nil
}

func normalizeLDAPClientError(err error) error {
	if errors.Is(err, ldap.ErrInvalidUID) {
		return user.ErrLDAPFailed
	}
	return err
}
