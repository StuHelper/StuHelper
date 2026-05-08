package admission

func memberBlacklistSelectSQL() string {
	return `
		SELECT id, platform, subject_type, subject_id, scope_type, guild_id, source,
		       reason_code, reason_text, created_by_type, created_by_id, created_from,
		       expires_at, released_at, released_by_type, released_by_id,
		       release_reason_code, release_reason, metadata, created_at, updated_at
		FROM member_blacklist_entries`
}

func activeMemberBlacklistSQL() string {
	return memberBlacklistSelectSQL() + `
		WHERE platform = $1 AND subject_type = $2 AND subject_id = $3
		  AND released_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
		  AND (scope_type = 'global' OR (scope_type = 'guild' AND guild_id = $4))
		ORDER BY CASE WHEN scope_type = 'global' THEN 0 ELSE 1 END, created_at DESC
		LIMIT 1`
}

func activeMemberBlacklistByScopeSQL() string {
	return memberBlacklistSelectSQL() + `
		WHERE platform = $1 AND subject_type = $2 AND subject_id = $3
		  AND scope_type = $4
		  AND (($4 = 'global' AND guild_id IS NULL) OR ($4 = 'guild' AND guild_id = $5))
		  AND released_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
		LIMIT 1`
}

func listMemberBlacklistSQL() string {
	return `
		SELECT id, platform, subject_type, subject_id, scope_type, guild_id, source,
		       reason_code, reason_text, created_by_type, created_by_id, created_from,
		       expires_at, released_at, released_by_type, released_by_id,
		       release_reason_code, release_reason, metadata, created_at, updated_at,
		       COUNT(*) OVER() AS total
		FROM member_blacklist_entries
		WHERE ($1::text = '' OR platform = $1)
		  AND ($2::text = '' OR subject_type = $2)
		  AND ($3::text = '' OR subject_id = $3)
		  AND ($4::text = '' OR scope_type = $4)
		  AND ($5::text = '' OR guild_id = $5)
		  AND ($6::bool = false OR (released_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())))
		ORDER BY created_at DESC, id ASC
		LIMIT $7 OFFSET $8`
}

func insertMemberBlacklistSQL() string {
	return `
		INSERT INTO member_blacklist_entries (
			id, platform, subject_type, subject_id, scope_type, guild_id, source,
			reason_code, reason_text, created_by_type, created_by_id, created_from,
			expires_at, metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $15)
		RETURNING id, platform, subject_type, subject_id, scope_type, guild_id, source,
		          reason_code, reason_text, created_by_type, created_by_id, created_from,
		          expires_at, released_at, released_by_type, released_by_id,
		          release_reason_code, release_reason, metadata, created_at, updated_at`
}

func releaseMemberBlacklistByIDSQL() string {
	return `
		UPDATE member_blacklist_entries
		SET released_at = $2, released_by_type = $3, released_by_id = $4,
		    release_reason_code = $5, release_reason = $6, updated_at = NOW()
		WHERE id = $1
		  AND released_at IS NULL
		  AND (expires_at IS NULL OR expires_at > $2)`
}

func releaseMemberBlacklistBySubjectSQL() string {
	return `
		UPDATE member_blacklist_entries
		SET released_at = $6, released_by_type = $7, released_by_id = $8,
		    release_reason_code = $9, release_reason = $10, updated_at = NOW()
		WHERE platform = $1 AND subject_type = $2 AND subject_id = $3 AND scope_type = $4
		  AND (($4 = 'global' AND guild_id IS NULL) OR ($4 = 'guild' AND guild_id = $5))
		  AND released_at IS NULL
		  AND (expires_at IS NULL OR expires_at > $6)`
}

func releaseExpiredMemberBlacklistByScopeSQL() string {
	return `
		UPDATE member_blacklist_entries
		SET released_at = $6, released_by_type = 'system', released_by_id = 'system',
		    release_reason_code = 'policy_expired_auto', release_reason = 'expired before replacement',
		    updated_at = NOW()
		WHERE platform = $1 AND subject_type = $2 AND subject_id = $3 AND scope_type = $4
		  AND (($4 = 'global' AND guild_id IS NULL) OR ($4 = 'guild' AND guild_id = $5))
		  AND released_at IS NULL
		  AND expires_at IS NOT NULL
		  AND expires_at <= $6`
}

func releaseAdmissionFailureMemberBlacklistSQL() string {
	return `
		UPDATE member_blacklist_entries
		SET released_at = $6, released_by_type = 'system', released_by_id = 'system',
		    release_reason_code = 'manual_pardon', release_reason = 'admission blacklist released',
		    updated_at = NOW()
		WHERE platform = $1 AND subject_type = $2 AND subject_id = $3
		  AND scope_type = $4 AND guild_id = $5
		  AND source = 'admission_failure'
		  AND released_at IS NULL
		  AND (expires_at IS NULL OR expires_at > $6)`
}
