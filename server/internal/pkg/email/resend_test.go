package email

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResendSenderSendOTPUsesHTMLAndIdempotency(t *testing.T) {
	var captured resendSendEmailRequest
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/emails", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"email-123"}`))
	}))
	defer server.Close()

	sender, err := NewResendSender(ResendConfig{
		APIKey:   "re_test",
		Endpoint: server.URL,
		From:     "noreply@notify.stuhelper.com",
		FromName: "StuHelper 系统邮件",
	})
	require.NoError(t, err)

	err = sender.SendOTP(
		context.Background(),
		"student@buaa.edu.cn",
		"学生认证验证码",
		"123456",
		"学校邮箱认证",
		"北京航空航天大学",
		5,
	)
	require.NoError(t, err)

	assert.Equal(t, "Bearer re_test", capturedHeaders.Get("Authorization"))
	assert.Equal(t, "application/json", capturedHeaders.Get("Content-Type"))
	assert.NotEmpty(t, capturedHeaders.Get("Idempotency-Key"))
	assert.Equal(t, "StuHelper 系统邮件 <noreply@notify.stuhelper.com>", captured.From)
	assert.Equal(t, []string{"student@buaa.edu.cn"}, captured.To)
	assert.Equal(t, "学生认证验证码", captured.Subject)
	assert.Contains(t, captured.HTML, "123456")
	assert.Contains(t, captured.HTML, "北京航空航天大学")
	assert.Contains(t, captured.Text, "123456")
}

func TestResendOTPBodyMatchesTencentTemplate(t *testing.T) {
	replacements := strings.NewReplacer(
		"{{code}}", "123456",
		"{{purpose}}", "学校邮箱认证",
		"{{school_name}}", "北京航空航天大学",
		"{{expire_minutes}}", "5",
	)
	templateDir := filepath.Join("..", "..", "..", "..", "infra", "email-templates", "tencent-ses")
	htmlTemplate, err := os.ReadFile(filepath.Join(templateDir, "stuhelper-school-email-otp.html"))
	require.NoError(t, err)
	textTemplate, err := os.ReadFile(filepath.Join(templateDir, "stuhelper-school-email-otp.txt"))
	require.NoError(t, err)

	assert.Equal(t,
		replacements.Replace(string(htmlTemplate)),
		buildOTPHTML("123456", "学校邮箱认证", "北京航空航天大学", 5),
	)
	assert.Equal(t,
		replacements.Replace(string(textTemplate)),
		buildOTPText("123456", "学校邮箱认证", "北京航空航天大学", 5),
	)
}

func TestResendSenderSendOTPReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"statusCode":429,"name":"rate_limit_exceeded","message":"too many requests"}`))
	}))
	defer server.Close()

	sender, err := NewResendSender(ResendConfig{
		APIKey:   "re_test",
		Endpoint: server.URL,
		From:     "noreply@notify.stuhelper.com",
	})
	require.NoError(t, err)

	err = sender.SendOTP(context.Background(), "student@buaa.edu.cn", "学生认证验证码", "123456", "", "", 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate_limit_exceeded")
	assert.Contains(t, err.Error(), "too many requests")
}
