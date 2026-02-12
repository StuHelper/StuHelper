// Package id 提供统一的 UUIDv7 主键生成
package id

import (
	"fmt"

	"github.com/google/uuid"
)

// New 生成一个 UUIDv7 字符串，用于内部业务表主键
func New() (string, error) {
	v, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate UUIDv7: %w", err)
	}
	return v.String(), nil
}
