//go:build wireinject
// +build wireinject

package wire

import (
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/auth"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/modules/course"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/health"
	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
	"github.com/google/wire"
)

// App 包含应用程序的所有依赖
type App struct {
	Config        *config.Config
	TokenService  *token.Service
	HealthHandler *health.Handler
	AuthHandler   *auth.Handler
	CourseHandler *course.Handler
	Cleanup       func()
}

// InitializeApp 初始化应用程序的所有依赖
func InitializeApp() (*App, error) {
	wire.Build(
		ConfigSet,
		InfraSet,
		ServiceSet,
		HandlerSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}
