package admission

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Service) ListAdmissionPolicies(ctx context.Context) ([]AdmissionPolicy, error) {
	items, err := s.repo.ListPolicies(ctx)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index] = normalizeAdmissionPolicyForOutput(items[index])
	}
	return items, nil
}

func (s *Service) UpdateAdmissionPolicy(ctx context.Context, policy AdmissionPolicy) (*AdmissionPolicy, error) {
	updated, err := s.repo.UpdatePolicy(ctx, normalizeAdmissionPolicy(policy))
	if err != nil {
		return nil, err
	}
	output := normalizeAdmissionPolicyForOutput(*updated)
	return &output, nil
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

func (s *Service) ListAdminFreshmanApplications(
	ctx context.Context,
	filter FreshmanApplicationListFilter,
) ([]FreshmanApplication, int, error) {
	rows, total, err := s.repo.ListAdminFreshmanApplications(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	items := make([]FreshmanApplication, 0, len(rows))
	for _, row := range rows {
		item, err := s.adminFreshmanApplicationFromRow(ctx, row)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (s *Service) GetFreshmanApplication(ctx context.Context, applicationID string) (*FreshmanApplication, error) {
	return s.repo.GetFreshmanApplicationByID(ctx, strings.TrimSpace(applicationID))
}

func (s *Service) adminFreshmanApplicationFromRow(
	ctx context.Context,
	row adminFreshmanApplicationRow,
) (FreshmanApplication, error) {
	app := row.Application
	app.QQID = trimmedStringPtr(row.QQID)
	app.FailureCount = row.FailureCount
	if row.ObjectKey == nil {
		return app, nil
	}
	if s.materialStore == nil {
		return app, nil
	}
	url, err := s.materialStore.GetAdmissionMaterialURL(ctx, strings.TrimSpace(*row.ObjectKey))
	if err != nil {
		return FreshmanApplication{}, err
	}
	app.MaterialURL = optionalStringPtr(url)
	return app, nil
}

func trimmedStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (s *Service) MarkFreshmanApplicationForwarded(ctx context.Context, applicationID string) error {
	return s.repo.MarkFreshmanApplicationForwarded(ctx, strings.TrimSpace(applicationID), s.now())
}

func (s *Service) RecordJoinRequestEvent(ctx context.Context, input AdmissionJoinRequestEventInput) error {
	event := joinRequestAuditEvent(ctx, normalizeJoinRequestEventInput(input))
	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return s.repo.InsertAuditEventTx(ctx, tx, event)
	})
}

func normalizeAdmissionPolicy(policy AdmissionPolicy) AdmissionPolicy {
	policy.ID = strings.TrimSpace(policy.ID)
	policy.ManagementGuildIDs = normalizeStringSlice(policy.ManagementGuildIDs)
	policy.Platform = strings.TrimSpace(policy.Platform)
	policy.GuildID = strings.TrimSpace(policy.GuildID)
	return policy
}

func normalizeAdmissionPolicyForOutput(policy AdmissionPolicy) AdmissionPolicy {
	policy = normalizeAdmissionPolicy(policy)
	if policy.ManagementGuildIDs == nil {
		policy.ManagementGuildIDs = []string{}
	}
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
