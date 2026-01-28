package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitHMACKey(t *testing.T) {
	// 初始化密钥
	InitHMACKey("test-secret-key")

	// 验证哈希功能正常
	hash := HMACHash("test-data")
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA256 输出 32 字节，hex 编码后 64 字符
}

func TestHMACHash(t *testing.T) {
	InitHMACKey("test-secret-key")

	tests := []struct {
		name  string
		input string
	}{
		{"空字符串", ""},
		{"普通字符串", "hello"},
		{"用户ID", "user-123"},
		{"特殊字符", "user@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HMACHash(tt.input)
			if tt.input == "" {
				assert.Empty(t, hash)
			} else {
				assert.NotEmpty(t, hash)
				assert.Len(t, hash, 64)
			}
		})
	}
}

func TestHMACHash_Consistency(t *testing.T) {
	InitHMACKey("test-secret-key")

	// 相同输入应该产生相同输出
	hash1 := HMACHash("test-data")
	hash2 := HMACHash("test-data")
	assert.Equal(t, hash1, hash2)
}

func TestHMACHash_Uniqueness(t *testing.T) {
	InitHMACKey("test-secret-key")

	// 不同输入应该产生不同输出
	hash1 := HMACHash("data1")
	hash2 := HMACHash("data2")
	assert.NotEqual(t, hash1, hash2)
}

func TestHMACHashShort(t *testing.T) {
	InitHMACKey("test-secret-key")

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
			hash := HMACHashShort(tt.input, tt.length)
			if tt.length >= 64 {
				assert.Len(t, hash, 64)
			} else {
				assert.Len(t, hash, tt.length)
			}
		})
	}
}
