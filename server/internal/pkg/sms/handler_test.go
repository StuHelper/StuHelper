package sms

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestInternalHandlerAcceptsCasdoorFormWithQueryInternalKey(t *testing.T) {
	service := NewService(Config{InternalKey: "expected-key"}, zap.NewNop())
	mux := http.NewServeMux()
	service.RegisterInternalHandler(mux)

	req := httptest.NewRequest(
		http.MethodPost,
		"/internal/sms/send?internal_key=expected-key",
		strings.NewReader("phoneNumber=%2B8613800138000&content=123456"),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "failed to send SMS")
}

func TestInternalHandlerAcceptsCasdoorFormWithBearerInternalKey(t *testing.T) {
	service := NewService(Config{InternalKey: "expected-key"}, zap.NewNop())
	mux := http.NewServeMux()
	service.RegisterInternalHandler(mux)

	req := httptest.NewRequest(
		http.MethodPost,
		"/internal/sms/send",
		strings.NewReader("phoneNumber=%2B8613800138000&content=123456"),
	)
	req.Header.Set("Authorization", "Bearer expected-key")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "failed to send SMS")
}
