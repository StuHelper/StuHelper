package pii

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func TestNewCipher_EmptyKeys(t *testing.T) {
	_, err := NewCipher(1, map[uint8][]byte{})
	assert.ErrorIs(t, err, ErrEmptyKeys)

	_, err = NewCipher(1, nil)
	assert.ErrorIs(t, err, ErrEmptyKeys)
}

func TestNewCipher_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name   string
		keyLen int
	}{
		{"太短_16字节", 16},
		{"太短_1字节", 1},
		{"太长_64字节", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := NewCipher(1, map[uint8][]byte{1: key})
			assert.ErrorIs(t, err, ErrInvalidKeyLen)
		})
	}
}

func TestNewCipher_MissingActiveKey(t *testing.T) {
	key := generateKey(t)
	_, err := NewCipher(2, map[uint8][]byte{1: key})
	assert.ErrorIs(t, err, ErrMissingActive)
}

func TestNewCipher_Success(t *testing.T) {
	key := generateKey(t)
	c, err := NewCipher(1, map[uint8][]byte{1: key})
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestNewCipher_DeepCopyKeys(t *testing.T) {
	key := generateKey(t)
	original := make([]byte, 32)
	copy(original, key)

	c, err := NewCipher(1, map[uint8][]byte{1: key})
	require.NoError(t, err)

	// 修改原始 key，不应影响已构造的 cipher
	for i := range key {
		key[i] = 0
	}

	ct, err := c.Encrypt("test")
	require.NoError(t, err)

	pt, err := c.decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, "test", string(pt))
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	key := generateKey(t)
	c, err := NewCipher(1, map[uint8][]byte{1: key})
	require.NoError(t, err)

	_, err = c.Encrypt("")
	assert.ErrorIs(t, err, ErrEmptyPlaintext)
}

func TestEncrypt_Roundtrip(t *testing.T) {
	key := generateKey(t)
	c, err := NewCipher(1, map[uint8][]byte{1: key})
	require.NoError(t, err)

	tests := []string{
		"110101199001011234",                   // 大陆身份证
		"W12345678",                            // 护照
		"Hello, 世界!",                           // 含中文
		"a",                                    // 单字符
		"abcdefghijklmnopqrstuvwxyz0123456789", // 较长字符串
	}

	for _, plaintext := range tests {
		t.Run(plaintext, func(t *testing.T) {
			ct, err := c.Encrypt(plaintext)
			require.NoError(t, err)

			pt, err := c.decrypt(ct)
			require.NoError(t, err)
			assert.Equal(t, plaintext, string(pt))
		})
	}
}

func TestEncrypt_NonDeterministic(t *testing.T) {
	key := generateKey(t)
	c, err := NewCipher(1, map[uint8][]byte{1: key})
	require.NoError(t, err)

	plaintext := "110101199001011234"
	ct1, err := c.Encrypt(plaintext)
	require.NoError(t, err)
	ct2, err := c.Encrypt(plaintext)
	require.NoError(t, err)

	// 同一明文两次加密结果必须不同（随机 nonce）
	assert.NotEqual(t, ct1, ct2)

	// 但解密后都能得到原文
	pt1, err := c.decrypt(ct1)
	require.NoError(t, err)
	pt2, err := c.decrypt(ct2)
	require.NoError(t, err)
	assert.Equal(t, string(pt1), string(pt2))
}

func TestEncrypt_EnvelopeFormat(t *testing.T) {
	key := generateKey(t)
	activeKeyID := uint8(3)
	c, err := NewCipher(activeKeyID, map[uint8][]byte{activeKeyID: key})
	require.NoError(t, err)

	ct, err := c.Encrypt("test-data")
	require.NoError(t, err)

	// 验证信封最小长度
	assert.GreaterOrEqual(t, len(ct), envelopeHeaderSize+1)

	// 验证版本号
	assert.Equal(t, envelopeVersion, ct[0])

	// 验证 keyID
	assert.Equal(t, activeKeyID, ct[1])

	// nonce 区域（第 3-14 字节）应非全零
	nonce := ct[2:14]
	allZero := true
	for _, b := range nonce {
		if b != 0 {
			allZero = false
			break
		}
	}
	assert.False(t, allZero, "nonce 不应全为零")
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := generateKey(t)
	c, err := NewCipher(1, map[uint8][]byte{1: key})
	require.NoError(t, err)

	ct, err := c.Encrypt("sensitive-data")
	require.NoError(t, err)

	// 篡改密文数据区（header 之后的部分）
	tampered := make([]byte, len(ct))
	copy(tampered, ct)
	tampered[envelopeHeaderSize] ^= 0xFF

	_, err = c.decrypt(tampered)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication error")
}

func TestDecrypt_UnknownVersion(t *testing.T) {
	key := generateKey(t)
	c, err := NewCipher(1, map[uint8][]byte{1: key})
	require.NoError(t, err)

	ct, err := c.Encrypt("test")
	require.NoError(t, err)

	// 修改版本号
	tampered := make([]byte, len(ct))
	copy(tampered, ct)
	tampered[0] = 0xFF

	_, err = c.decrypt(tampered)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown envelope version")
}

func TestDecrypt_UnknownKeyID(t *testing.T) {
	key1 := generateKey(t)
	key2 := generateKey(t)

	// 用 key1 加密
	c1, err := NewCipher(1, map[uint8][]byte{1: key1})
	require.NoError(t, err)

	ct, err := c1.Encrypt("test")
	require.NoError(t, err)

	// 用只包含 key2 的 cipher 尝试解密
	c2, err := NewCipher(2, map[uint8][]byte{2: key2})
	require.NoError(t, err)

	_, err = c2.decrypt(ct)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown key ID")
}

func TestDecrypt_CiphertextTooShort(t *testing.T) {
	key := generateKey(t)
	c, err := NewCipher(1, map[uint8][]byte{1: key})
	require.NoError(t, err)

	_, err = c.decrypt([]byte{0x01, 0x01})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestMultiKeyRotation(t *testing.T) {
	key1 := generateKey(t)
	key2 := generateKey(t)

	// 用 key1 加密
	c1, err := NewCipher(1, map[uint8][]byte{1: key1})
	require.NoError(t, err)

	ct, err := c1.Encrypt("rotation-test")
	require.NoError(t, err)

	// 创建包含两个 key 的 cipher（activeKeyID=2），仍可解密 key1 加密的数据
	c2, err := NewCipher(2, map[uint8][]byte{1: key1, 2: key2})
	require.NoError(t, err)

	pt, err := c2.decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, "rotation-test", string(pt))

	// 新加密使用 key2
	ct2, err := c2.Encrypt("new-data")
	require.NoError(t, err)
	assert.Equal(t, uint8(2), ct2[1]) // keyID 应为 2
}
