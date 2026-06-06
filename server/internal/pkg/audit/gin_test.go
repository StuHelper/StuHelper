package audit

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
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
	assert.Equal(t, "user", event.ActorType)
	assert.NotEmpty(t, event.UserAgent)
	assert.LessOrEqual(t, len(event.UserAgent), 512)
}

func TestEventFromGinInfersAdminActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Set(middleware.CtxKeyUserID, "admin-1")
	c.Set(middleware.CtxKeyCapabilitySet, map[string]struct{}{
		capability.AdminEntryCapabilities[0]: {},
	})

	event := EventFromGin(c, Event{Type: EventAdminConfigChange})

	assert.Equal(t, "admin-1", event.UserID)
	assert.Equal(t, "admin", event.ActorType)
}

func TestEventFromGinHandlesNilContextAndRequest(t *testing.T) {
	var event Event
	assert.NotPanics(t, func() {
		event = EventFromGin(nil, Event{Type: EventDataAccess})
	})
	assert.Equal(t, "system", event.ActorType)

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(middleware.CtxKeyUserID, "user-without-request")
	c.Set(middleware.CtxKeyRequestID, "req-without-request")

	assert.NotPanics(t, func() {
		event = EventFromGin(c, Event{Type: EventDataAccess})
	})
	assert.Equal(t, "user-without-request", event.UserID)
	assert.Equal(t, "req-without-request", event.RequestID)
	assert.Equal(t, "user", event.ActorType)
}

func TestEventFromContextUsesLoggerRequestID(t *testing.T) {
	t.Parallel()

	ctx := logger.WithRequestID(context.Background(), "req-context-1")
	event := EventFromContext(ctx, Event{Type: EventUserLogin, Result: "success"})

	assert.Equal(t, "req-context-1", event.RequestID)
}
