package auth

import (
	"context"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/usersync"
)

// UserSyncInput 认证同步输入，别名到 usersync.Input。
type UserSyncInput = usersync.Input

// UserSyncRepo 定义 auth domain 所需的用户同步窄接口。
// 实现位于 user.UserSyncRepository。
type UserSyncRepo interface {
	UpsertUser(ctx context.Context, input UserSyncInput) error
	ExistsByCasdoorSubject(ctx context.Context, casdoorSubject string) (bool, error)
}
