package admission

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

const (
	memberBlacklistDecisionAllowed = "allowed"
	memberBlacklistDecisionBlocked = "blocked"
	defaultMemberBlacklistPageSize = 50
	maxMemberBlacklistPageSize     = 100
)

func (s *Service) GetMemberBlacklistAccess(
	ctx context.Context,
	query MemberBlacklistAccessQuery,
) (*MemberBlacklistAccessDecision, error) {
	query = normalizeMemberBlacklistAccessQuery(query)
	if err := validateMemberBlacklistAccessQuery(query); err != nil {
		return nil, err
	}
	entry, err := s.repo.FindActiveMemberBlacklist(ctx, query)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return &MemberBlacklistAccessDecision{CanJoin: true, Decision: memberBlacklistDecisionAllowed}, nil
	}
	return &MemberBlacklistAccessDecision{
		CanJoin:          false,
		Decision:         memberBlacklistDecisionBlocked,
		MatchedBlacklist: entry,
	}, nil
}

func (s *Service) ListMemberBlacklist(
	ctx context.Context,
	filter MemberBlacklistListFilter,
) ([]MemberBlacklistEntry, int, error) {
	return s.repo.ListMemberBlacklist(ctx, normalizeMemberBlacklistListFilter(filter))
}

func (s *Service) CreateMemberBlacklist(
	ctx context.Context,
	input MemberBlacklistCreateInput,
) (*MemberBlacklistEntry, error) {
	input = normalizeMemberBlacklistCreateInput(input)
	if err := validateMemberBlacklistCreateInput(input); err != nil {
		return nil, err
	}
	entryID, err := id.New()
	if err != nil {
		return nil, err
	}
	input.ID = entryID
	var created *MemberBlacklistEntry
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		entry, err := s.repo.CreateMemberBlacklistTx(ctx, tx, MemberBlacklistCreateTxInput{
			Entry: input,
			Now:   s.now(),
		})
		created = entry
		return err
	})
	return created, err
}

func (s *Service) ReleaseMemberBlacklist(ctx context.Context, input MemberBlacklistReleaseInput) error {
	input = normalizeMemberBlacklistReleaseInput(input)
	if input.ID == "" || input.ReleasedByID == "" || input.ReleaseReasonCode == "" {
		return ErrMemberBlacklistInvalidInput
	}
	return s.repo.ReleaseMemberBlacklist(ctx, input, s.now())
}

func (s *Service) ReleaseMemberBlacklistBySubject(
	ctx context.Context,
	input MemberBlacklistReleaseBySubjectInput,
) error {
	input = normalizeMemberBlacklistReleaseBySubjectInput(input)
	if err := validateMemberBlacklistReleaseBySubjectInput(input); err != nil {
		return err
	}
	return s.repo.ReleaseMemberBlacklistBySubject(ctx, input, s.now())
}

func normalizeMemberBlacklistAccessQuery(input MemberBlacklistAccessQuery) MemberBlacklistAccessQuery {
	input.Platform = strings.TrimSpace(input.Platform)
	input.SubjectID = strings.TrimSpace(input.SubjectID)
	input.GuildID = strings.TrimSpace(input.GuildID)
	if input.SubjectType == "" {
		input.SubjectType = MemberBlacklistSubjectQQUser
	}
	return input
}

func normalizeMemberBlacklistCreateInput(input MemberBlacklistCreateInput) MemberBlacklistCreateInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.SubjectID = strings.TrimSpace(input.SubjectID)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.ReasonCode = strings.TrimSpace(input.ReasonCode)
	input.ReasonText = strings.TrimSpace(input.ReasonText)
	input.CreatedByID = strings.TrimSpace(input.CreatedByID)
	return input
}
