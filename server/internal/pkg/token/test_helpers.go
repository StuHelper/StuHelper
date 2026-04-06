package token

import (
	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
)

// NewServiceForTest 创建用于测试的 Token 管理服务。
// Zitadel 架构下 Token Service 不再验证 JWT，只管理黑名单和 TTL。
func NewServiceForTest(rdb *redis.Client) *Service {
	_ = crypto.InitHMACKey("test-token-helper-secret", false) //nolint:errcheck // 测试初始化，幂等

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
