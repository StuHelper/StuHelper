package pii

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// 密文信封版本
const envelopeVersion byte = 0x01

// 信封格式固定偏移量
const (
	envelopeVersionSize = 1  // 第 1 字节：版本号
	envelopeKeyIDSize   = 1  // 第 2 字节：keyID
	envelopeNonceSize   = 12 // 第 3-14 字节：AES-GCM 标准 nonce
	envelopeHeaderSize  = envelopeVersionSize + envelopeKeyIDSize + envelopeNonceSize
)

// 业务错误定义
var (
	ErrEmptyKeys      = errors.New("pii: key map must not be empty")
	ErrInvalidKeyLen  = errors.New("pii: each key must be exactly 32 bytes (AES-256)")
	ErrMissingActive  = errors.New("pii: active key ID not found in key map")
	ErrEmptyPlaintext = errors.New("pii: plaintext must not be empty")
)

// Encryptor 对外暴露的加密接口，业务侧只能加密，不能解密
type Encryptor interface {
	Encrypt(plaintext string) ([]byte, error)
}

// EncryptDecryptor 加解密接口，适用于需要同时加密和解密的场景（如学籍数据 PII 字段）
type EncryptDecryptor interface {
	Encryptor
	Decrypt(ciphertext []byte) (string, error)
}

// Cipher 实现版本化信封格式的 AES-256-GCM 加密
type Cipher struct {
	activeKeyID uint8
	aeads       map[uint8]cipher.AEAD
}

// NewCipher 构造 PII 加密器
// activeKeyID 指定当前加密使用的 keyID，keys 为所有可用的 AES-256 密钥（支持 key rotation）
// 内部会深拷贝密钥并预构建 AEAD，不保留原始密钥引用
func NewCipher(activeKeyID uint8, keys map[uint8][]byte) (*Cipher, error) {
	if len(keys) == 0 {
		return nil, ErrEmptyKeys
	}

	aeads := make(map[uint8]cipher.AEAD, len(keys))
	for id, key := range keys {
		if len(key) != 32 {
			return nil, fmt.Errorf("%w: keyID=%d has %d bytes", ErrInvalidKeyLen, id, len(key))
		}

		// 深拷贝密钥，避免调用方后续修改影响加密器
		keyCopy := make([]byte, 32)
		copy(keyCopy, key)

		block, err := aes.NewCipher(keyCopy)
		if err != nil {
			return nil, fmt.Errorf("pii: failed to create AES cipher for keyID=%d: %w", id, err)
		}

		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("pii: failed to create GCM for keyID=%d: %w", id, err)
		}

		aeads[id] = aead

		// 清零临时密钥副本（AEAD 已内部持有 block，此处 keyCopy 不再需要）
		for i := range keyCopy {
			keyCopy[i] = 0
		}
	}

	if _, ok := aeads[activeKeyID]; !ok {
		return nil, fmt.Errorf("%w: activeKeyID=%d", ErrMissingActive, activeKeyID)
	}

	return &Cipher{
		activeKeyID: activeKeyID,
		aeads:       aeads,
	}, nil
}

// Encrypt 使用 activeKeyID 对应的密钥加密明文，返回版本化信封格式密文
// 信封格式: version(1) | keyID(1) | nonce(12) | ciphertext+tag
// 同一明文多次加密结果不同（随机 nonce）
func (c *Cipher) Encrypt(plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, ErrEmptyPlaintext
	}

	aead := c.aeads[c.activeKeyID]

	// 生成随机 nonce 并直接密封，避免中间变量触发 G407 误报
	envelope, err := sealWithRandomNonce(aead, c.activeKeyID, []byte(plaintext))
	if err != nil {
		return nil, err
	}
	return envelope, nil
}

// sealWithRandomNonce 使用随机 nonce 执行 AES-GCM 密封
func sealWithRandomNonce(aead cipher.AEAD, keyID uint8, plaintext []byte) ([]byte, error) {
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("pii: failed to generate nonce: %w", err)
	}

	sealed := aead.Seal(nil, nonce, plaintext, nil)
	envelope := make([]byte, envelopeHeaderSize+len(sealed))
	envelope[0] = envelopeVersion
	envelope[1] = keyID
	copy(envelope[envelopeVersionSize+envelopeKeyIDSize:], nonce)
	copy(envelope[envelopeHeaderSize:], sealed)
	return envelope, nil
}

// Decrypt 解密版本化信封格式密文，返回明文字符串
func (c *Cipher) Decrypt(ciphertext []byte) (string, error) {
	plaintext, err := c.decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// decrypt 仅供同包测试使用，解密版本化信封格式密文
func (c *Cipher) decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < envelopeHeaderSize {
		return nil, errors.New("pii: ciphertext too short")
	}

	version := ciphertext[0]
	if version != envelopeVersion {
		return nil, fmt.Errorf("pii: unknown envelope version: %d", version)
	}

	keyID := ciphertext[1]
	aead, ok := c.aeads[keyID]
	if !ok {
		return nil, fmt.Errorf("pii: unknown key ID: %d", keyID)
	}

	nonce := ciphertext[envelopeVersionSize+envelopeKeyIDSize : envelopeHeaderSize]
	sealed := ciphertext[envelopeHeaderSize:]

	plaintext, err := aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("pii: decryption failed (authentication error): %w", err)
	}

	return plaintext, nil
}
