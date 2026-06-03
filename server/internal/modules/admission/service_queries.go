package admission

import (
	"context"
	"strconv"
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

func (s *Service) ResolveJoinRequestDecision(
	ctx context.Context,
	input AdmissionJoinRequestDecisionInput,
) (*AdmissionJoinRequestDecision, error) {
	input = normalizeJoinRequestDecisionInput(input)
	if input.Platform == "" || input.GuildID == "" || input.QQID == "" || input.RequestID == "" {
		return nil, ErrAdmissionInvalidInput
	}
	policy, err := s.loadPolicy(ctx, input.Platform, input.GuildID)
	if err != nil {
		return nil, err
	}
	userID, err := s.repo.GetVerifiedAdmissionUserByQQ(ctx, input.QQID, policy.SchoolID, s.now())
	if err != nil {
		return nil, err
	}
	if userID != nil {
		return verifiedJoinRequestDecision(policy, userID), nil
	}
	return unverifiedJoinRequestDecision(policy), nil
}

func normalizeAdmissionPolicy(policy AdmissionPolicy) AdmissionPolicy {
	policy.ID = strings.TrimSpace(policy.ID)
	policy.ManagementGuildIDs = normalizeStringSlice(policy.ManagementGuildIDs)
	policy.Platform = strings.TrimSpace(policy.Platform)
	policy.GuildID = strings.TrimSpace(policy.GuildID)
	if policy.AutoApproveJoin && !policy.AutoApproveVerifiedJoin && !policy.AutoApproveUnverifiedJoin {
		policy.AutoApproveVerifiedJoin = true
		policy.AutoApproveUnverifiedJoin = true
	}
	policy.AutoApproveJoin = policy.AutoApproveVerifiedJoin && policy.AutoApproveUnverifiedJoin
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
	input.Decision = normalizeJoinRequestDecisionAction(input.Decision)
	input.Error = strings.TrimSpace(input.Error)
	return input
}

func normalizeJoinRequestDecisionInput(input AdmissionJoinRequestDecisionInput) AdmissionJoinRequestDecisionInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.QQID = strings.TrimSpace(input.QQID)
	input.RequestID = strings.TrimSpace(input.RequestID)
	return input
}

func normalizeJoinRequestDecisionAction(
	action AdmissionJoinRequestDecisionAction,
) AdmissionJoinRequestDecisionAction {
	switch action {
	case AdmissionJoinRequestDecisionApprove, AdmissionJoinRequestDecisionReject:
		return action
	default:
		return AdmissionJoinRequestDecisionApprove
	}
}

func verifiedJoinRequestDecision(policy *AdmissionPolicy, userID *int64) *AdmissionJoinRequestDecision {
	var outputUserID *string
	if userID != nil {
		value := strconv.FormatInt(*userID, 10)
		outputUserID = &value
	}
	decision := &AdmissionJoinRequestDecision{
		VerificationState:         AdmissionJoinRequestVerified,
		AutoApproveVerifiedJoin:   policy.AutoApproveVerifiedJoin,
		AutoApproveUnverifiedJoin: policy.AutoApproveUnverifiedJoin,
		PolicyID:                  policy.ID,
		UserID:                    outputUserID,
	}
	if policy.AutoApproveVerifiedJoin {
		decision.Decision = AdmissionJoinRequestDecisionApprove
		decision.Reason = "verified_auto_approve"
		return decision
	}
	decision.Decision = AdmissionJoinRequestDecisionReject
	decision.Reason = "verified_auto_approve_disabled"
	return decision
}

func unverifiedJoinRequestDecision(policy *AdmissionPolicy) *AdmissionJoinRequestDecision {
	decision := &AdmissionJoinRequestDecision{
		VerificationState:         AdmissionJoinRequestUnverified,
		AutoApproveVerifiedJoin:   policy.AutoApproveVerifiedJoin,
		AutoApproveUnverifiedJoin: policy.AutoApproveUnverifiedJoin,
		PolicyID:                  policy.ID,
	}
	if policy.AutoApproveUnverifiedJoin {
		decision.Decision = AdmissionJoinRequestDecisionApprove
		decision.Reason = "unverified_auto_approve"
		return decision
	}
	decision.Decision = AdmissionJoinRequestDecisionReject
	decision.Reason = "unverified_auto_approve_disabled"
	return decision
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
