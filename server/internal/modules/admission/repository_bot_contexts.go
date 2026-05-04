package admission

import (
	"context"
	"fmt"
)

func (r *Repository) ListPoliciesByGuildKeys(
	ctx context.Context,
	keys []admissionGuildKey,
) (map[admissionGuildKey]*AdmissionPolicy, error) {
	result := make(map[admissionGuildKey]*AdmissionPolicy, len(keys))
	if len(keys) == 0 {
		return result, nil
	}
	rows, err := r.db.Query(ctx, listPoliciesByGuildKeysSQL(), admissionGuildKeyArrays(keys)...)
	if err != nil {
		return nil, fmt.Errorf("ListPoliciesByGuildKeys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		policy, err := scanAdmissionPolicy(rows)
		if err != nil {
			return nil, err
		}
		result[admissionGuildKey{Platform: policy.Platform, GuildID: policy.GuildID}] = policy
	}
	return result, rows.Err()
}

func (r *Repository) ListAdmissionFailuresByKeys(
	ctx context.Context,
	keys []admissionFailureKey,
) (map[admissionFailureKey]*AdmissionFailure, error) {
	result := make(map[admissionFailureKey]*AdmissionFailure, len(keys))
	if len(keys) == 0 {
		return result, nil
	}
	rows, err := r.db.Query(ctx, listAdmissionFailuresByKeysSQL(), admissionFailureKeyArrays(keys)...)
	if err != nil {
		return nil, fmt.Errorf("ListAdmissionFailuresByKeys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		failure, err := scanAdmissionFailure(rows)
		if err != nil {
			return nil, err
		}
		key := admissionFailureKey{Platform: failure.Platform, GuildID: failure.GuildID, QQID: failure.QQID}
		result[key] = failure
	}
	return result, rows.Err()
}

func listPoliciesByGuildKeysSQL() string {
	return `
		WITH wanted(platform, guild_id) AS (
			SELECT * FROM unnest($1::text[], $2::text[])
		)
		SELECT id, platform, guild_id, school_id, auto_approve_join, initial_mute_duration_seconds,
		       link_wait_seconds, submission_wait_seconds, manual_review_timeout_seconds,
		       reminder_interval_seconds, failed_join_limit, blacklist_duration_seconds,
		       freshman_channel_enabled, freshman_channel_closes_at, freshman_default_expires_at,
		       forward_raw_material_to_qq, management_guild_ids, max_material_bytes, max_extension_days
		FROM group_admission_policies p
		JOIN wanted w USING (platform, guild_id)
	`
}

func listAdmissionFailuresByKeysSQL() string {
	return `
		WITH wanted(platform, guild_id, qq_id) AS (
			SELECT * FROM unnest($1::text[], $2::text[], $3::text[])
		)
		SELECT platform, guild_id, qq_id, failure_count, blacklisted_at, blacklist_expires_at, released_at
		FROM group_admission_failures f
		JOIN wanted w USING (platform, guild_id, qq_id)
		WHERE released_at IS NULL
	`
}

func admissionGuildKeyArrays(keys []admissionGuildKey) []any {
	platforms := make([]string, 0, len(keys))
	guildIDs := make([]string, 0, len(keys))
	for _, key := range keys {
		platforms = append(platforms, key.Platform)
		guildIDs = append(guildIDs, key.GuildID)
	}
	return []any{platforms, guildIDs}
}

func admissionFailureKeyArrays(keys []admissionFailureKey) []any {
	platforms := make([]string, 0, len(keys))
	guildIDs := make([]string, 0, len(keys))
	qqIDs := make([]string, 0, len(keys))
	for _, key := range keys {
		platforms = append(platforms, key.Platform)
		guildIDs = append(guildIDs, key.GuildID)
		qqIDs = append(qqIDs, key.QQID)
	}
	return []any{platforms, guildIDs, qqIDs}
}
