package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
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

type devBindPhoneOTPGenerator struct {
	user.OTPGenerator
	rdb *redis.Client
}

func newDevBindPhoneSMSSender(rdb *redis.Client) *devBindPhoneSMSSender {
	return &devBindPhoneSMSSender{rdb: rdb}
}

func newDevBindPhoneOTPGenerator(base user.OTPGenerator, rdb *redis.Client) user.OTPGenerator {
	return devBindPhoneOTPGenerator{
		OTPGenerator: base,
		rdb:          rdb,
	}
}

func (s *devBindPhoneSMSSender) Send(ctx context.Context, phone, content string) error {
	if s == nil || s.rdb == nil {
		return fmt.Errorf("development bind phone sms sender requires redis")
	}

	normalized := normalizeDevBindPhoneOTPPhone(phone)
	key := devBindPhoneOTPKey(phone)
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

func (g devBindPhoneOTPGenerator) Consume(ctx context.Context, phone, code string) error {
	if err := g.OTPGenerator.Consume(ctx, phone, code); err != nil {
		return err
	}
	if g.rdb == nil {
		return nil
	}
	key := devBindPhoneOTPKey(phone)
	if err := g.rdb.Del(ctx, key).Err(); err != nil {
		logger.L().Warn(
			"failed to remove development bind phone OTP capture",
			zap.String("phone", phoneutil.Mask(normalizeDevBindPhoneOTPPhone(phone))),
			zap.String("redisKey", key),
			zap.Error(err),
		)
	}
	return nil
}

func devBindPhoneOTPKey(phone string) string {
	return devBindPhoneOTPKeyPrefix + normalizeDevBindPhoneOTPPhone(phone)
}

func normalizeDevBindPhoneOTPPhone(phone string) string {
	trimmed := strings.TrimSpace(phone)
	trimmed = strings.TrimPrefix(trimmed, "+86")
	if phoneutil.IsValidMainlandPhone(trimmed) {
		return trimmed
	}
	return strings.TrimSpace(phone)
}
