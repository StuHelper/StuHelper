package token

import (
	"github.com/redis/go-redis/v9"

	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
)

// NewServiceForTest 创建用于测试的 Token 管理服务。
// 此函数仅供 _test.go 文件调用，不应被生产代码引用。
func NewServiceForTest(rdb *redis.Client) *Service {
	if err := crypto.InitHMACKey("test-token-helper-secret", false); err != nil {
		panic("NewServiceForTest: InitHMACKey: " + err.Error())
	}

	svc, err := NewService(ServiceConfig{
		RedisClient: rdb,
		AccessTTL:   3600,
		RefreshTTL:  86400,
	})
	if err != nil {
		panic("NewServiceForTest: " + err.Error())
	}
	return svc
}
