package admission

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

const blacklistReason = "blacklisted"

func (s *Service) ListAdmissionPolicies(ctx context.Context) ([]AdmissionPolicy, error) {
	return s.repo.ListPolicies(ctx)
}

func (s *Service) UpdateAdmissionPolicy(ctx context.Context, policy AdmissionPolicy) (*AdmissionPolicy, error) {
	return s.repo.UpdatePolicy(ctx, normalizeAdmissionPolicy(policy))
}

func (s *Service) ListAdmissionSessions(
	ctx context.Context,
	filter AdmissionSessionListFilter,
) ([]AdmissionSession, int, error) {
	return s.repo.ListSessions(ctx, filter)
}

func (s *Service) ListFreshmanApplications(
	ctx context.Context,
	filter FreshmanApplicationListFilter,
) ([]FreshmanApplication, int, error) {
	return s.repo.ListFreshmanApplications(ctx, filter)
}

func (s *Service) GetFreshmanApplication(ctx context.Context, applicationID string) (*FreshmanApplication, error) {
	return s.repo.GetFreshmanApplicationByID(ctx, strings.TrimSpace(applicationID))
}

func (s *Service) MarkFreshmanApplicationForwarded(ctx context.Context, applicationID string) error {
	return s.repo.MarkFreshmanApplicationForwarded(ctx, strings.TrimSpace(applicationID), s.now())
}

func (s *Service) GetAdmissionQQAccess(ctx context.Context, qqID string) (*AdmissionQQAccess, error) {
	failure, err := s.repo.GetActiveAdmissionFailure(ctx, strings.TrimSpace(qqID))
	if err != nil {
		return nil, err
	}
	if failure != nil {
		reason := blacklistReason
		return &AdmissionQQAccess{CanJoin: false, Reason: &reason}, nil
	}
	return &AdmissionQQAccess{CanJoin: true, AutoApproveJoin: true}, nil
}

func (s *Service) RecordJoinRequestEvent(ctx context.Context, input AdmissionJoinRequestEventInput) error {
	event := joinRequestAuditEvent(ctx, normalizeJoinRequestEventInput(input))
	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return s.repo.InsertAuditEventTx(ctx, tx, event)
	})
}

func (s *Service) ReleaseAdmissionBlacklist(ctx context.Context, qqID string) error {
	return s.repo.ReleaseAdmissionBlacklist(ctx, strings.TrimSpace(qqID), s.now())
}

func normalizeAdmissionPolicy(policy AdmissionPolicy) AdmissionPolicy {
	policy.ID = strings.TrimSpace(policy.ID)
	policy.ManagementGuildIDs = normalizeStringSlice(policy.ManagementGuildIDs)
	policy.Platform = strings.TrimSpace(policy.Platform)
	policy.GuildID = strings.TrimSpace(policy.GuildID)
	return policy
}

func normalizeJoinRequestEventInput(input AdmissionJoinRequestEventInput) AdmissionJoinRequestEventInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.QQID = strings.TrimSpace(input.QQID)
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.Error = strings.TrimSpace(input.Error)
	return input
}

func normalizeStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
