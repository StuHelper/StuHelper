package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"sync"
)

var (
	hmacKey     []byte
	hmacKeyOnce sync.Once
)

// ErrEmptyHMACSecret 表示生产环境 HMAC 密钥为空的错误
var ErrEmptyHMACSecret = errors.New("HMAC_SECRET is required in production environment")

// InitHMACKey 初始化 HMAC 密钥（应在应用启动时调用一次）
// 生产环境必须提供非空密钥，开发环境会生成随机密钥并警告
func InitHMACKey(secret string, isProduction bool) error {
	var initErr error
	hmacKeyOnce.Do(func() {
		if secret == "" {
			if isProduction {
				initErr = ErrEmptyHMACSecret
				return
			}
			// 开发环境：生成随机密钥并警告
			b := make([]byte, 32)
			if _, err := rand.Read(b); err != nil {
				initErr = errors.New("failed to generate random HMAC key: " + err.Error())
				return
			}
			hmacKey = b
			log.Println("WARNING: HMAC_SECRET not set, using random key. Set HMAC_SECRET in production!")
			return
		}
		// 验证密钥长度
		if len(secret) < 16 {
			log.Println("WARNING: HMAC_SECRET is too short (< 16 chars), consider using a longer key")
		}
		hmacKey = []byte(secret)
	})
	return initErr
}

// HMACHash 使用 HMAC-SHA256 对数据进行哈希
func HMACHash(data string) string {
	if data == "" {
		return ""
	}
	h := hmac.New(sha256.New, hmacKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// HMACHashShort 返回截断的 HMAC 哈希（用于缓存 key 等场景）
func HMACHashShort(data string, length int) string {
	hash := HMACHash(data)
	if len(hash) > length {
		return hash[:length]
	}
	return hash
}
