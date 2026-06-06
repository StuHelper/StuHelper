package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

const (
	devBindPhoneOTPKeyPrefix = "dev:bind_phone_otp:"
	devBindPhoneOTPTTL       = 5 * time.Minute
)

type devBindPhoneSMSSender struct {
	rdb *redis.Client
}

func newDevBindPhoneSMSSender(rdb *redis.Client) *devBindPhoneSMSSender {
	return &devBindPhoneSMSSender{rdb: rdb}
}

func (s *devBindPhoneSMSSender) Send(ctx context.Context, phone, content string) error {
	if s == nil || s.rdb == nil {
		return fmt.Errorf("development bind phone sms sender requires redis")
	}

	normalized := normalizeDevBindPhoneOTPPhone(phone)
	key := devBindPhoneOTPKeyPrefix + normalized
	if err := s.rdb.Set(ctx, key, strings.TrimSpace(content), devBindPhoneOTPTTL).Err(); err != nil {
		return fmt.Errorf("store development bind phone otp: %w", err)
	}

	logger.L().Warn(
		"development bind phone OTP captured in Redis; do not enable this sender outside development",
		zap.String("phone", phoneutil.Mask(normalized)),
		zap.String("redisKey", key),
		zap.Duration("ttl", devBindPhoneOTPTTL),
	)
	return nil
}

func normalizeDevBindPhoneOTPPhone(phone string) string {
	trimmed := strings.TrimSpace(phone)
	trimmed = strings.TrimPrefix(trimmed, "+86")
	if phoneutil.IsValidMainlandPhone(trimmed) {
		return trimmed
	}
	return strings.TrimSpace(phone)
}
