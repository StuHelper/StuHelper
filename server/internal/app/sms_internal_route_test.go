package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/sms"
)

func TestRegisterSMSInternalRouteUsesMainRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	smsService := sms.NewService(sms.Config{InternalKey: "expected-key"}, zap.NewNop())
	registerSMSInternalRoute(router, smsService)

	req := httptest.NewRequest(http.MethodPost, "/internal/sms/send", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
