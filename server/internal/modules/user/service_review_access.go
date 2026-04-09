package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/systemconfig"
)

const userReviewAccessPolicyTTL = 5 * time.Minute

func (s *Service) getReviewAccessPolicySnapshot(ctx context.Context) (systemconfig.ReviewAccessPolicySnapshot, error) {
	cached := systemconfig.GetReviewAccessPolicySnapshot()
	if userReviewAccessPolicyFresh(cached) {
		return cached, nil
	}

	schools, err := s.repo.ListSchoolConfigs(ctx)
	if err != nil {
		return cached, fmt.Errorf("load enabled schools: %w", err)
	}
	configs, err := s.repo.ListSystemConfigs(ctx)
	if err != nil {
		return cached, fmt.Errorf("load system configs: %w", err)
	}

	policy, err := buildUserReviewAccessPolicy(schools, configs)
	if err != nil {
		return cached, err
	}
	policy.LoadedAt = time.Now()
	systemconfig.SetReviewAccessPolicySnapshot(policy)
	return policy, nil
}

func userReviewAccessPolicyFresh(policy systemconfig.ReviewAccessPolicySnapshot) bool {
	if policy.LoadedAt.IsZero() {
		return false
	}
	return time.Since(policy.LoadedAt) <= userReviewAccessPolicyTTL
}

func buildUserReviewAccessPolicy(schools []SchoolConfig, configs []SystemConfig) (systemconfig.ReviewAccessPolicySnapshot, error) {
	policy := systemconfig.DefaultReviewAccessPolicySnapshot()
	enabledSchoolIDs := make([]string, 0, len(schools))
	for _, school := range schools {
		enabledSchoolIDs = append(enabledSchoolIDs, strconv.FormatInt(school.SchoolID, 10))
	}

	configMap := make(map[string]string, len(configs))
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	allowedSchoolIDs, err := parseUserReviewAccessSchoolIDs(configMap, enabledSchoolIDs)
	if err != nil {
		return systemconfig.ReviewAccessPolicySnapshot{}, err
	}
	policy.AllowedSchoolIDs = allowedSchoolIDs
	return policy, nil
}

func parseUserReviewAccessSchoolIDs(configs map[string]string, enabledSchoolIDs []string) ([]string, error) {
	if raw, ok := firstNonEmptyUserConfig(configs, systemconfig.ReviewAccessSchoolIDsKey); ok {
		return systemconfig.ParseStringList(raw)
	}
	return systemconfig.NormalizeStringList(enabledSchoolIDs), nil
}

func firstNonEmptyUserConfig(configs map[string]string, keys ...string) (string, bool) {
	for _, key := range keys {
		value := strings.TrimSpace(configs[key])
		if value == "" {
			continue
		}
		return value, true
	}
	return "", false
}
