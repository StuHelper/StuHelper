package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// parseSecurityConfig 解析 PII 加密相关环境变量，返回强类型配置和解析错误列表
func parseSecurityConfig() (SecurityConfig, []string) {
	var errs []string
	var cfg SecurityConfig

	activeKeyIDStr := os.Getenv("DOC_AES_ACTIVE_KEY_ID")
	if activeKeyIDStr == "" {
		errs = append(errs, "DOC_AES_ACTIVE_KEY_ID is required but not set")
		return cfg, errs
	}

	activeKeyID, err := strconv.ParseUint(activeKeyIDStr, 10, 8)
	if err != nil || activeKeyID < 1 || activeKeyID > 255 {
		errs = append(errs, fmt.Sprintf("DOC_AES_ACTIVE_KEY_ID must be 1-255, got %q", activeKeyIDStr))
		return cfg, errs
	}
	cfg.DocAESActiveKeyID = uint8(activeKeyID)

	keysStr := os.Getenv("DOC_AES_KEYS")
	if keysStr == "" {
		errs = append(errs, "DOC_AES_KEYS is required but not set")
		return cfg, errs
	}

	keys, keyErrs := parseDocAESKeys(keysStr)
	if len(keyErrs) > 0 {
		errs = append(errs, keyErrs...)
		return cfg, errs
	}
	cfg.DocAESKeys = keys

	return cfg, errs
}

// parseDocAESKeys 解析 "keyID:hex,keyID:hex" 格式的密钥配置
func parseDocAESKeys(raw string) (map[uint8][]byte, []string) {
	var errs []string
	keys := make(map[uint8][]byte)

	entries := strings.Split(raw, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			errs = append(errs, fmt.Sprintf("DOC_AES_KEYS: invalid entry %q, expected format keyID:hex", entry))
			continue
		}

		idStr := strings.TrimSpace(parts[0])
		hexStr := strings.TrimSpace(parts[1])

		id, err := strconv.ParseUint(idStr, 10, 8)
		if err != nil || id < 1 || id > 255 {
			errs = append(errs, fmt.Sprintf("DOC_AES_KEYS: invalid key ID %q, must be 1-255", idStr))
			continue
		}
		keyID := uint8(id)

		if _, exists := keys[keyID]; exists {
			errs = append(errs, fmt.Sprintf("DOC_AES_KEYS: duplicate key ID %d", keyID))
			continue
		}

		keyBytes, err := hex.DecodeString(hexStr)
		if err != nil {
			errs = append(errs, fmt.Sprintf("DOC_AES_KEYS: invalid hex for key ID %d: %v", keyID, err))
			continue
		}
		if len(keyBytes) != 32 {
			errs = append(errs, fmt.Sprintf("DOC_AES_KEYS: key ID %d must be exactly 32 bytes (got %d)", keyID, len(keyBytes)))
			continue
		}

		keys[keyID] = keyBytes
	}

	if len(keys) == 0 && len(errs) == 0 {
		errs = append(errs, "DOC_AES_KEYS: no valid keys found")
	}

	return keys, errs
}
