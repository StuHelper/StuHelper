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

func (s *Service) GetAdmissionQQAccess(ctx context.Context, query AdmissionQQAccessQuery) (*AdmissionQQAccess, error) {
	query = normalizeAdmissionQQAccessQuery(query)
	memberAccess, err := s.GetMemberBlacklistAccess(ctx, MemberBlacklistAccessQuery{
		Platform:    query.Platform,
		SubjectType: MemberBlacklistSubjectQQUser,
		SubjectID:   query.QQID,
		GuildID:     query.GuildID,
	})
	if err != nil {
		return nil, err
	}
	if !memberAccess.CanJoin {
		reason := blacklistReason
		return &AdmissionQQAccess{CanJoin: false, Reason: &reason}, nil
	}
	failure, err := s.repo.GetActiveAdmissionFailure(ctx, query)
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

func (s *Service) ReleaseAdmissionBlacklist(ctx context.Context, input AdmissionBlacklistReleaseInput) error {
	input = normalizeAdmissionBlacklistReleaseInput(input)
	if input.Platform == "" || input.GuildID == "" || input.QQID == "" {
		return ErrMemberBlacklistInvalidInput
	}
	now := s.now()
	releasedFailure, err := s.repo.ReleaseAdmissionBlacklist(ctx, input, now)
	if err != nil {
		return err
	}
	releasedMember, err := s.repo.ReleaseAdmissionFailureMemberBlacklist(ctx, input, now)
	if err != nil {
		return err
	}
	if !releasedFailure && !releasedMember {
		return ErrAdmissionBlacklistNotFound
	}
	return nil
}

func normalizeAdmissionPolicy(policy AdmissionPolicy) AdmissionPolicy {
	policy.ID = strings.TrimSpace(policy.ID)
	policy.ManagementGuildIDs = normalizeStringSlice(policy.ManagementGuildIDs)
	policy.Platform = strings.TrimSpace(policy.Platform)
	policy.GuildID = strings.TrimSpace(policy.GuildID)
	return policy
}

func normalizeAdmissionQQAccessQuery(input AdmissionQQAccessQuery) AdmissionQQAccessQuery {
	input.Platform = strings.TrimSpace(input.Platform)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.QQID = strings.TrimSpace(input.QQID)
	return input
}

func normalizeAdmissionBlacklistReleaseInput(
	input AdmissionBlacklistReleaseInput,
) AdmissionBlacklistReleaseInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.QQID = strings.TrimSpace(input.QQID)
	return input
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
