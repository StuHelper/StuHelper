// Package usersync 定义 auth 与 user 两个领域包共享的用户同步值类型。
// 本包不依赖 modules/auth 或 modules/user，仅用于打破二者之间的循环引用。
package usersync

import "errors"

// Input 认证同步输入（OIDC / SSO 登录后写入 shadow user）。
type Input struct {
	CasdoorSubject string
	Username       string
	Email          string
	AvatarURL      *string
	Roles          []string
}

// PhoneUser 通过手机号查询到的用户信息
type PhoneUser struct {
	ID             int64
	CasdoorSubject string
	Username       string
	Email          string
	MaskedPhone    string
	AvatarURL      *string
}

// ErrPhoneUserNotFound 手机号对应的用户不存在
var ErrPhoneUserNotFound = errors.New("usersync: user not found by phone")
