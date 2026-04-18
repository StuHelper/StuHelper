package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sms"
)

type rewriteTargetTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t *rewriteTargetTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = t.target.Scheme
	cloned.URL.Host = t.target.Host
	return t.base.RoundTrip(cloned)
}

func setUnexportedField(target any, field string, value any) {
	rv := reflect.ValueOf(target).Elem().FieldByName(field)
	reflect.NewAt(rv.Type(), unsafe.Pointer(rv.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func newTestSMSService(t *testing.T, handler http.HandlerFunc) *sms.Service {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	target, err := url.Parse(server.URL)
	require.NoError(t, err)

	client := server.Client()
	client.Transport = &rewriteTargetTransport{
		target: target,
		base:   client.Transport,
	}

	svc := sms.NewService(sms.Config{
		SecretID:   "sid",
		SecretKey:  "skey",
		AppID:      "app",
		SignName:   "StuHelper",
		TemplateID: "tpl",
	}, zap.NewNop())
	setUnexportedField(svc, "client", client)
	return svc
}

func TestRequestPhoneOTP_SuccessAndSMSFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newRefreshTestHandler(t, &fakeUserSyncRepo{})
	h.otpService = NewOTPService(h.redisClient)

	var requests int
	h.smsService = newTestSMSService(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, []any{"+8613800138000"}, payload["PhoneNumberSet"])
		require.Equal(t, "app", payload["SmsSdkAppId"])
		require.Equal(t, "StuHelper", payload["SignName"])
		require.Equal(t, "tpl", payload["TemplateId"])
		require.Len(t, payload["TemplateParamSet"], 1)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":{"SendStatusSet":[{"Code":"Ok","Message":"send ok"}]}}`))
	})

	r := gin.New()
	r.POST("/otp/request", h.RequestPhoneOTP)

	body := bytes.NewBufferString(`{"phone":"13800138000"}`)
	req := httptest.NewRequest(http.MethodPost, "/otp/request", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "verification code sent")
	assert.Contains(t, w.Body.String(), `"cooldown":60`)
	assert.Equal(t, 1, requests)

	// cooldown branch
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewBufferString(`{"phone":"13800138000"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "please wait before requesting a new code")

	// cleanup then hit SMS send failure branch; OTP code should be cleaned while cooldown remains.
	require.NoError(t, h.otpService.Cleanup(context.Background(), "13800138000"))
	h.smsService = newTestSMSService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":{"Error":{"Code":"InternalError","Message":"boom"}}}`))
	})

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewBufferString(`{"phone":"13800138000"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to send verification code")

	// code is cleaned up, so verify should report expired instead of invalid.
	verifyBody := bytes.NewBufferString(`{"phone":"13800138000","code":"000000"}`)
	verifyReq := httptest.NewRequest(http.MethodPost, "/otp/verify", verifyBody)
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyW := httptest.NewRecorder()
	rVerify := gin.New()
	rVerify.POST("/otp/verify", h.VerifyPhoneOTP)
	rVerify.ServeHTTP(verifyW, verifyReq)
	require.Equal(t, http.StatusUnauthorized, verifyW.Code)
	assert.Contains(t, verifyW.Body.String(), "verification code expired")
}

func TestRequestPhoneOTP_PhoneRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newRefreshTestHandler(t, &fakeUserSyncRepo{})
	h.otpService = NewOTPService(h.redisClient)
	h.smsService = newTestSMSService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":{"SendStatusSet":[{"Code":"Ok","Message":"send ok"}]}}`))
	})

	r := gin.New()
	r.POST("/otp/request", h.RequestPhoneOTP)

	for i := 0; i < otpPhoneLimit; i++ {
		require.NoError(t, h.otpService.Cleanup(context.Background(), "13800138001"))
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewBufferString(`{"phone":"13800138001"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "attempt %d failed: %s", i+1, w.Body.String())
	}

	// one more request should be rejected by per-phone limit before code generation.
	require.NoError(t, h.otpService.Cleanup(context.Background(), "13800138001"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewBufferString(`{"phone":"13800138001"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "too many requests for this phone number")
}
