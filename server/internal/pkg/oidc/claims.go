package oidc

import (
	"encoding/json"
	"strings"
	"time"
)

// Claims OIDC ID Token 解析后的用户信息
// PreferredUsername 对齐标准字段 preferred_username。
//
//nolint:revive // 保持与 OIDC 标准 claims 命名一致
type Claims struct {
	Sub               string   `json:"sub"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	Picture           string   `json:"picture"`
	AMR               []string `json:"amr,omitempty"`
	AuthTime          int64    `json:"auth_time,omitempty"`
	AppID             string   `json:"-"`
	// ExpiresAt comes from the expiry accepted by the verified ID token.
	// Callers use it to retain revocation state for the token's real remaining
	// lifetime instead of a local cookie/session policy TTL.
	ExpiresAt int64 `json:"-"`
}

// GetUserID 返回 OIDC subject（唯一用户标识）
func (c *Claims) GetUserID() string {
	return c.Sub
}

func (c *Claims) GetAppID() string {
	if c == nil {
		return ""
	}
	return c.AppID
}

// GetUsername 返回用户名
func (c *Claims) GetUsername() string {
	if c.PreferredUsername != "" {
		return c.PreferredUsername
	}
	return c.Name
}

// GetDisplayName 返回显示名称
func (c *Claims) GetDisplayName() string {
	return c.Name
}

// GetEmail 返回邮箱
func (c *Claims) GetEmail() string {
	return c.Email
}

// GetAvatar 返回头像 URL，空值返回 nil
func (c *Claims) GetAvatar() *string {
	if c.Picture == "" {
		return nil
	}
	return &c.Picture
}

func (c *Claims) MFAProofVerifiedAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	return MFAProofVerifiedAt(c.AMR, c.AuthTime)
}

func appIDFromRawClaims(rawJSON []byte) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return ""
	}
	if appID := authorizedPartyFromRaw(raw); appID != "" {
		return appID
	}
	audiences := claimStringList(raw["aud"])
	if len(audiences) == 1 {
		return audiences[0]
	}
	return ""
}

func authorizedPartyFromRawClaims(rawJSON []byte) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return ""
	}
	return authorizedPartyFromRaw(raw)
}

func authorizedPartyFromRaw(raw map[string]json.RawMessage) string {
	if appID := claimString(raw["azp"]); appID != "" {
		return appID
	}
	return claimString(raw["client_id"])
}

func claimString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func claimStringList(raw json.RawMessage) []string {
	if single := claimString(raw); single != "" {
		return []string{single}
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	return nonEmptyStrings(values)
}

func nonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

const (
	amrMFA      = "mfa"
	amrOTP      = "otp"
	amrSMS      = "sms"
	amrPhone    = "phone"
	amrApp      = "app"
	amrTOTP     = "totp"
	amrWebAuthn = "webauthn"
	amrFIDO     = "fido"
	amrFIDO2    = "fido2"
	amrHWK      = "hwk"
)

var mfaAMRMethods = map[string]struct{}{
	amrMFA:      {},
	amrOTP:      {},
	amrSMS:      {},
	amrPhone:    {},
	amrApp:      {},
	amrTOTP:     {},
	amrWebAuthn: {},
	amrFIDO:     {},
	amrFIDO2:    {},
	amrHWK:      {},
}

func MFAProofVerifiedAt(amr []string, authTime int64) time.Time {
	if authTime <= 0 || !HasMFAAMR(amr) {
		return time.Time{}
	}
	return time.Unix(authTime, 0).UTC()
}

func HasMFAAMR(amr []string) bool {
	for _, method := range amr {
		normalized := strings.ToLower(strings.TrimSpace(method))
		if _, ok := mfaAMRMethods[normalized]; ok {
			return true
		}
	}
	return false
}
