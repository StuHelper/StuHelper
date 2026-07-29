package phoneutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"

	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

// mainlandPhonePattern 中国大陆手机号正则（号段 13x-19x，共 11 位）
var mainlandPhonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

// IsValidMainlandPhone 判断是否为合法的中国大陆手机号。
func IsValidMainlandPhone(phone string) bool {
	return mainlandPhonePattern.MatchString(phone)
}

const phoneLookupScope = "phone_lookup:"

var ErrEmptyHMACKey = errors.New("phoneutil: hmac key must not be empty")

func HashLookupWithKey(phone string, key []byte) (string, error) {
	if len(key) == 0 {
		return "", ErrEmptyHMACKey
	}
	h := hmac.New(sha256.New, key)
	h.Write([]byte(phoneLookupScope + strings.TrimSpace(phone)))
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashLookup 生成用于手机号精确查询/去重的稳定 HMAC。
func HashLookup(phone string) (string, error) {
	key := crypto.GetHMACKey()
	if len(key) == 0 {
		return "", crypto.ErrHMACNotInitialized
	}
	return HashLookupWithKey(phone, key)
}

// Mask 返回手机号掩码。
func Mask(phone string) string {
	return logger.MaskPhone(strings.TrimSpace(phone))
}
