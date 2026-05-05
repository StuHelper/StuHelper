package admission

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func scanMemberBlacklistEntry(row pgx.Row) (*MemberBlacklistEntry, error) {
	entry, err := scanMemberBlacklistEntryWithTotal(row, nil)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return entry, err
}

func scanMemberBlacklistEntryWithTotal(row pgx.Row, total *int) (*MemberBlacklistEntry, error) {
	var entry MemberBlacklistEntry
	var metadata []byte
	var releasedByType *string
	var releaseReasonCode *string
	dest := memberBlacklistScanDest(&entry, &metadata, &releasedByType, &releaseReasonCode)
	if total != nil {
		dest = append(dest, total)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
		return nil, fmt.Errorf("scan member blacklist metadata: %w", err)
	}
	entry.ReleasedByType = memberBlacklistActorTypePtr(releasedByType)
	entry.ReleaseReasonCode = memberBlacklistReleaseReasonPtr(releaseReasonCode)
	return &entry, nil
}

func memberBlacklistScanDest(
	entry *MemberBlacklistEntry,
	metadata *[]byte,
	releasedByType **string,
	releaseReasonCode **string,
) []any {
	return []any{
		&entry.ID, &entry.Platform, &entry.SubjectType, &entry.SubjectID, &entry.ScopeType, &entry.GuildID,
		&entry.Source, &entry.ReasonCode, &entry.ReasonText, &entry.CreatedByType, &entry.CreatedByID,
		&entry.CreatedFrom, &entry.ExpiresAt, &entry.ReleasedAt, releasedByType, &entry.ReleasedByID,
		releaseReasonCode, &entry.ReleaseReason, metadata, &entry.CreatedAt, &entry.UpdatedAt,
	}
}

func memberBlacklistActorTypePtr(value *string) *MemberBlacklistActorType {
	if value == nil {
		return nil
	}
	converted := MemberBlacklistActorType(*value)
	return &converted
}

func memberBlacklistReleaseReasonPtr(value *string) *MemberBlacklistReleaseReasonCode {
	if value == nil {
		return nil
	}
	converted := MemberBlacklistReleaseReasonCode(*value)
	return &converted
}

func scanMemberBlacklistEntryList(rows pgx.Rows) ([]MemberBlacklistEntry, int, error) {
	items := make([]MemberBlacklistEntry, 0)
	total := 0
	for rows.Next() {
		entry, err := scanMemberBlacklistEntryListRow(rows, &total)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *entry)
	}
	return items, total, rows.Err()
}

func scanMemberBlacklistEntryListRow(row pgx.Row, total *int) (*MemberBlacklistEntry, error) {
	entry, err := scanMemberBlacklistEntryWithTotal(row, total)
	if err != nil {
		return nil, fmt.Errorf("scan member blacklist entry: %w", err)
	}
	return entry, nil
}
