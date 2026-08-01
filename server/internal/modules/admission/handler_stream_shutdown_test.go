package admission

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestBotAdmissionActionStreamStopsOnApplicationShutdown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	shutdownCtx, beginShutdown := context.WithCancel(context.Background())
	handler := NewHandler(
		svc,
		func(context.Context, string) (int64, error) { return 1, nil },
		nil,
		WithStreamShutdown(shutdownCtx),
	)

	router := gin.New()
	router.GET("/stream", handler.handleStreamBotAdmissionActions)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	requestCtx, cancelRequest := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancelRequest)
	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodGet,
		server.URL+"/stream?platform=qq&botSelfID=bot-shutdown",
		nil,
	)
	require.NoError(t, err)
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	assert.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))

	started := time.Now()
	beginShutdown()
	httpShutdownCtx, cancelHTTPShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelHTTPShutdown()
	require.NoError(t, server.Config.Shutdown(httpShutdownCtx))
	body := readSSEBodyAfterShutdown(t, response.Body)

	assert.Less(t, time.Since(started), 2*time.Second)
	assert.Contains(t, body, "event:end")
	assert.Contains(t, body, "data:shutdown")
}

func TestFreshmanCameraHandoffStreamStopsOnApplicationShutdown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedLinkedAdmissionUser(t, fixture, svc, "camera-stream-shutdown")
	application := createFreshmanTestApplication(t, svc, userID)
	handoff, err := svc.CreateFreshmanCameraHandoff(
		context.Background(),
		FreshmanCameraHandoffCreateInput{
			UserID:        userID,
			ApplicationID: application.ID,
		},
	)
	require.NoError(t, err)

	shutdownCtx, beginShutdown := context.WithCancel(context.Background())
	handler := NewHandler(
		svc,
		func(context.Context, string) (int64, error) { return userID, nil },
		nil,
		WithStreamShutdown(shutdownCtx),
	)

	router := gin.New()
	router.GET(
		"/camera/:id",
		func(c *gin.Context) {
			c.Set(middleware.CtxKeyUserID, "camera-stream-user")
			c.Next()
		},
		handler.handleWatchFreshmanCameraHandoff,
	)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	requestCtx, cancelRequest := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancelRequest)
	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodGet,
		server.URL+"/camera/"+handoff.ID,
		nil,
	)
	require.NoError(t, err)
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	assert.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))

	started := time.Now()
	beginShutdown()
	body := readSSEBodyAfterShutdown(t, response.Body)

	assert.Less(t, time.Since(started), 2*time.Second)
	assert.Contains(t, body, "event:end")
	assert.Contains(t, body, "data:shutdown")
}

func readSSEBodyAfterShutdown(t *testing.T, body io.Reader) string {
	t.Helper()
	type result struct {
		body []byte
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		content, err := io.ReadAll(body)
		resultCh <- result{body: content, err: err}
	}()

	select {
	case got := <-resultCh:
		require.NoError(t, got.err)
		return string(got.body)
	case <-time.After(3 * time.Second):
		t.Fatal("SSE handler did not return promptly after application shutdown")
		return ""
	}
}
