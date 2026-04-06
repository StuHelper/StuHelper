package ldap

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	ldapv3 "github.com/go-ldap/ldap/v3"
)

var (
	// ErrInvalidUID 表示 uid 非法（防止 LDAP 注入）。
	ErrInvalidUID = errors.New("invalid uid")
	// ErrUserNotFound 表示查无此人。
	ErrUserNotFound = errors.New("ldap user not found")
	// ErrMissingConfig 表示必填配置项缺失。
	ErrMissingConfig = errors.New("ldap: missing required config")
)

const (
	defaultTimeout  = 5 * time.Second
	uidRegexPattern = `^[a-zA-Z0-9._-]+$`
)

var uidRegex = regexp.MustCompile(uidRegexPattern)

// Config LDAP 连接配置。
type Config struct {
	// URL 为 LDAP 服务器地址，如 ldap://host:389 或 ldaps://host:636，必填。
	URL string
	// BaseDN 为搜索的基础 DN，必填。
	BaseDN string
	// SystemBindDN 为系统绑定账号的完整 DN，必填。
	SystemBindDN string
	// SystemBindPassword 为系统绑定账号的密码，必填。
	SystemBindPassword string
	// Timeout 为连接超时时间，默认 5s。
	Timeout time.Duration
	// UseTLS 启用 StartTLS 升级（仅对 ldap:// 连接生效，ldaps:// 自动处理）。
	UseTLS bool
	// InsecureSkipVerify 跳过 TLS 证书验证（仅用于开发/测试环境）。
	InsecureSkipVerify bool
}

// Client LDAP 客户端。
type Client struct {
	cfg Config
}

// LoginResult 登录结果。
type LoginResult struct {
	Authenticated bool   `json:"authenticated"`
	ResultCode    uint16 `json:"resultCode,omitempty"`
	Message       string `json:"message,omitempty"`
}

// UserInfo LDAP 用户信息（按需求暴露字段）。
type UserInfo struct {
	UID              string `json:"uid"`
	CN               string `json:"cn"`
	SN               string `json:"sn"`
	EmployeeNumber   string `json:"employeeNumber"`
	DepartmentNumber string `json:"departmentNumber"`
	Mail             string `json:"mail"`
	Mobile           string `json:"mobile"`
	EmployeeType     string `json:"employeeType"`
}

// NewClient 创建 LDAP 客户端，必填配置缺失时返回错误。
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("%w: URL is required", ErrMissingConfig)
	}
	if strings.TrimSpace(cfg.BaseDN) == "" {
		return nil, fmt.Errorf("%w: BaseDN is required", ErrMissingConfig)
	}
	if strings.TrimSpace(cfg.SystemBindDN) == "" {
		return nil, fmt.Errorf("%w: SystemBindDN is required", ErrMissingConfig)
	}
	if cfg.SystemBindPassword == "" {
		return nil, fmt.Errorf("%w: SystemBindPassword is required", ErrMissingConfig)
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}

	return &Client{cfg: cfg}, nil
}

// Login 使用 uid + password 验证登录。
// 仅暴露 uid/password，内部按用户 DN 进行 bind。
func (c *Client) Login(ctx context.Context, uid, password string) (*LoginResult, error) {
	if err := validateUID(uid); err != nil {
		return nil, err
	}
	uid = strings.ToLower(uid)
	if password == "" {
		return &LoginResult{Authenticated: false, Message: "empty password"}, nil
	}

	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }() //nolint:errcheck // best-effort cleanup

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	userDN := fmt.Sprintf("uid=%s,%s", uid, c.cfg.BaseDN)
	if err := conn.Bind(userDN, password); err != nil {
		if ldapErr, ok := err.(*ldapv3.Error); ok {
			if ldapErr.ResultCode == ldapv3.LDAPResultInvalidCredentials {
				return &LoginResult{
					Authenticated: false,
					ResultCode:    ldapErr.ResultCode,
					Message:       ldapErr.Error(),
				}, nil
			}
		}
		return nil, fmt.Errorf("ldap bind failed: %w", err)
	}

	// Bind 成功即表示认证通过，无需额外搜索。
	return &LoginResult{Authenticated: true, ResultCode: ldapv3.LDAPResultSuccess}, nil
}

// QueryUserByUID 使用系统账号查人，仅暴露 uid 入参。
func (c *Client) QueryUserByUID(ctx context.Context, uid string) (*UserInfo, error) {
	if err := validateUID(uid); err != nil {
		return nil, err
	}
	uid = strings.ToLower(uid)

	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }() //nolint:errcheck // best-effort cleanup

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := conn.Bind(c.cfg.SystemBindDN, c.cfg.SystemBindPassword); err != nil {
		return nil, fmt.Errorf("system bind failed: %w", err)
	}

	searchReq := ldapv3.NewSearchRequest(
		c.cfg.BaseDN,
		ldapv3.ScopeWholeSubtree,
		ldapv3.NeverDerefAliases,
		1, // SizeLimit=1，防御性编程：uid 应唯一
		0,
		false,
		fmt.Sprintf("(uid=%s)", ldapv3.EscapeFilter(uid)),
		[]string{"uid", "cn", "sn", "employeeNumber", "departmentNumber", "mail", "mobile", "employeeType"},
		nil,
	)

	sr, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("ldap search failed: %w", err)
	}
	if len(sr.Entries) == 0 {
		return nil, ErrUserNotFound
	}

	e := sr.Entries[0]
	return &UserInfo{
		UID:              e.GetAttributeValue("uid"),
		CN:               e.GetAttributeValue("cn"),
		SN:               e.GetAttributeValue("sn"),
		EmployeeNumber:   e.GetAttributeValue("employeeNumber"),
		DepartmentNumber: e.GetAttributeValue("departmentNumber"),
		Mail:             e.GetAttributeValue("mail"),
		Mobile:           e.GetAttributeValue("mobile"),
		EmployeeType:     e.GetAttributeValue("employeeType"),
	}, nil
}

// dial 建立 LDAP 连接，根据配置决定是否升级 TLS。
// 同时设置操作超时，防止 Bind/Search 无限挂起。
func (c *Client) dial() (*ldapv3.Conn, error) {
	conn, err := ldapv3.DialURL(
		c.cfg.URL,
		ldapv3.DialWithDialer(&net.Dialer{Timeout: c.cfg.Timeout}),
	)
	if err != nil {
		return nil, fmt.Errorf("dial ldap failed: %w", err)
	}

	// 设置后续 Bind/Search 等操作的超时，防止远程服务器无响应时 goroutine 泄漏
	conn.SetTimeout(c.cfg.Timeout)

	// go-ldap 对 ldaps:// 自动处理 TLS；对 ldap:// 且启用 TLS 时执行 StartTLS 升级。
	if c.cfg.UseTLS && strings.HasPrefix(strings.ToLower(c.cfg.URL), "ldap://") {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: c.cfg.InsecureSkipVerify, //nolint:gosec // 仅开发/测试环境使用
		}
		if err := conn.StartTLS(tlsCfg); err != nil {
			_ = conn.Close() //nolint:errcheck // best-effort cleanup on StartTLS failure
			return nil, fmt.Errorf("ldap StartTLS failed: %w", err)
		}
	}

	return conn, nil
}

// validateUID 校验 uid 合法性（防止 LDAP 注入）。
func validateUID(uid string) error {
	uid = strings.TrimSpace(uid)
	if uid == "" || !uidRegex.MatchString(uid) {
		return ErrInvalidUID
	}
	return nil
}
