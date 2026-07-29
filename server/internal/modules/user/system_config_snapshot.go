package user

import (
	"context"
	"fmt"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/pkg/systemconfig"
)

func (s *Service) LoadSystemConfigSnapshots(ctx context.Context) error {
	if err := s.loadAuthTokenPolicySnapshot(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Service) loadAuthTokenPolicySnapshot(ctx context.Context) error {
	configs, err := s.repo.ListSystemConfigs(ctx)
	if err != nil {
		return fmt.Errorf("load auth token policy snapshot: %w", err)
	}

	for i := range configs {
		if !systemconfig.AffectsAuthTokenPolicy(configs[i].Key) {
			continue
		}
		return applyAuthTokenPolicySnapshotValue(configs[i].Value)
	}

	systemconfig.InvalidateAuthTokenPolicySnapshot()
	return nil
}

func applyAuthTokenPolicySnapshotValue(raw string) error {
	ttlSeconds, err := systemconfig.ParseAuthAccessTokenTTL(raw)
	if err != nil {
		return fmt.Errorf("%w: auth access token ttl %v", ErrInvalidSystemConfigValue, err)
	}

	systemconfig.SetAuthTokenPolicySnapshot(systemconfig.AuthTokenPolicySnapshot{
		AccessTokenTTLSeconds: ttlSeconds,
		LoadedAt:              time.Now(),
	})
	return nil
}
