package openplatform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/StuHelper/StuHelper/server/internal/pkg/metrics"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
)

const (
	disclosureRateLimitDimensionApp      = "app"
	disclosureRateLimitDimensionAppUser  = "app_user"
	disclosureRateLimitDimensionEndpoint = "endpoint"
	disclosureRateLimitDimensionConsent  = "consent"
)

type DisclosureRateLimitConfig struct {
	AppLimit            int
	AppUserLimit        int
	EndpointLimit       int
	ConsentLimit        int
	ReplayLimit         int
	Window              time.Duration
	ReplayWindow        time.Duration
	ReplayAuditCooldown time.Duration
}

type disclosureRateLimiter struct {
	app      *middleware.RedisRateLimiter
	appUser  *middleware.RedisRateLimiter
	endpoint *middleware.RedisRateLimiter
	consent  *middleware.RedisRateLimiter
}

type disclosureReplayDetector struct {
	rdb           *redis.Client
	limit         int
	window        time.Duration
	auditCooldown time.Duration
}

type disclosureReplayDetection struct {
	Detected      bool
	SignatureHash string
	Count         int
}

type DisclosureRateLimitExceededError struct {
	Dimension string
}

func (e DisclosureRateLimitExceededError) Error() string {
	if e.Dimension == "" {
		return ErrDisclosureRateLimited.Error()
	}
	return ErrDisclosureRateLimited.Error() + ": " + e.Dimension
}

func (e DisclosureRateLimitExceededError) Unwrap() error {
	return ErrDisclosureRateLimited
}

func defaultDisclosureRateLimitConfig() DisclosureRateLimitConfig {
	return DisclosureRateLimitConfig{
		AppLimit:            600,
		AppUserLimit:        120,
		EndpointLimit:       1200,
		ConsentLimit:        20,
		ReplayLimit:         8,
		Window:              time.Minute,
		ReplayWindow:        5 * time.Minute,
		ReplayAuditCooldown: 10 * time.Minute,
	}
}

func newDisclosureRateLimiter(rdb *redis.Client, cfg DisclosureRateLimitConfig) *disclosureRateLimiter {
	if rdb == nil {
		return nil
	}
	cfg = normalizeDisclosureRateLimitConfig(cfg)
	return &disclosureRateLimiter{
		app:      newDisclosureDimensionLimiter(rdb, cfg.AppLimit, cfg.Window),
		appUser:  newDisclosureDimensionLimiter(rdb, cfg.AppUserLimit, cfg.Window),
		endpoint: newDisclosureDimensionLimiter(rdb, cfg.EndpointLimit, cfg.Window),
		consent:  newDisclosureDimensionLimiter(rdb, cfg.ConsentLimit, cfg.Window),
	}
}

func normalizeDisclosureRateLimitConfig(cfg DisclosureRateLimitConfig) DisclosureRateLimitConfig {
	defaults := defaultDisclosureRateLimitConfig()
	if cfg.Window <= 0 {
		cfg.Window = defaults.Window
	}
	if cfg.ReplayWindow <= 0 {
		cfg.ReplayWindow = defaults.ReplayWindow
	}
	if cfg.ReplayAuditCooldown <= 0 {
		cfg.ReplayAuditCooldown = defaults.ReplayAuditCooldown
	}
	if cfg.AppLimit == 0 {
		cfg.AppLimit = defaults.AppLimit
	}
	if cfg.AppUserLimit == 0 {
		cfg.AppUserLimit = defaults.AppUserLimit
	}
	if cfg.EndpointLimit == 0 {
		cfg.EndpointLimit = defaults.EndpointLimit
	}
	if cfg.ConsentLimit == 0 {
		cfg.ConsentLimit = defaults.ConsentLimit
	}
	if cfg.ReplayLimit == 0 {
		cfg.ReplayLimit = defaults.ReplayLimit
	}
	return cfg
}

func newDisclosureReplayDetector(rdb *redis.Client, cfg DisclosureRateLimitConfig) *disclosureReplayDetector {
	if rdb == nil {
		return nil
	}
	cfg = normalizeDisclosureRateLimitConfig(cfg)
	if cfg.ReplayLimit < 0 {
		return nil
	}
	return &disclosureReplayDetector{
		rdb:           rdb,
		limit:         cfg.ReplayLimit,
		window:        cfg.ReplayWindow,
		auditCooldown: cfg.ReplayAuditCooldown,
	}
}

func newDisclosureDimensionLimiter(
	rdb *redis.Client,
	limit int,
	window time.Duration,
) *middleware.RedisRateLimiter {
	if limit < 0 {
		return nil
	}
	return middleware.NewRedisRateLimiter(rdb, limit, window)
}

func (l *disclosureRateLimiter) checkDisclosure(
	ctx context.Context,
	appID int64,
	userID int64,
	endpoint string,
) error {
	if l == nil {
		return nil
	}
	checks := []struct {
		dimension string
		key       string
		limiter   *middleware.RedisRateLimiter
	}{
		{
			dimension: disclosureRateLimitDimensionApp,
			key:       "open_platform:rl:disclosure:app:" + strconv.FormatInt(appID, 10),
			limiter:   l.app,
		},
		{
			dimension: disclosureRateLimitDimensionAppUser,
			key: "open_platform:rl:disclosure:app_user:" +
				strconv.FormatInt(appID, 10) + ":" + strconv.FormatInt(userID, 10),
			limiter: l.appUser,
		},
		{
			dimension: disclosureRateLimitDimensionEndpoint,
			key:       "open_platform:rl:disclosure:endpoint:" + endpoint,
			limiter:   l.endpoint,
		},
	}
	for _, check := range checks {
		if err := allowDisclosureDimension(ctx, check.dimension, check.key, check.limiter); err != nil {
			return err
		}
	}
	return nil
}

func (l *disclosureRateLimiter) checkConsentChallenge(ctx context.Context, appID, userID int64) error {
	if l == nil {
		return nil
	}
	key := "open_platform:rl:disclosure:consent:" +
		strconv.FormatInt(appID, 10) + ":" + strconv.FormatInt(userID, 10)
	return allowDisclosureDimension(ctx, disclosureRateLimitDimensionConsent, key, l.consent)
}

func allowDisclosureDimension(
	ctx context.Context,
	dimension string,
	key string,
	limiter *middleware.RedisRateLimiter,
) error {
	if limiter == nil {
		return nil
	}
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		metrics.ObserveOpenPlatformDisclosureRateLimit(dimension, "error")
		return fmt.Errorf("%w: disclosure rate limiter %s unavailable: %v",
			ErrDisclosureUnavailable, dimension, err)
	}
	if !allowed {
		metrics.ObserveOpenPlatformDisclosureRateLimit(dimension, "limited")
		return DisclosureRateLimitExceededError{Dimension: dimension}
	}
	metrics.ObserveOpenPlatformDisclosureRateLimit(dimension, "allowed")
	return nil
}

func disclosureRateLimitDimension(err error) string {
	var limited DisclosureRateLimitExceededError
	if errors.As(err, &limited) {
		return limited.Dimension
	}
	return ""
}

func (d *disclosureReplayDetector) check(
	ctx context.Context,
	appID int64,
	userID int64,
	endpoint string,
	result string,
	scopes []string,
) (disclosureReplayDetection, error) {
	if d == nil {
		return disclosureReplayDetection{}, nil
	}
	signatureHash := disclosureReplaySignatureHash(appID, userID, endpoint, result, scopes)
	key := "open_platform:replay:disclosure:" + signatureHash
	count, err := d.rdb.Incr(ctx, key).Result()
	if err != nil {
		metrics.ObserveOpenPlatformDisclosureReplay(endpoint, "error")
		return disclosureReplayDetection{}, fmt.Errorf("increment disclosure replay detector: %w", err)
	}
	if count == 1 {
		if err := d.rdb.Expire(ctx, key, d.window).Err(); err != nil {
			metrics.ObserveOpenPlatformDisclosureReplay(endpoint, "error")
			return disclosureReplayDetection{}, fmt.Errorf("expire disclosure replay detector: %w", err)
		}
	}
	if int(count) < d.limit {
		return disclosureReplayDetection{SignatureHash: signatureHash, Count: int(count)}, nil
	}
	cooldownKey := "open_platform:replay:disclosure:audit:" + signatureHash
	setResult, err := d.rdb.SetArgs(ctx, cooldownKey, "1", redis.SetArgs{
		Mode: "NX",
		TTL:  d.auditCooldown,
	}).Result()
	if errors.Is(err, redis.Nil) {
		metrics.ObserveOpenPlatformDisclosureReplay(endpoint, "suppressed")
		return disclosureReplayDetection{SignatureHash: signatureHash, Count: int(count)}, nil
	}
	if err != nil {
		metrics.ObserveOpenPlatformDisclosureReplay(endpoint, "error")
		return disclosureReplayDetection{}, fmt.Errorf("debounce disclosure replay detector: %w", err)
	}
	if setResult != "OK" {
		metrics.ObserveOpenPlatformDisclosureReplay(endpoint, "suppressed")
		return disclosureReplayDetection{SignatureHash: signatureHash, Count: int(count)}, nil
	}
	metrics.ObserveOpenPlatformDisclosureReplay(endpoint, "detected")
	return disclosureReplayDetection{Detected: true, SignatureHash: signatureHash, Count: int(count)}, nil
}

func disclosureReplaySignatureHash(appID int64, userID int64, endpoint string, result string, scopes []string) string {
	normalizedScopes := append([]string(nil), scopes...)
	sort.Strings(normalizedScopes)
	signature := strings.Join([]string{
		strconv.FormatInt(appID, 10),
		strconv.FormatInt(userID, 10),
		endpoint,
		result,
		strings.Join(normalizedScopes, " "),
	}, "|")
	sum := sha256.Sum256([]byte(signature))
	return hex.EncodeToString(sum[:16])
}
