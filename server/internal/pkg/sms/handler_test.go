package sms

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestDecodeInternalSendRequestRejectsLooseJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "unknown field",
			body: `{"phone":"+8613800138000","content":"123456","unused":true}`,
		},
		{
			name: "trailing json value",
			body: `{"phone":"+8613800138000","content":"123456"} {}`,
		},
		{
			name: "trailing garbage",
			body: `{"phone":"+8613800138000","content":"123456"} garbage`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/internal/sms/send", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			_, err := decodeInternalSendRequest(w, req)

			require.Error(t, err)
		})
	}
}

func TestDecodeInternalSendRequestAcceptsSingleJSONBody(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/internal/sms/send",
		strings.NewReader(`{"phone":"+8613800138000","content":"123456"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	got, err := decodeInternalSendRequest(w, req)

	require.NoError(t, err)
	assert.Equal(t, SendRequest{Phone: "+8613800138000", Content: "123456"}, got)
}
