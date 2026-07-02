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

func (s *Service) ListAdmissionPolicyTargets(ctx context.Context) ([]AdmissionPolicyTarget, error) {
	items, err := s.repo.ListPolicyTargets(ctx)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index].PolicyID = strings.TrimSpace(items[index].PolicyID)
		items[index].Platform = strings.TrimSpace(items[index].Platform)
		items[index].GuildID = strings.TrimSpace(items[index].GuildID)
		items[index].JoinHandlingStrategy = normalizeJoinHandlingStrategy(items[index].JoinHandlingStrategy)
	}
	return items, nil
}

func (s *Service) CreateAdmissionPolicy(ctx context.Context, input AdmissionPolicyCreateRequest) (*AdmissionPolicy, error) {
	normalized, err := normalizeAdmissionPolicyCreateRequest(input)
	if err != nil {
		return nil, err
	}
	created, err := s.repo.CreatePolicyFromSource(ctx, normalized)
	if err != nil {
		return nil, err
	}
	output := normalizeAdmissionPolicyForOutput(*created)
	return &output, nil
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
	normalized, err := normalizeJoinRequestEventInput(input)
	if err != nil {
		return err
	}
	event := joinRequestAuditEvent(ctx, normalized)
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
	if !policy.GuardEnabled {
		return nil, ErrAdmissionPolicyNotFound
	}
	if policy.JoinHandlingStrategy == AdmissionJoinHandlingPostJoinTimeCode {
		return postJoinGuardDecision(policy, nil), nil
	}
	userID, err := s.repo.GetVerifiedAdmissionUserByQQ(ctx, input.QQID, policy.SchoolID, s.now())
	if err != nil {
		return nil, err
	}
	// F060：任何自动批准前先做服务端权威黑名单判定（active 且 global/guild scope 命中），
	// 命中即拒绝，不依赖机器人侧另行查询。
	blacklistEntry, err := s.repo.GetMemberBlacklistAccess(ctx, MemberBlacklistAccessQuery{
		Platform:    input.Platform,
		SubjectType: BlacklistSubjectQQUser,
		SubjectID:   input.QQID,
		GuildID:     &input.GuildID,
	}, s.now())
	if err != nil {
		return nil, err
	}
	if blacklistEntry != nil {
		return blacklistedJoinRequestDecision(policy, userID), nil
	}
	if policy.JoinHandlingStrategy == AdmissionJoinHandlingPostJoinGuard {
		return postJoinGuardDecision(policy, userID), nil
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
	policy.JoinHandlingStrategy = normalizeJoinHandlingStrategy(policy.JoinHandlingStrategy)
	policy.UnverifiedJoinRejectReason = normalizeUnverifiedJoinRejectReason(policy.UnverifiedJoinRejectReason)
	if policy.JoinHandlingStrategy == AdmissionJoinHandlingJoinRequestReview {
		policy.AutoApproveVerifiedJoin = true
		policy.AutoApproveUnverifiedJoin = false
		policy.AutoApproveJoin = false
		return policy
	}
	if policy.JoinHandlingStrategy == AdmissionJoinHandlingPostJoinTimeCode {
		policy.AutoApproveVerifiedJoin = true
		policy.AutoApproveUnverifiedJoin = true
		policy.AutoApproveJoin = true
		return policy
	}
	policy.AutoApproveVerifiedJoin = true
	policy.AutoApproveUnverifiedJoin = true
	policy.AutoApproveJoin = true
	return policy
}

func normalizeAdmissionPolicyCreateRequest(input AdmissionPolicyCreateRequest) (AdmissionPolicyCreateRequest, error) {
	input.SourcePolicyID = strings.TrimSpace(input.SourcePolicyID)
	input.Platform = strings.TrimSpace(input.Platform)
	input.GuildID = strings.TrimSpace(input.GuildID)
	if input.SourcePolicyID == "" || input.Platform == "" || input.GuildID == "" {
		return AdmissionPolicyCreateRequest{}, ErrAdmissionInvalidInput
	}
	return input, nil
}

func normalizeAdmissionPolicyForOutput(policy AdmissionPolicy) AdmissionPolicy {
	policy = normalizeAdmissionPolicy(policy)
	if policy.ManagementGuildIDs == nil {
		policy.ManagementGuildIDs = []string{}
	}
	return policy
}

func normalizeJoinRequestEventInput(input AdmissionJoinRequestEventInput) (AdmissionJoinRequestEventInput, error) {
	input.Platform = strings.TrimSpace(input.Platform)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.QQID = strings.TrimSpace(input.QQID)
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.Error = strings.TrimSpace(input.Error)
	if input.Platform == "" || input.GuildID == "" || input.QQID == "" || input.RequestID == "" {
		return AdmissionJoinRequestEventInput{}, ErrAdmissionInvalidInput
	}
	decision, err := normalizeJoinRequestEventDecision(input.Decision)
	if err != nil {
		return AdmissionJoinRequestEventInput{}, err
	}
	input.Decision = decision
	return input, nil
}

func normalizeJoinRequestDecisionInput(input AdmissionJoinRequestDecisionInput) AdmissionJoinRequestDecisionInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.QQID = strings.TrimSpace(input.QQID)
	input.RequestID = strings.TrimSpace(input.RequestID)
	return input
}

func normalizeJoinRequestEventDecision(
	action AdmissionJoinRequestDecisionAction,
) (AdmissionJoinRequestDecisionAction, error) {
	normalized := AdmissionJoinRequestDecisionAction(strings.ToLower(strings.TrimSpace(string(action))))
	switch normalized {
	case "":
		return "", nil
	case AdmissionJoinRequestDecisionApprove, AdmissionJoinRequestDecisionReject:
		return normalized, nil
	default:
		return "", ErrAdmissionInvalidInput
	}
}

func postJoinGuardDecision(policy *AdmissionPolicy, userID *int64) *AdmissionJoinRequestDecision {
	decision := &AdmissionJoinRequestDecision{
		Decision:                  AdmissionJoinRequestDecisionApprove,
		Reason:                    postJoinAutoApproveReason(policy.JoinHandlingStrategy),
		VerificationState:         AdmissionJoinRequestUnverified,
		JoinHandlingStrategy:      policy.JoinHandlingStrategy,
		AutoApproveVerifiedJoin:   policy.AutoApproveVerifiedJoin,
		AutoApproveUnverifiedJoin: policy.AutoApproveUnverifiedJoin,
		PolicyID:                  policy.ID,
	}
	if userID != nil {
		value := strconv.FormatInt(*userID, 10)
		decision.VerificationState = AdmissionJoinRequestVerified
		decision.UserID = &value
	}
	return decision
}

func postJoinAutoApproveReason(strategy AdmissionJoinHandlingStrategy) string {
	if strategy == AdmissionJoinHandlingPostJoinTimeCode {
		return "post_join_time_code_auto_approve"
	}
	return "post_join_guard_auto_approve"
}

func blacklistedJoinRequestDecision(policy *AdmissionPolicy, userID *int64) *AdmissionJoinRequestDecision {
	decision := &AdmissionJoinRequestDecision{
		Decision:                  AdmissionJoinRequestDecisionReject,
		Reason:                    "member_blacklisted",
		VerificationState:         AdmissionJoinRequestUnverified,
		JoinHandlingStrategy:      policy.JoinHandlingStrategy,
		AutoApproveVerifiedJoin:   policy.AutoApproveVerifiedJoin,
		AutoApproveUnverifiedJoin: policy.AutoApproveUnverifiedJoin,
		PolicyID:                  policy.ID,
	}
	if userID != nil {
		value := strconv.FormatInt(*userID, 10)
		decision.VerificationState = AdmissionJoinRequestVerified
		decision.UserID = &value
	}
	return decision
}

func verifiedJoinRequestDecision(policy *AdmissionPolicy, userID *int64) *AdmissionJoinRequestDecision {
	var outputUserID *string
	if userID != nil {
		value := strconv.FormatInt(*userID, 10)
		outputUserID = &value
	}
	decision := &AdmissionJoinRequestDecision{
		VerificationState:         AdmissionJoinRequestVerified,
		JoinHandlingStrategy:      policy.JoinHandlingStrategy,
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
		JoinHandlingStrategy:      policy.JoinHandlingStrategy,
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
	decision.Reason = policy.UnverifiedJoinRejectReason
	return decision
}

func normalizeJoinHandlingStrategy(strategy AdmissionJoinHandlingStrategy) AdmissionJoinHandlingStrategy {
	normalized := AdmissionJoinHandlingStrategy(strings.ToLower(strings.TrimSpace(string(strategy))))
	switch normalized {
	case AdmissionJoinHandlingJoinRequestReview:
		return AdmissionJoinHandlingJoinRequestReview
	case AdmissionJoinHandlingPostJoinTimeCode, "join_request_time_code":
		return AdmissionJoinHandlingPostJoinTimeCode
	case AdmissionJoinHandlingPostJoinGuard, "":
		return AdmissionJoinHandlingPostJoinGuard
	default:
		return normalized
	}
}

func normalizeUnverifiedJoinRejectReason(reason string) string {
	trimmed := strings.TrimSpace(reason)
	if trimmed != "" {
		return trimmed
	}
	return "请先完成 StuHelper 学生认证后再申请入群。"
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
