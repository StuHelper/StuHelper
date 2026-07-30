// Package usersync 定义 auth 与 user 两个领域包共享的用户同步值类型。
// 本包不依赖 modules/auth 或 modules/user，仅用于打破二者之间的循环引用。
package usersync

// Input 认证同步输入（OIDC / SSO 登录后写入 shadow user）。
type Input struct {
	CasdoorSubject string
	Username       string
	Email          string
	AvatarURL      *string
	Roles          []string
	// RolesAuthoritative 仅在 Roles 来自刚签发并完成验签、且显式包含合法 roles claim
	// 的 ID token 时为 true。旧 access token、userinfo 或缺失/畸形 claim 不得据此
	// 增删平台级 OpenFGA role tuple。
	RolesAuthoritative bool
}
