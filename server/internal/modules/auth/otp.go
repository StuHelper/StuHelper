package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/ctxutil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

// OTP 配置常量
const (
	otpLength      = 6                // 验证码位数
	otpTTL         = 5 * time.Minute  // 验证码有效期
	otpCooldown    = 60 * time.Second // 发送冷却期
	otpMaxAttempts = 5                // 最大验证尝试次数
	otpPhoneLimit  = 5                // 单个手机号每小时最大请求次数
	otpPhoneTTL    = 1 * time.Hour    // 单个手机号限流窗口
	otpCompTimeout = 5 * time.Second  // 发送失败补偿超时
)

// OTP Redis key 前缀
const (
	otpCodePrefix       = "otp:code:"        // otp:code:{phone_hash} → code
	otpCooldownPrefix   = "otp:cd:"          // otp:cd:{phone_hash} → 1
	otpAttemptsPrefix   = "otp:attempts:"    // otp:attempts:{phone_hash} → count
	otpPhoneLimitPrefix = "otp:phone_limit:" // otp:phone_limit:{phone_hash} → count
)

// OTP 错误
var (
	ErrOTPCooldown         = errors.New("otp: please wait before requesting a new code")
	ErrOTPInvalidCode      = errors.New("otp: invalid verification code")
	ErrOTPExpired          = errors.New("otp: verification code expired")
	ErrOTPMaxAttempts      = errors.New("otp: too many failed attempts")
	ErrOTPPhoneRateLimited = errors.New("otp: too many requests for this phone number")
	ErrOTPInvalidPhone     = errors.New("otp: invalid phone number")
)

// OTPCooldownSeconds 返回 OTP 冷却期秒数，供外部模块在响应中使用。
func OTPCooldownSeconds() int {
	return int(otpCooldown.Seconds())
}

// PhoneSMSSender 发送 OTP 短信所需的最小依赖。
type PhoneSMSSender interface {
	Send(ctx context.Context, phone, content string) error
}

// OTPService 短信验证码管理服务
type OTPService struct {
	rdb *redis.Client
}

var otpFailureScript = redis.NewScript(`
local attempts = redis.call("INCR", KEYS[1])
if attempts == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
if attempts >= tonumber(ARGV[2]) then
  redis.call("DEL", KEYS[2], KEYS[1])
end
return attempts
`)

var otpPhoneLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

var otpPhoneLimitRollbackScript = redis.NewScript(`
local count = tonumber(redis.call("GET", KEYS[1]) or "0")
if count <= 1 then
  redis.call("DEL", KEYS[1])
  return 0
end
return redis.call("DECR", KEYS[1])
`)

var otpConsumeCheckedScript = redis.NewScript(`
local stored = redis.call("GET", KEYS[1])
if not stored then
  return 0
end
if stored ~= ARGV[1] then
  return 0
end
redis.call("DEL", KEYS[1], KEYS[2])
return 1
`)

// NewOTPService 创建 OTP 服务
func NewOTPService(rdb *redis.Client) *OTPService {
	return &OTPService{rdb: rdb}
}

// CooldownSeconds 返回 OTP 冷却期秒数，供 handler 响应使用。
func (s *OTPService) CooldownSeconds() int {
	return OTPCooldownSeconds()
}

func otpPhoneKey(phone string) (string, error) {
	normalized, err := normalizeOTPPhone(phone)
	if err != nil {
		return "", err
	}
	hash, err := phoneutil.HashLookup(normalized)
	if err != nil {
		return "", fmt.Errorf("otp: hash phone: %w", err)
	}
	return hash, nil
}

// CheckPhoneRateLimit 对单个手机号执行每小时限流，防止分布式来源对同一手机号刷 OTP。
func (s *OTPService) CheckPhoneRateLimit(ctx context.Context, phone string) error {
	phoneKey, err := otpPhoneKey(phone)
	if err != nil {
		return err
	}

	limitKey := otpPhoneLimitPrefix + phoneKey
	count, err := otpPhoneLimitScript.Run(ctx, s.rdb, []string{limitKey}, otpPhoneTTL.Milliseconds()).Int64()
	if err != nil {
		return fmt.Errorf("otp: check phone rate limit: %w", err)
	}
	if count > int64(otpPhoneLimit) {
		return ErrOTPPhoneRateLimited
	}
	return nil
}

// IssueCode 执行完整的 OTP 发送流程：手机号限流、生成验证码、发送短信与失败补偿。
func (s *OTPService) IssueCode(ctx context.Context, phone string, smsSender PhoneSMSSender) error {
	if smsSender == nil {
		return errors.New("otp: sms sender is required")
	}
	normalizedPhone, err := normalizeOTPPhone(phone)
	if err != nil {
		return err
	}
	if err := s.CheckPhoneRateLimit(ctx, normalizedPhone); err != nil {
		return err
	}

	code, err := s.Generate(ctx, normalizedPhone)
	if err != nil {
		rollbackCtx, cancel := otpCompensationContext(ctx)
		defer cancel()
		if rollbackErr := s.rollbackPhoneRateLimit(rollbackCtx, normalizedPhone); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}

	internationalPhone := "+86" + normalizedPhone
	if err := smsSender.Send(ctx, internationalPhone, code); err != nil {
		sendErr := fmt.Errorf("otp: send sms: %w", err)
		compCtx, cancel := otpCompensationContext(ctx)
		defer cancel()
		cleanupErr := s.CleanupCodeOnly(compCtx, normalizedPhone)
		rollbackErr := s.rollbackPhoneRateLimit(compCtx, normalizedPhone)
		if cleanupErr != nil || rollbackErr != nil {
			joined := sendErr
			if cleanupErr != nil {
				joined = errors.Join(joined, fmt.Errorf("otp: cleanup code after send failure: %w", cleanupErr))
			}
			if rollbackErr != nil {
				joined = errors.Join(joined, rollbackErr)
			}
			return joined
		}
		return sendErr
	}
	return nil
}

func otpCompensationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctxutil.DetachedTimeout(ctx, otpCompTimeout)
}

func (s *OTPService) rollbackPhoneRateLimit(ctx context.Context, phone string) error {
	phoneKey, err := otpPhoneKey(phone)
	if err != nil {
		return err
	}

	limitKey := otpPhoneLimitPrefix + phoneKey
	if _, err := otpPhoneLimitRollbackScript.Run(ctx, s.rdb, []string{limitKey}).Int64(); err != nil {
		return fmt.Errorf("otp: rollback phone rate limit: %w", err)
	}
	return nil
}

// Generate 生成并存储验证码，返回明文验证码（用于 SMS 发送）。
// 如果在冷却期内，返回 ErrOTPCooldown。
func (s *OTPService) Generate(ctx context.Context, phone string) (string, error) {
	phoneKey, err := otpPhoneKey(phone)
	if err != nil {
		return "", err
	}
	cooldownKey := otpCooldownPrefix + phoneKey

	// 原子性冷却期检查：SetNX 仅在 key 不存在时设置成功，消除 TOCTOU 竞态
	setResult, err := s.rdb.SetArgs(ctx, cooldownKey, "1", redis.SetArgs{
		Mode: "NX",
		TTL:  otpCooldown,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrOTPCooldown
	}
	if err != nil {
		return "", fmt.Errorf("otp: check cooldown: %w", err)
	}
	if setResult != "OK" {
		return "", ErrOTPCooldown
	}

	// 生成随机验证码
	code, err := generateNumericCode(otpLength)
	if err != nil {
		return "", fmt.Errorf("otp: generate code: %w", err)
	}

	codeKey := otpCodePrefix + phoneKey
	attemptsKey := otpAttemptsPrefix + phoneKey

	// 使用 pipeline 原子性地设置验证码和重置尝试次数（冷却期已由 SetNX 设置）
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, codeKey, code, otpTTL)
	pipe.Del(ctx, attemptsKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("otp: store code: %w", err)
	}

	return code, nil
}

// Cleanup 删除指定手机号的验证码、冷却期和尝试计数。
// 用于验证码过期或完全重置场景。
func (s *OTPService) Cleanup(ctx context.Context, phone string) error {
	phoneKey, err := otpPhoneKey(phone)
	if err != nil {
		return err
	}
	codeKey := otpCodePrefix + phoneKey
	cooldownKey := otpCooldownPrefix + phoneKey
	attemptsKey := otpAttemptsPrefix + phoneKey

	if err := s.rdb.Del(ctx, codeKey, cooldownKey, attemptsKey).Err(); err != nil {
		return fmt.Errorf("otp: cleanup code: %w", err)
	}

	return nil
}

// CleanupCodeOnly 仅删除验证码，保留冷却期和尝试计数。
// 用于短信发送失败后的补偿，避免攻击者借由发送失败绕过冷却窗口。
func (s *OTPService) CleanupCodeOnly(ctx context.Context, phone string) error {
	phoneKey, err := otpPhoneKey(phone)
	if err != nil {
		return err
	}

	codeKey := otpCodePrefix + phoneKey
	if err := s.rdb.Del(ctx, codeKey).Err(); err != nil {
		return fmt.Errorf("otp: cleanup code only: %w", err)
	}
	return nil
}

// Check 验证验证码是否正确；成功时不删除验证码，方便调用方在本地提交成功后再显式消费。
func (s *OTPService) Check(ctx context.Context, phone, code string) error {
	phoneKey, err := otpPhoneKey(phone)
	if err != nil {
		return err
	}
	normalizedCode, err := normalizeOTPCode(code)
	if err != nil {
		return err
	}
	attemptsKey := otpAttemptsPrefix + phoneKey
	codeKey := otpCodePrefix + phoneKey

	// 获取存储的验证码
	stored, err := s.rdb.Get(ctx, codeKey).Result()
	if errors.Is(err, redis.Nil) {
		return ErrOTPExpired
	}
	if err != nil {
		return fmt.Errorf("otp: get code: %w", err)
	}

	if subtle.ConstantTimeCompare([]byte(stored), []byte(normalizedCode)) != 1 {
		attempts, attemptErr := otpFailureScript.Run(
			ctx,
			s.rdb,
			[]string{attemptsKey, codeKey},
			otpTTL.Milliseconds(),
			otpMaxAttempts,
		).Int64()
		if attemptErr != nil {
			return fmt.Errorf("otp: record failed attempt: %w", attemptErr)
		}
		if attempts >= int64(otpMaxAttempts) {
			return ErrOTPMaxAttempts
		}
		return ErrOTPInvalidCode
	}

	return nil
}

// Consume 删除仍匹配本次验证码的 code 和尝试计数。
// 调用方应在依赖该验证码的业务提交成功后调用；如果期间用户申请了新验证码，
// 这里会保留新验证码，避免旧请求误删后续凭据。
func (s *OTPService) Consume(ctx context.Context, phone, code string) error {
	phoneKey, err := otpPhoneKey(phone)
	if err != nil {
		return err
	}
	normalizedCode, err := normalizeOTPCode(code)
	if err != nil {
		return err
	}
	codeKey := otpCodePrefix + phoneKey
	attemptsKey := otpAttemptsPrefix + phoneKey
	if _, delErr := otpConsumeCheckedScript.Run(ctx, s.rdb, []string{codeKey, attemptsKey}, normalizedCode).Int64(); delErr != nil {
		return fmt.Errorf("otp: consume code: %w", delErr)
	}
	return nil
}

// Verify 验证验证码。成功后自动删除，确保一次性使用。
func (s *OTPService) Verify(ctx context.Context, phone, code string) error {
	if err := s.Check(ctx, phone, code); err != nil {
		return err
	}
	return s.Consume(ctx, phone, code)
}

// generateNumericCode 生成指定位数的随机数字验证码
func generateNumericCode(length int) (string, error) {
	maxValue := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)

	n, err := rand.Int(rand.Reader, maxValue)
	if err != nil {
		return "", fmt.Errorf("generate random: %w", err)
	}

	// 补齐前导零
	format := fmt.Sprintf("%%0%dd", length)
	return fmt.Sprintf(format, n), nil
}

func normalizeOTPCode(code string) (string, error) {
	normalized := strings.TrimSpace(code)
	if utf8.RuneCountInString(normalized) != otpLength {
		return "", ErrOTPInvalidCode
	}
	return normalized, nil
}

func normalizeOTPPhone(phone string) (string, error) {
	normalized := strings.TrimSpace(phone)
	if !phoneutil.IsValidMainlandPhone(normalized) {
		return "", ErrOTPInvalidPhone
	}
	return normalized, nil
}
