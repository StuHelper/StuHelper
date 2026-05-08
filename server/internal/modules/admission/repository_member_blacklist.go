package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) FindActiveMemberBlacklist(
	ctx context.Context,
	query MemberBlacklistAccessQuery,
) (*MemberBlacklistEntry, error) {
	row := r.db.QueryRow(ctx, activeMemberBlacklistSQL(), query.Platform, query.SubjectType, query.SubjectID, query.GuildID)
	return scanNullableMemberBlacklistEntry(row)
}

func (r *Repository) ListMemberBlacklist(
	ctx context.Context,
	filter MemberBlacklistListFilter,
) ([]MemberBlacklistEntry, int, error) {
	rows, err := r.db.Query(ctx, listMemberBlacklistSQL(),
		filter.Platform, filter.SubjectType, filter.SubjectID, filter.ScopeType,
		filter.GuildID, filter.ActiveOnly, filter.PageSize, filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("ListMemberBlacklist: %w", err)
	}
	defer rows.Close()
	return scanMemberBlacklistEntryList(rows)
}

func (r *Repository) CreateMemberBlacklistTx(
	ctx context.Context,
	tx pgx.Tx,
	input MemberBlacklistCreateTxInput,
) (*MemberBlacklistEntry, error) {
	entry := input.Entry
	if existing, err := r.findActiveMemberBlacklistByScopeTx(ctx, tx, entry); err != nil || existing != nil {
		return existing, err
	}
	if err := r.releaseExpiredMemberBlacklistByScopeTx(ctx, tx, input); err != nil {
		return nil, err
	}
	metadata, err := json.Marshal(nonNilMetadata(entry.Metadata))
	if err != nil {
		return nil, fmt.Errorf("marshal member blacklist metadata: %w", err)
	}
	return scanMemberBlacklistEntry(tx.QueryRow(ctx, insertMemberBlacklistSQL(),
		entry.ID, entry.Platform, entry.SubjectType, entry.SubjectID, entry.ScopeType,
		nullableGuildID(entry.ScopeType, entry.GuildID), entry.Source, entry.ReasonCode,
		entry.ReasonText, entry.CreatedByType, entry.CreatedByID, entry.CreatedFrom,
		entry.ExpiresAt, metadata, input.Now,
	))
}

func (r *Repository) ReleaseMemberBlacklist(ctx context.Context, input MemberBlacklistReleaseInput, now time.Time) error {
	tag, err := r.db.Exec(ctx, releaseMemberBlacklistByIDSQL(),
		input.ID, now, input.ReleasedByType, input.ReleasedByID,
		input.ReleaseReasonCode, input.ReleaseReason,
	)
	if err != nil {
		return fmt.Errorf("ReleaseMemberBlacklist: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMemberBlacklistNotFound
	}
	return nil
}

func (r *Repository) ReleaseMemberBlacklistBySubject(
	ctx context.Context,
	input MemberBlacklistReleaseBySubjectInput,
	now time.Time,
) error {
	tag, err := r.db.Exec(ctx, releaseMemberBlacklistBySubjectSQL(),
		input.Platform, input.SubjectType, input.SubjectID, input.ScopeType,
		nullableGuildID(input.ScopeType, input.GuildID), now, input.ReleasedByType,
		input.ReleasedByID, input.ReleaseReasonCode, input.ReleaseReason,
	)
	if err != nil {
		return fmt.Errorf("ReleaseMemberBlacklistBySubject: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMemberBlacklistNotFound
	}
	return nil
}

func (r *Repository) ReleaseAdmissionFailureMemberBlacklist(
	ctx context.Context,
	input AdmissionBlacklistReleaseInput,
	now time.Time,
) (bool, error) {
	tag, err := r.db.Exec(ctx, releaseAdmissionFailureMemberBlacklistSQL(),
		input.Platform, MemberBlacklistSubjectQQUser, input.QQID, MemberBlacklistScopeGuild,
		input.GuildID, now,
	)
	if err != nil {
		return false, fmt.Errorf("ReleaseAdmissionFailureMemberBlacklist: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) findActiveMemberBlacklistByScopeTx(
	ctx context.Context,
	tx pgx.Tx,
	input MemberBlacklistCreateInput,
) (*MemberBlacklistEntry, error) {
	row := tx.QueryRow(ctx, activeMemberBlacklistByScopeSQL(),
		input.Platform, input.SubjectType, input.SubjectID, input.ScopeType,
		nullableGuildID(input.ScopeType, input.GuildID),
	)
	return scanNullableMemberBlacklistEntry(row)
}

func (r *Repository) releaseExpiredMemberBlacklistByScopeTx(
	ctx context.Context,
	tx pgx.Tx,
	input MemberBlacklistCreateTxInput,
) error {
	entry := input.Entry
	_, err := tx.Exec(ctx, releaseExpiredMemberBlacklistByScopeSQL(),
		entry.Platform, entry.SubjectType, entry.SubjectID, entry.ScopeType,
		nullableGuildID(entry.ScopeType, entry.GuildID), input.Now,
	)
	return err
}
