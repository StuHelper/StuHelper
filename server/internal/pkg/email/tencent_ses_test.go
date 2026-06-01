package email

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTencentSESSenderSendOTPUsesApprovedTemplateData(t *testing.T) {
	var captured tencentSESSendEmailRequest
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-1","MessageId":"msg-1"}}`))
	}))
	defer server.Close()

	sender, err := NewTencentSESSender(TencentSESConfig{
		SecretID:       "AKIDEXAMPLE",
		SecretKey:      "secret-example",
		Region:         "ap-guangzhou",
		Endpoint:       server.URL,
		From:           "noreply@notify.stuhelper.com",
		FromName:       "StuHelper",
		TemplateID:     49779,
		DefaultPurpose: "学校邮箱认证",
		DefaultSchool:  "北京航空航天大学",
		DefaultExpire:  5,
	})
	require.NoError(t, err)
	sender.now = func() time.Time { return time.Unix(1767225600, 0).UTC() }

	err = sender.SendOTP(
		context.Background(),
		"student@buaa.edu.cn",
		"StuHelper 学校邮箱验证码",
		"123456",
		"",
		"",
		0,
	)
	require.NoError(t, err)

	assert.Equal(t, "StuHelper <noreply@notify.stuhelper.com>", captured.FromEmailAddress)
	assert.Equal(t, []string{"student@buaa.edu.cn"}, captured.Destination)
	assert.Equal(t, int64(49779), captured.Template.TemplateID)
	assert.Equal(t, "StuHelper 学校邮箱验证码", captured.Subject)

	var templateData map[string]string
	require.NoError(t, json.Unmarshal([]byte(captured.Template.TemplateData), &templateData))
	assert.Equal(t, map[string]string{
		"code":           "123456",
		"expire_minutes": "5",
		"purpose":        "学校邮箱认证",
		"school_name":    "北京航空航天大学",
	}, templateData)

	assert.Equal(t, "SendEmail", capturedHeaders.Get("X-TC-Action"))
	assert.Equal(t, "2020-10-02", capturedHeaders.Get("X-TC-Version"))
	assert.Equal(t, "ap-guangzhou", capturedHeaders.Get("X-TC-Region"))
	assert.Equal(t, "1767225600", capturedHeaders.Get("X-TC-Timestamp"))
	assert.Contains(t, capturedHeaders.Get("Authorization"), "TC3-HMAC-SHA256 Credential=AKIDEXAMPLE/2026-01-01/ses/tc3_request")
}

func TestTencentSESSenderSendOTPReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":{"Error":{"Code":"FailedOperation.TemplateNotApproved","Message":"template not approved"},"RequestId":"req-1"}}`))
	}))
	defer server.Close()

	sender, err := NewTencentSESSender(TencentSESConfig{
		SecretID:   "AKIDEXAMPLE",
		SecretKey:  "secret-example",
		Endpoint:   server.URL,
		From:       "noreply@notify.stuhelper.com",
		TemplateID: 49779,
	})
	require.NoError(t, err)

	err = sender.SendOTP(context.Background(), "student@buaa.edu.cn", "subject", "123456", "", "", 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "FailedOperation.TemplateNotApproved")
	assert.Contains(t, err.Error(), "template not approved")
}
