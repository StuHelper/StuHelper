package ldap

import (
	"context"
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
)

// 默认配置（与现有文档保持一致）。
const (
	defaultLDAPURL  = "ldap://10.212.24.175"
	defaultBaseDN   = "ou=people,dc=buaa,dc=edu,dc=cn"
	defaultSystemDN = "uid=test,ou=system,dc=buaa,dc=edu,dc=cn"
	defaultSystemPW = "test"
	defaultTimeout  = 5 * time.Second
	uidRegexPattern = `^[a-zA-Z0-9._-]+$`
)

var uidRegex = regexp.MustCompile(uidRegexPattern)

// Config LDAP 连接配置。
type Config struct {
	URL                string
	BaseDN             string
	SystemBindDN       string // test
	SystemBindPassword string // test
	Timeout            time.Duration
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

// NewClient 创建 LDAP 客户端。
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		cfg.URL = defaultLDAPURL
	}
	if strings.TrimSpace(cfg.BaseDN) == "" {
		cfg.BaseDN = defaultBaseDN
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}

	if strings.TrimSpace(cfg.SystemBindDN) == "" {
		cfg.SystemBindDN = defaultSystemDN
	}
	if cfg.SystemBindPassword == "" {
		cfg.SystemBindPassword = defaultSystemPW
	}

	return &Client{cfg: cfg}, nil
}

// Login 使用 uid + password 验证登录。
// 仅暴露 uid/password，内部按用户 DN 进行 bind。
func (c *Client) Login(ctx context.Context, uid, password string) (*LoginResult, error) {
	if err := validateUID(uid); err != nil {
		return nil, err
	}
	uid = strings.ToLower(strings.TrimSpace(uid))
	if password == "" {
		return &LoginResult{Authenticated: false, Message: "empty password"}, nil
	}

	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

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

	// 登录成功后做一次 objectclass=* 查询。
	searchReq := ldapv3.NewSearchRequest(
		c.cfg.BaseDN,
		ldapv3.ScopeWholeSubtree,
		ldapv3.NeverDerefAliases,
		0,
		0,
		false,
		"(objectclass=*)",
		[]string{"dn"},
		nil,
	)

	if _, err := conn.Search(searchReq); err != nil {
		if ldapErr, ok := err.(*ldapv3.Error); ok {
			return &LoginResult{
				Authenticated: true,
				ResultCode:    ldapErr.ResultCode,
				Message:       ldapErr.Error(),
			}, nil
		}

		return &LoginResult{
			Authenticated: true,
			Message:       err.Error(),
		}, nil
	}

	return &LoginResult{Authenticated: true, ResultCode: ldapv3.LDAPResultSuccess}, nil
}

// QueryUserByUID 使用系统账号查人，仅暴露 uid 入参。
func (c *Client) QueryUserByUID(ctx context.Context, uid string) (*UserInfo, error) {
	if err := validateUID(uid); err != nil {
		return nil, err
	}
	uid = strings.ToLower(strings.TrimSpace(uid))

	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

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
		0,
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

func (c *Client) dial() (*ldapv3.Conn, error) {
	conn, err := ldapv3.DialURL(
		c.cfg.URL,
		ldapv3.DialWithDialer(&net.Dialer{Timeout: c.cfg.Timeout}),
	)
	if err != nil {
		return nil, fmt.Errorf("dial ldap failed: %w", err)
	}
	return conn, nil
}

func validateUID(uid string) error {
	uid = strings.TrimSpace(uid)
	if uid == "" || !uidRegex.MatchString(uid) {
		return ErrInvalidUID
	}
	return nil
}
