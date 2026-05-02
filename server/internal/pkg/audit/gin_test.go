package audit

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

func TestEventFromGin(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", strings.Repeat("ua", 256))
	req.RemoteAddr = "203.0.113.9:1234"
	c.Request = req
	c.Set(middleware.CtxKeyUserID, "user-1")
	c.Set(middleware.CtxKeyUsername, "alice")
	c.Set(middleware.CtxKeyRequestID, "req-1")

	event := EventFromGin(c, Event{
		Type:     EventDataUpdate,
		Resource: "school_config",
		Action:   "update",
	})

	assert.Equal(t, "user-1", event.UserID)
	assert.Equal(t, "alice", event.Username)
	assert.Equal(t, "203.0.113.9", event.IP)
	assert.Equal(t, "req-1", event.RequestID)
	assert.NotEmpty(t, event.UserAgent)
	assert.LessOrEqual(t, len(event.UserAgent), 512)
}

func TestEventFromContextUsesLoggerRequestID(t *testing.T) {
	t.Parallel()

	ctx := logger.WithRequestID(context.Background(), "req-context-1")
	event := EventFromContext(ctx, Event{Type: EventUserLogin, Result: "success"})

	assert.Equal(t, "req-context-1", event.RequestID)
}
