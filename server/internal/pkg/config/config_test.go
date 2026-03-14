package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDocAESKeys_Valid(t *testing.T) {
	// 有效的 32 字节 hex key
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	keys, errs := parseDocAESKeys("1:" + hexKey)
	assert.Empty(t, errs)
	assert.Len(t, keys, 1)
	assert.Len(t, keys[1], 32)
}

func TestParseDocAESKeys_MultipleKeys(t *testing.T) {
	hexKey1 := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hexKey2 := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	keys, errs := parseDocAESKeys("1:" + hexKey1 + ",2:" + hexKey2)
	assert.Empty(t, errs)
	assert.Len(t, keys, 2)
}

func TestParseDocAESKeys_InvalidFormat(t *testing.T) {
	_, errs := parseDocAESKeys("invalid-no-colon")
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid entry")
}

func TestParseDocAESKeys_InvalidKeyID(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name  string
		input string
	}{
		{"零值", "0:" + hexKey},
		{"超范围", "256:" + hexKey},
		{"非数字", "abc:" + hexKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := parseDocAESKeys(tt.input)
			assert.NotEmpty(t, errs)
			assert.Contains(t, errs[0], "invalid key ID")
		})
	}
}

func TestParseDocAESKeys_DuplicateKeyID(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	_, errs := parseDocAESKeys("1:" + hexKey + ",1:" + hexKey)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "duplicate key ID")
}

func TestParseDocAESKeys_InvalidHex(t *testing.T) {
	_, errs := parseDocAESKeys("1:not-valid-hex!")
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid hex")
}

func TestParseDocAESKeys_WrongKeyLength(t *testing.T) {
	// 16 字节而不是 32 字节
	shortHex := "0123456789abcdef0123456789abcdef"
	_, errs := parseDocAESKeys("1:" + shortHex)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "must be exactly 32 bytes")
}

func TestParseDocAESKeys_EmptyInput(t *testing.T) {
	_, errs := parseDocAESKeys("")
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "no valid keys found")
}

func TestParseSecurityConfig_MissingActiveKeyID(t *testing.T) {
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "")
	t.Setenv("DOC_AES_KEYS", "1:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	_, errs := parseSecurityConfig()
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "DOC_AES_ACTIVE_KEY_ID is required")
}

func TestParseSecurityConfig_MissingKeys(t *testing.T) {
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "1")
	t.Setenv("DOC_AES_KEYS", "")

	_, errs := parseSecurityConfig()
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "DOC_AES_KEYS is required")
}

func TestParseSecurityConfig_ActiveKeyNotInKeySet(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "2")
	t.Setenv("DOC_AES_KEYS", "1:"+hexKey)

	cfg, errs := parseSecurityConfig()
	// parseSecurityConfig 本身不校验 activeKeyID 是否在 keys 中，
	// 由 validate() 统一校验
	assert.Empty(t, errs)
	assert.Equal(t, uint8(2), cfg.DocAESActiveKeyID)
}

func TestValidate_SecurityConfig_ActiveKeyMissing(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "2")
	t.Setenv("DOC_AES_KEYS", "1:"+hexKey)

	cfg, securityErrs := parseSecurityConfig()
	require.Empty(t, securityErrs)

	c := &Config{
		Security: cfg,
	}
	// 只验证安全配置相关校验
	err := c.validate(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DOC_AES_ACTIVE_KEY_ID=2 not found in DOC_AES_KEYS")
}

func TestParseSecurityConfig_ValidConfig(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("DOC_AES_ACTIVE_KEY_ID", "1")
	t.Setenv("DOC_AES_KEYS", "1:"+hexKey)

	cfg, errs := parseSecurityConfig()
	assert.Empty(t, errs)
	assert.Equal(t, uint8(1), cfg.DocAESActiveKeyID)
	assert.Len(t, cfg.DocAESKeys, 1)
	assert.Len(t, cfg.DocAESKeys[1], 32)
}
