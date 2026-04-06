package phoneutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

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
