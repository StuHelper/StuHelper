package sms

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewService_DefaultsRegionToBeijing(t *testing.T) {
	svc := NewService(Config{
		SecretID:  "test-id",
		SecretKey: "test-key",
	}, zap.NewNop())

	assert.Equal(t, "ap-beijing", svc.cfg.Region)
}

func TestNewService_UsesCustomRegion(t *testing.T) {
	svc := NewService(Config{
		SecretID:  "test-id",
		SecretKey: "test-key",
		Region:    "ap-shanghai",
	}, zap.NewNop())

	assert.Equal(t, "ap-shanghai", svc.cfg.Region)
}

func TestMaskPhone_ShortPhone(t *testing.T) {
	assert.Equal(t, "***", maskPhone("123"))
	assert.Equal(t, "***", maskPhone("1234567"))
}

func TestMaskPhone_FullPhone(t *testing.T) {
	assert.Equal(t, "+86****8000", maskPhone("+8613800138000"))
}

func TestSend_ReturnsErrorWhenCredentialsMissing(t *testing.T) {
	svc := NewService(Config{}, zap.NewNop())
	err := svc.Send(context.TODO(), "+8613800138000", "123456")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credentials not configured")
}

func TestSignV3_ProducesValidFormat(t *testing.T) {
	svc := NewService(Config{
		SecretID:  "AKIDz8krbsJ5yKBZQpn74WFkmLPx3EXAMPLE",
		SecretKey: "Gu5t9xGARNpq86cd98joQYCN3EXAMPLE",
	}, zap.NewNop())

	ts := time.Unix(1551113065, 0).UTC()
	auth := svc.signV3("sms.tencentcloudapi.com", ts, "1551113065", []byte("{}"))
	assert.Contains(t, auth, "TC3-HMAC-SHA256")
	assert.Contains(t, auth, "AKIDz8krbsJ5yKBZQpn74WFkmLPx3EXAMPLE")
}
