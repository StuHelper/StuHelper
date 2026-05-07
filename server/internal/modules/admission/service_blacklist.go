package admission

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const memberBlacklistReasonBlocked = "member_blacklisted"

func (s *Service) GetMemberBlacklistAccess(
	ctx context.Context,
	query MemberBlacklistAccessQuery,
) (*MemberBlacklistAccessDecision, error) {
	query = normalizeMemberBlacklistAccessQuery(query)
	if err := validateMemberBlacklistAccessQuery(query); err != nil {
		return nil, err
	}
	entry, err := s.repo.GetMemberBlacklistAccess(ctx, query, s.now())
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return &MemberBlacklistAccessDecision{CanJoin: true, Decision: "allowed"}, nil
	}
	reason := memberBlacklistReasonBlocked
	return &MemberBlacklistAccessDecision{
		CanJoin:          false,
		Decision:         "blocked",
		MatchedBlacklist: entry,
		Reason:           &reason,
	}, nil
}

func (s *Service) ListMemberBlacklist(
	ctx context.Context,
	filter MemberBlacklistListFilter,
) ([]MemberBlacklistEntry, int, error) {
	filter = normalizeMemberBlacklistListFilter(filter)
	if err := validateMemberBlacklistListFilter(filter); err != nil {
		return nil, 0, err
	}
	return s.repo.ListMemberBlacklist(ctx, filter, s.now())
}

func (s *Service) CreateMemberBlacklistFromAdmin(
	ctx context.Context,
	input MemberBlacklistCreateInput,
) (*MemberBlacklistEntry, error) {
	return s.createMemberBlacklist(ctx, input, memberBlacklistEntryPointAdmin)
}

func (s *Service) CreateMemberBlacklistFromBot(
	ctx context.Context,
	input MemberBlacklistCreateInput,
) (*MemberBlacklistEntry, error) {
	return s.createMemberBlacklist(ctx, input, memberBlacklistEntryPointBot)
}

func (s *Service) createAdmissionFailureBlacklistTx(
	ctx context.Context,
	tx pgx.Tx,
	input MemberBlacklistCreateInput,
) (*MemberBlacklistEntry, error) {
	return s.createMemberBlacklistTx(ctx, tx, input, memberBlacklistEntryPointInternal)
}

func (s *Service) ReleaseMemberBlacklistFromAdmin(
	ctx context.Context,
	input MemberBlacklistReleaseInput,
) (*MemberBlacklistEntry, error) {
	return s.releaseMemberBlacklistByID(ctx, input, memberBlacklistEntryPointAdmin)
}

func (s *Service) ReleaseMemberBlacklistFromBot(
	ctx context.Context,
	input MemberBlacklistReleaseInput,
) (*MemberBlacklistEntry, error) {
	return s.releaseMemberBlacklistByID(ctx, input, memberBlacklistEntryPointBot)
}

func (s *Service) ReleaseMemberBlacklistBySubjectFromAdmin(
	ctx context.Context,
	input MemberBlacklistReleaseBySubjectInput,
) (*MemberBlacklistEntry, error) {
	return s.releaseMemberBlacklistBySubject(ctx, input, memberBlacklistEntryPointAdmin)
}

func (s *Service) ReleaseMemberBlacklistBySubjectFromBot(
	ctx context.Context,
	input MemberBlacklistReleaseBySubjectInput,
) (*MemberBlacklistEntry, error) {
	return s.releaseMemberBlacklistBySubject(ctx, input, memberBlacklistEntryPointBot)
}

func (s *Service) createMemberBlacklist(
	ctx context.Context,
	input MemberBlacklistCreateInput,
	entryPoint memberBlacklistEntryPoint,
) (*MemberBlacklistEntry, error) {
	var created *MemberBlacklistEntry
	err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		entry, err := s.createMemberBlacklistTx(ctx, tx, input, entryPoint)
		created = entry
		return err
	})
	return created, err
}

func (s *Service) createMemberBlacklistTx(
	ctx context.Context,
	tx pgx.Tx,
	input MemberBlacklistCreateInput,
	entryPoint memberBlacklistEntryPoint,
) (*MemberBlacklistEntry, error) {
	input = normalizeMemberBlacklistCreateInput(input)
	if err := validateMemberBlacklistCreateInput(input, entryPoint); err != nil {
		return nil, err
	}
	now := s.now()
	if err := validateMemberBlacklistCreateTime(input, now); err != nil {
		return nil, err
	}
	key := memberBlacklistCreateKey(input)
	if err := s.releaseExpiredMemberBlacklistForCreateTx(ctx, tx, key, now); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetActiveMemberBlacklistByKeyTx(ctx, tx, key, now)
	if err != nil || existing != nil {
		return existing, err
	}
	if err := s.repo.CreateMemberBlacklistSavepointTx(ctx, tx); err != nil {
		return nil, err
	}
	created, err := s.repo.CreateMemberBlacklistTx(ctx, tx, input, now)
	if err != nil {
		return s.memberBlacklistCreateConflictResultTx(ctx, tx, key, now, err)
	}
	if err := s.repo.ReleaseMemberBlacklistCreateSavepointTx(ctx, tx); err != nil {
		return nil, err
	}
	return created, s.repo.InsertAuditEventTx(ctx, tx, memberBlacklistCreatedAuditEvent(ctx, created))
}

func (s *Service) memberBlacklistCreateConflictResultTx(
	ctx context.Context,
	tx pgx.Tx,
	key memberBlacklistKey,
	now time.Time,
	createErr error,
) (*MemberBlacklistEntry, error) {
	if !isMemberBlacklistActiveUniqueViolation(createErr) {
		return nil, createErr
	}
	if err := s.repo.RollbackMemberBlacklistCreateSavepointTx(ctx, tx); err != nil {
		return nil, fmt.Errorf("recover member blacklist create conflict: %w", err)
	}
	if err := s.repo.ReleaseMemberBlacklistCreateSavepointTx(ctx, tx); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetActiveMemberBlacklistByKeyTx(ctx, tx, key, now)
	if err != nil || existing != nil {
		return existing, err
	}
	return nil, createErr
}

func (s *Service) releaseExpiredMemberBlacklistForCreateTx(
	ctx context.Context,
	tx pgx.Tx,
	key memberBlacklistKey,
	now time.Time,
) error {
	released, err := s.repo.ReleaseExpiredMemberBlacklistTx(ctx, tx, key, now)
	if err != nil {
		return err
	}
	for i := range released {
		if err := s.afterMemberBlacklistReleaseTx(ctx, tx, &released[i], BlacklistReleasePolicyExpiredAuto); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) releaseMemberBlacklistByID(
	ctx context.Context,
	input MemberBlacklistReleaseInput,
	entryPoint memberBlacklistEntryPoint,
) (*MemberBlacklistEntry, error) {
	input = normalizeMemberBlacklistReleaseInput(input)
	if err := validateMemberBlacklistReleaseInput(input, entryPoint); err != nil {
		return nil, err
	}
	var released *MemberBlacklistEntry
	err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		entry, err := s.repo.ReleaseMemberBlacklistByIDTx(ctx, tx, input, s.now())
		if err != nil {
			return err
		}
		released = entry
		return s.afterMemberBlacklistReleaseTx(ctx, tx, entry, input.ReleaseReasonCode)
	})
	return released, err
}

func (s *Service) releaseMemberBlacklistBySubject(
	ctx context.Context,
	input MemberBlacklistReleaseBySubjectInput,
	entryPoint memberBlacklistEntryPoint,
) (*MemberBlacklistEntry, error) {
	input = normalizeMemberBlacklistReleaseBySubjectInput(input)
	if err := validateMemberBlacklistReleaseBySubjectInput(input, entryPoint); err != nil {
		return nil, err
	}
	var released *MemberBlacklistEntry
	err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		entry, err := s.repo.ReleaseMemberBlacklistBySubjectTx(ctx, tx, input, s.now())
		if err != nil {
			return err
		}
		released = entry
		return s.afterMemberBlacklistReleaseTx(ctx, tx, entry, input.ReleaseReasonCode)
	})
	return released, err
}

func (s *Service) afterMemberBlacklistReleaseTx(
	ctx context.Context,
	tx pgx.Tx,
	entry *MemberBlacklistEntry,
	reason MemberBlacklistReleaseReasonCode,
) error {
	effects, err := s.resetAdmissionFailureAfterPardonTx(ctx, tx, entry, reason)
	if err != nil {
		return err
	}
	return s.repo.InsertAuditEventTx(ctx, tx, memberBlacklistReleasedAuditEvent(ctx, entry, effects))
}

func (s *Service) resetAdmissionFailureAfterPardonTx(
	ctx context.Context,
	tx pgx.Tx,
	entry *MemberBlacklistEntry,
	reason MemberBlacklistReleaseReasonCode,
) (memberBlacklistReleaseEffects, error) {
	if entry.Source != BlacklistSourceAdmissionFailure || reason == BlacklistReleaseOnly || entry.GuildID == nil {
		return memberBlacklistReleaseEffects{}, nil
	}
	previous, err := s.repo.ResetAdmissionFailureCountTx(ctx, tx, entry.Platform, *entry.GuildID, entry.SubjectID, s.now())
	return memberBlacklistReleaseEffects{
		AdmissionFailureCountReset: true,
		PreviousFailureCount:       previous,
	}, err
}

func memberBlacklistCreateKey(input MemberBlacklistCreateInput) memberBlacklistKey {
	return memberBlacklistKey{
		Platform: input.Platform, SubjectType: input.SubjectType, SubjectID: input.SubjectID,
		ScopeType: input.ScopeType, GuildID: input.GuildID,
	}
}

func normalizeMemberBlacklistString(value string) string {
	return strings.TrimSpace(value)
}
