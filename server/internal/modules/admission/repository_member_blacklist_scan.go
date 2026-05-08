package admission

import (
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

func scanNullableMemberBlacklistEntry(row pgx.Row) (*MemberBlacklistEntry, error) {
	entry, err := scanMemberBlacklistEntry(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return entry, err
}

func scanMemberBlacklistEntryList(rows pgx.Rows) ([]MemberBlacklistEntry, int, error) {
	items := make([]MemberBlacklistEntry, 0)
	total := 0
	for rows.Next() {
		entry, rowTotal, err := scanMemberBlacklistEntryWithTotal(rows)
		if err != nil {
			return nil, 0, err
		}
		total = rowTotal
		items = append(items, *entry)
	}
	return items, total, rows.Err()
}

func scanMemberBlacklistEntryWithTotal(row pgx.Row) (*MemberBlacklistEntry, int, error) {
	entry := &MemberBlacklistEntry{}
	total := 0
	err := scanMemberBlacklistEntryFields(row, entry, &total)
	return entry, total, err
}

func scanMemberBlacklistEntry(row pgx.Row) (*MemberBlacklistEntry, error) {
	entry := &MemberBlacklistEntry{}
	err := scanMemberBlacklistEntryFields(row, entry, nil)
	return entry, err
}

func scanMemberBlacklistEntryFields(row pgx.Row, entry *MemberBlacklistEntry, total *int) error {
	var metadata []byte
	var releasedByType *string
	targets := memberBlacklistScanTargets(entry, &releasedByType, &metadata)
	if total != nil {
		targets = append(targets, total)
	}
	if err := row.Scan(targets...); err != nil {
		return err
	}
	if releasedByType != nil {
		value := MemberBlacklistActorType(*releasedByType)
		entry.ReleasedByType = &value
	}
	return json.Unmarshal(metadata, &entry.Metadata)
}

func memberBlacklistScanTargets(
	entry *MemberBlacklistEntry,
	releasedByType **string,
	metadata *[]byte,
) []any {
	return []any{
		&entry.ID, &entry.Platform, &entry.SubjectType, &entry.SubjectID,
		&entry.ScopeType, &entry.GuildID, &entry.Source, &entry.ReasonCode,
		&entry.ReasonText, &entry.CreatedByType, &entry.CreatedByID,
		&entry.CreatedFrom, &entry.ExpiresAt, &entry.ReleasedAt,
		releasedByType, &entry.ReleasedByID, &entry.ReleaseReasonCode,
		&entry.ReleaseReason, metadata, &entry.CreatedAt, &entry.UpdatedAt,
	}
}
