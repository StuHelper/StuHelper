package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// getEnv 获取环境变量，如果未设置则返回默认值。
// 使用 LookupEnv 区分"未设置"和"显式设为空字符串"。
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvSlice 获取逗号分隔的环境变量值
func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return defaultValue
}

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int, parseErrs *[]string) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	if intValue, err := strconv.Atoi(val); err == nil {
		return intValue
	}

	errMsg := fmt.Sprintf("invalid integer value for %s: %s", key, val)
	*parseErrs = append(*parseErrs, errMsg)
	return defaultValue
}

// getEnvInt32 获取 int32 类型的环境变量
func getEnvInt32(key string, defaultValue int32, parseErrs *[]string) int32 {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	intValue, err := strconv.ParseInt(val, 10, 32)
	if err == nil {
		return int32(intValue)
	}

	errMsg := fmt.Sprintf("invalid integer value for %s: %s", key, val)
	*parseErrs = append(*parseErrs, errMsg)
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool, parseErrs *[]string) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	if boolValue, err := strconv.ParseBool(val); err == nil {
		return boolValue
	}

	errMsg := fmt.Sprintf("invalid boolean value for %s: %s", key, val)
	*parseErrs = append(*parseErrs, errMsg)
	return defaultValue
}

// getEnvInt64 获取 int64 类型的环境变量
func getEnvInt64(key string, defaultValue int64, parseErrs *[]string) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	if intValue, err := strconv.ParseInt(val, 10, 64); err == nil {
		return intValue
	}

	errMsg := fmt.Sprintf("invalid int64 value for %s: %s", key, val)
	*parseErrs = append(*parseErrs, errMsg)
	return defaultValue
}

// getEnvFloat64 获取 float64 类型的环境变量
func getEnvFloat64(key string, defaultValue float64, parseErrs *[]string) float64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	if floatValue, err := strconv.ParseFloat(val, 64); err == nil {
		return floatValue
	}

	errMsg := fmt.Sprintf("invalid float64 value for %s: %s", key, val)
	*parseErrs = append(*parseErrs, errMsg)
	return defaultValue
}
