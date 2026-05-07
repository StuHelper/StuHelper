package admission

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

const memberBlacklistColumns = `
	id, platform, subject_type, subject_id, scope_type, guild_id, source,
	reason_code, reason_text, created_by_type, created_by_id, created_from,
	expires_at, released_at, released_by_type, released_by_id,
	release_reason_code, release_reason, metadata, created_at, updated_at`

const memberBlacklistCreateSavepoint = "member_blacklist_create"

func (r *Repository) CreateMemberBlacklistTx(
	ctx context.Context,
	tx pgx.Tx,
	input MemberBlacklistCreateInput,
	now time.Time,
) (*MemberBlacklistEntry, error) {
	entryID, err := id.New()
	if err != nil {
		return nil, err
	}
	entry, err := scanMemberBlacklistEntry(tx.QueryRow(ctx, createMemberBlacklistSQL(), createMemberBlacklistArgs(entryID, input, now)...))
	if err != nil {
		return nil, fmt.Errorf("CreateMemberBlacklistTx: %w", err)
	}
	return entry, nil
}

func (r *Repository) CreateMemberBlacklistSavepointTx(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "SAVEPOINT "+memberBlacklistCreateSavepoint)
	if err != nil {
		return fmt.Errorf("CreateMemberBlacklistSavepointTx: %w", err)
	}
	return nil
}

func (r *Repository) RollbackMemberBlacklistCreateSavepointTx(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+memberBlacklistCreateSavepoint)
	if err != nil {
		return fmt.Errorf("RollbackMemberBlacklistCreateSavepointTx: %w", err)
	}
	return nil
}

func (r *Repository) ReleaseMemberBlacklistCreateSavepointTx(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "RELEASE SAVEPOINT "+memberBlacklistCreateSavepoint)
	if err != nil {
		return fmt.Errorf("ReleaseMemberBlacklistCreateSavepointTx: %w", err)
	}
	return nil
}

func (r *Repository) GetActiveMemberBlacklistByKeyTx(
	ctx context.Context,
	tx pgx.Tx,
	key memberBlacklistKey,
	now time.Time,
) (*MemberBlacklistEntry, error) {
	return scanMemberBlacklistEntry(tx.QueryRow(ctx, memberBlacklistByKeySQL(activeMemberBlacklistCondition()), memberBlacklistKeyArgs(key, now)...))
}

func (r *Repository) GetMemberBlacklistAccess(
	ctx context.Context,
	query MemberBlacklistAccessQuery,
	now time.Time,
) (*MemberBlacklistEntry, error) {
	return scanMemberBlacklistEntry(r.db.QueryRow(ctx, memberBlacklistAccessSQL(), query.Platform, query.SubjectType, query.SubjectID, query.GuildID, now))
}

func (r *Repository) ListMemberBlacklist(
	ctx context.Context,
	filter MemberBlacklistListFilter,
	now time.Time,
) ([]MemberBlacklistEntry, int, error) {
	query, args := memberBlacklistListSQL(filter, now)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListMemberBlacklist: %w", err)
	}
	defer rows.Close()
	return scanMemberBlacklistEntryList(rows)
}

func (r *Repository) ReleaseExpiredMemberBlacklistTx(
	ctx context.Context,
	tx pgx.Tx,
	key memberBlacklistKey,
	now time.Time,
) ([]MemberBlacklistEntry, error) {
	rows, err := tx.Query(ctx, releaseExpiredMemberBlacklistSQL(), releaseExpiredMemberBlacklistArgs(key, now)...)
	if err != nil {
		return nil, fmt.Errorf("ReleaseExpiredMemberBlacklistTx: %w", err)
	}
	defer rows.Close()
	items, _, err := scanMemberBlacklistEntryList(rows)
	return items, err
}

func (r *Repository) ReleaseMemberBlacklistByIDTx(
	ctx context.Context,
	tx pgx.Tx,
	input MemberBlacklistReleaseInput,
	now time.Time,
) (*MemberBlacklistEntry, error) {
	entry, err := scanMemberBlacklistEntry(tx.QueryRow(ctx, releaseMemberBlacklistByIDSQL(), releaseByIDArgs(input, now)...))
	return mapMemberBlacklistReleaseResult("ReleaseMemberBlacklistByIDTx", entry, err)
}

func (r *Repository) ReleaseMemberBlacklistBySubjectTx(
	ctx context.Context,
	tx pgx.Tx,
	input MemberBlacklistReleaseBySubjectInput,
	now time.Time,
) (*MemberBlacklistEntry, error) {
	entry, err := scanMemberBlacklistEntry(tx.QueryRow(ctx, releaseMemberBlacklistBySubjectSQL(), releaseBySubjectArgs(input, now)...))
	return mapMemberBlacklistReleaseResult("ReleaseMemberBlacklistBySubjectTx", entry, err)
}

func (r *Repository) ListExpiredMemberBlacklist(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]MemberBlacklistEntry, error) {
	rows, err := r.db.Query(ctx, expiredMemberBlacklistSQL(), now, limit)
	if err != nil {
		return nil, fmt.Errorf("ListExpiredMemberBlacklist: %w", err)
	}
	defer rows.Close()
	items, _, err := scanMemberBlacklistEntryList(rows)
	return items, err
}

func createMemberBlacklistSQL() string {
	return `INSERT INTO member_blacklist_entries (` + memberBlacklistColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
		        $12, $13, NULL, NULL, NULL, NULL, NULL, $14, $15, $15)
		RETURNING ` + memberBlacklistColumns
}

func createMemberBlacklistArgs(id string, input MemberBlacklistCreateInput, now time.Time) []any {
	return []any{
		id, input.Platform, input.SubjectType, input.SubjectID, input.ScopeType, input.GuildID,
		input.Source, input.ReasonCode, input.ReasonText, input.CreatedByType, input.CreatedByID,
		input.CreatedFrom, input.ExpiresAt, memberBlacklistMetadata(input.Metadata), now,
	}
}

func memberBlacklistByKeySQL(condition string) string {
	return `SELECT ` + memberBlacklistColumns + `
		FROM member_blacklist_entries
		WHERE platform = $1 AND subject_type = $2 AND subject_id = $3 AND scope_type = $4
		  AND (($5::text IS NULL AND guild_id IS NULL) OR guild_id = $5)
		  AND ` + condition + `
		ORDER BY created_at DESC
		LIMIT 1`
}

func activeMemberBlacklistCondition() string {
	return `released_at IS NULL AND (expires_at IS NULL OR expires_at > $6)`
}

func memberBlacklistAccessSQL() string {
	return `SELECT ` + memberBlacklistColumns + `
		FROM member_blacklist_entries
		WHERE platform = $1 AND subject_type = $2 AND subject_id = $3
		  AND released_at IS NULL AND (expires_at IS NULL OR expires_at > $5)
		  AND (scope_type = 'global' OR (scope_type = 'guild' AND guild_id = $4))
		ORDER BY CASE WHEN scope_type = 'global' THEN 0 ELSE 1 END, created_at DESC
		LIMIT 1`
}

func memberBlacklistListSQL(filter MemberBlacklistListFilter, now time.Time) (string, []any) {
	clauses, args := memberBlacklistListClauses(filter, now)
	args = append(args, filter.PageSize, filter.Offset)
	return fmt.Sprintf(`SELECT `+memberBlacklistColumns+`, COUNT(*) OVER() AS total
		FROM member_blacklist_entries
		WHERE %s
		ORDER BY created_at DESC, id ASC
		LIMIT $%d OFFSET $%d`, strings.Join(clauses, " AND "), len(args)-1, len(args)), args
}

func memberBlacklistListClauses(filter MemberBlacklistListFilter, now time.Time) ([]string, []any) {
	clauses := []string{"TRUE"}
	args := make([]any, 0, 9)
	addMemberBlacklistStringFilter(&clauses, &args, "platform", filter.Platform)
	addMemberBlacklistStringFilter(&clauses, &args, "subject_type", string(filter.SubjectType))
	addMemberBlacklistStringFilter(&clauses, &args, "subject_id", filter.SubjectID)
	addMemberBlacklistStringFilter(&clauses, &args, "scope_type", string(filter.ScopeType))
	addMemberBlacklistStringFilter(&clauses, &args, "source", string(filter.Source))
	addMemberBlacklistStringFilter(&clauses, &args, "guild_id", filter.GuildID)
	addMemberBlacklistStringFilter(&clauses, &args, "created_by_id", filter.CreatedByID)
	return append(clauses, memberBlacklistStatusClause(filter.Status, &args, now)), args
}

func addMemberBlacklistStringFilter(clauses *[]string, args *[]any, column string, value string) {
	if value == "" {
		return
	}
	*args = append(*args, value)
	*clauses = append(*clauses, fmt.Sprintf("%s = $%d", column, len(*args)))
}

func memberBlacklistStatusClause(status MemberBlacklistStatus, args *[]any, now time.Time) string {
	*args = append(*args, now)
	switch status {
	case BlacklistStatusAll:
		return "TRUE"
	case BlacklistStatusReleased:
		return "released_at IS NOT NULL"
	case BlacklistStatusExpired:
		return fmt.Sprintf("released_at IS NULL AND expires_at IS NOT NULL AND expires_at <= $%d", len(*args))
	default:
		return fmt.Sprintf("released_at IS NULL AND (expires_at IS NULL OR expires_at > $%d)", len(*args))
	}
}

func releaseExpiredMemberBlacklistSQL() string {
	return `UPDATE member_blacklist_entries
		SET released_at = $6, released_by_type = $7, released_by_id = $8,
		    release_reason_code = $9, updated_at = $6
		WHERE platform = $1 AND subject_type = $2 AND subject_id = $3 AND scope_type = $4
		  AND (($5::text IS NULL AND guild_id IS NULL) OR guild_id = $5)
		  AND released_at IS NULL AND expires_at IS NOT NULL AND expires_at <= $6
		RETURNING ` + memberBlacklistColumns + `, 0 AS total`
}

func releaseMemberBlacklistByIDSQL() string {
	return `UPDATE member_blacklist_entries
		SET released_at = $2, released_by_type = $3, released_by_id = $4,
		    release_reason_code = $5, release_reason = $6, updated_at = $2
		WHERE id = $1 AND released_at IS NULL
		RETURNING ` + memberBlacklistColumns
}

func releaseMemberBlacklistBySubjectSQL() string {
	return `UPDATE member_blacklist_entries
		SET released_at = $6, released_by_type = $7, released_by_id = $8,
		    release_reason_code = $9, release_reason = $10, updated_at = $6
		WHERE platform = $1 AND subject_type = $2 AND subject_id = $3 AND scope_type = $4
		  AND (($5::text IS NULL AND guild_id IS NULL) OR guild_id = $5)
		  AND released_at IS NULL
		RETURNING ` + memberBlacklistColumns
}

func expiredMemberBlacklistSQL() string {
	return `SELECT ` + memberBlacklistColumns + `, 0 AS total
		FROM member_blacklist_entries
		WHERE released_at IS NULL AND expires_at IS NOT NULL AND expires_at <= $1
		ORDER BY expires_at ASC, id ASC
		LIMIT $2`
}

func memberBlacklistKeyArgs(key memberBlacklistKey, now time.Time) []any {
	return []any{key.Platform, key.SubjectType, key.SubjectID, key.ScopeType, key.GuildID, now}
}

func releaseExpiredMemberBlacklistArgs(key memberBlacklistKey, now time.Time) []any {
	return append(memberBlacklistKeyArgs(key, now), BlacklistActorSystem, "system", BlacklistReleasePolicyExpiredAuto)
}

func releaseByIDArgs(input MemberBlacklistReleaseInput, now time.Time) []any {
	return []any{input.ID, now, input.ReleasedByType, input.ReleasedByID, input.ReleaseReasonCode, input.ReleaseReason}
}

func releaseBySubjectArgs(input MemberBlacklistReleaseBySubjectInput, now time.Time) []any {
	return []any{
		input.Platform, input.SubjectType, input.SubjectID, input.ScopeType, input.GuildID,
		now, input.ReleasedByType, input.ReleasedByID, input.ReleaseReasonCode, input.ReleaseReason,
	}
}

func memberBlacklistMetadata(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func mapMemberBlacklistReleaseResult(op string, entry *MemberBlacklistEntry, err error) (*MemberBlacklistEntry, error) {
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if entry == nil {
		return nil, ErrMemberBlacklistNotFound
	}
	return entry, nil
}
