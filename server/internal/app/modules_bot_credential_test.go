package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/admission"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

// F023 回归：BOT_SERVICE_TOKEN 为空时 initBotCredentialVerifier 必须返回真 nil 接口。
// 若回退为返回具体类型 *serviceaccount.Verifier 的 nil 指针，装入下游
// admission.BotCredentialVerifier 接口后 nil 守卫失效，带 Bearer 头的
// /api/v1/bot/** 请求会触发 nil 接收者 panic。
func TestInitBotCredentialVerifierReturnsTrueNilInterfaceWhenUnconfigured(t *testing.T) {
	rt := &Runtime{cfg: &config.Config{}}

	verifier, err := rt.initBotCredentialVerifier(context.Background())

	require.NoError(t, err)
	// 必须用 == nil 判定：testify 的 Nil 断言基于反射，typed-nil 也会通过，无法捕获回归。
	require.True(t, verifier == nil, "initBotCredentialVerifier 未配置时必须返回真 nil 接口")
	var downstream admission.BotCredentialVerifier = verifier
	assert.True(t, downstream == nil, "typed-nil 装入下游接口后必须仍为 nil，否则 handler nil 守卫失效")
}
