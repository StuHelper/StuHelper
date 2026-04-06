package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitHMACKey(t *testing.T) {
	// 初始化密钥
	require.NoError(t, InitHMACKey("test-secret-key", false))

	// 验证哈希功能正常
	hash, err := HMACHash("test-data")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA256 输出 32 字节，hex 编码后 64 字符
}

func TestHMACHash(t *testing.T) {
	require.NoError(t, InitHMACKey("test-secret-key", false))

	t.Run("空字符串应返回ErrEmptyInput", func(t *testing.T) {
		hash, err := HMACHash("")
		assert.ErrorIs(t, err, ErrEmptyInput)
		assert.Empty(t, hash)
	})

	tests := []struct {
		name  string
		input string
	}{
		{"普通字符串", "hello"},
		{"用户ID", "user-123"},
		{"特殊字符", "user@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HMACHash(tt.input)
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.Len(t, hash, 64)
		})
	}
}

func TestHMACHash_Consistency(t *testing.T) {
	require.NoError(t, InitHMACKey("test-secret-key", false))

	// 相同输入应该产生相同输出
	hash1, err := HMACHash("test-data")
	require.NoError(t, err)
	hash2, err := HMACHash("test-data")
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)
}

func TestHMACHash_Uniqueness(t *testing.T) {
	require.NoError(t, InitHMACKey("test-secret-key", false))

	// 不同输入应该产生不同输出
	hash1, err := HMACHash("data1")
	require.NoError(t, err)
	hash2, err := HMACHash("data2")
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash2)
}

func TestHMACHashShort(t *testing.T) {
	require.NoError(t, InitHMACKey("test-secret-key", false))

	tests := []struct {
		name   string
		input  string
		length int
	}{
		{"长度16", "test-data", 16},
		{"长度8", "test-data", 8},
		{"长度超过哈希长度", "test-data", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HMACHashShort(tt.input, tt.length)
			require.NoError(t, err)
			if tt.length >= 64 {
				assert.Len(t, hash, 64)
			} else {
				assert.Len(t, hash, tt.length)
			}
		})
	}
}

func TestGetHMACKeyReturnsCopy(t *testing.T) {
	require.NoError(t, InitHMACKey("test-secret-key", false))

	key := GetHMACKey()
	require.NotNil(t, key)
	original := append([]byte(nil), key...)

	key[0] ^= 0xFF

	assert.Equal(t, original, GetHMACKey())
}
