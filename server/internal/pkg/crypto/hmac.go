package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

var (
	hmacKey     []byte
	hmacKeyOnce sync.Once
)

// InitHMACKey 初始化 HMAC 密钥（应在应用启动时调用一次）
func InitHMACKey(secret string) {
	hmacKeyOnce.Do(func() {
		hmacKey = []byte(secret)
	})
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
