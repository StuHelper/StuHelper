package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

// OTP 配置常量
const (
	otpLength      = 6                // 验证码位数
	otpTTL         = 5 * time.Minute  // 验证码有效期
	otpCooldown    = 60 * time.Second // 发送冷却期
	otpMaxAttempts = 5                // 最大验证尝试次数
)

// OTP Redis key 前缀
const (
	otpCodePrefix     = "otp:code:"     // otp:code:{phone_hash} → code
	otpCooldownPrefix = "otp:cd:"       // otp:cd:{phone_hash} → 1
	otpAttemptsPrefix = "otp:attempts:" // otp:attempts:{phone_hash} → count
)

// OTP 错误
var (
	ErrOTPCooldown    = errors.New("otp: please wait before requesting a new code")
	ErrOTPInvalidCode = errors.New("otp: invalid verification code")
	ErrOTPExpired     = errors.New("otp: verification code expired")
	ErrOTPMaxAttempts = errors.New("otp: too many failed attempts")
)

// OTPCooldownSeconds 返回 OTP 冷却期秒数，供外部模块在响应中使用。
func OTPCooldownSeconds() int {
	return int(otpCooldown.Seconds())
}

// OTPService 短信验证码管理服务
type OTPService struct {
	rdb *redis.Client
}

var otpAttemptsScript = redis.NewScript(`
local attempts = redis.call("INCR", KEYS[1])
if attempts == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return attempts
`)

// NewOTPService 创建 OTP 服务
func NewOTPService(rdb *redis.Client) *OTPService {
	return &OTPService{rdb: rdb}
}

func otpPhoneKey(phone string) (string, error) {
	hash, err := phoneutil.HashLookup(phone)
	if err != nil {
		return "", fmt.Errorf("otp: hash phone: %w", err)
	}
	return hash, nil
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
	set, err := s.rdb.SetNX(ctx, cooldownKey, "1", otpCooldown).Result()
	if err != nil {
		return "", fmt.Errorf("otp: check cooldown: %w", err)
	}
	if !set {
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
// 用于短信发送失败后的补偿回滚，避免“未收到短信却被冷却”的情况。
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

// Verify 验证验证码。成功后自动删除，确保一次性使用。
func (s *OTPService) Verify(ctx context.Context, phone, code string) error {
	phoneKey, err := otpPhoneKey(phone)
	if err != nil {
		return err
	}
	attemptsKey := otpAttemptsPrefix + phoneKey
	codeKey := otpCodePrefix + phoneKey

	// 检查尝试次数（Lua 原子执行 INCR + 首次 EXPIRE）
	attempts, err := otpAttemptsScript.Run(ctx, s.rdb, []string{attemptsKey}, otpTTL.Milliseconds()).Int64()
	if err != nil {
		return fmt.Errorf("otp: check attempts: %w", err)
	}
	if attempts > int64(otpMaxAttempts) {
		// 超过最大尝试次数，删除验证码
		if delErr := s.rdb.Del(ctx, codeKey).Err(); delErr != nil {
			return fmt.Errorf("otp: delete code after max attempts: %w", delErr)
		}
		return ErrOTPMaxAttempts
	}

	// 获取存储的验证码
	stored, err := s.rdb.Get(ctx, codeKey).Result()
	if errors.Is(err, redis.Nil) {
		return ErrOTPExpired
	}
	if err != nil {
		return fmt.Errorf("otp: get code: %w", err)
	}

	if subtle.ConstantTimeCompare([]byte(stored), []byte(code)) != 1 {
		return ErrOTPInvalidCode
	}

	// 验证成功，删除验证码和尝试计数
	if delErr := s.rdb.Del(ctx, codeKey, attemptsKey).Err(); delErr != nil {
		return fmt.Errorf("otp: cleanup after verify: %w", delErr)
	}
	return nil
}

// generateNumericCode 生成指定位数的随机数字验证码
func generateNumericCode(length int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generate random: %w", err)
	}

	// 补齐前导零
	format := fmt.Sprintf("%%0%dd", length)
	return fmt.Sprintf(format, n), nil
}
