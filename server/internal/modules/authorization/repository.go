package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
	"github.com/StuHelper/StuHelper/server/internal/pkg/id"
	"github.com/StuHelper/StuHelper/server/internal/pkg/outbox"
)

const authorizationGrantsTable = "authorization_grants"
const superAdminMutationAdvisoryLock int64 = 0x53545541555448

const grantColumns = `
	g.id,
	g.subject_user_id,
	g.role,
	g.school_id,
	g.section_id,
	g.desired_state,
	g.projection_status,
	g.revision,
	g.reason,
	g.created_by_user_id,
	g.updated_by_user_id,
	g.activated_at,
	g.revoked_at,
	g.projected_at,
	g.last_error,
	g.created_at,
	g.updated_at
`

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	if database == nil {
		panic("authorization.NewRepository: database is required")
	}
	return &Repository{db: database}
}

func (r *Repository) UserExists(ctx context.Context, userID int64) (bool, error) {
	ctx = db.WithTableHint(ctx, "users")
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check authorization user: %w", err)
	}
	return exists, nil
}

func (r *Repository) ResolveInternalUserID(ctx context.Context, casdoorSubject string) (int64, error) {
	ctx = db.WithTableHint(ctx, "users")
	var userID int64
	err := r.db.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE casdoor_subject = $1
	`, strings.TrimSpace(casdoorSubject)).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrActorUserNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("resolve authorization actor: %w", err)
	}
	return userID, nil
}

func (r *Repository) SchoolExists(ctx context.Context, schoolID int64) (bool, error) {
	ctx = db.WithTableHint(ctx, "schools")
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schools WHERE id = $1)`, schoolID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check authorization school: %w", err)
	}
	return exists, nil
}

func (r *Repository) CreateOrRestoreGrant(ctx context.Context, input CreateGrantInput) (MutationResult, error) {
	var result MutationResult
	err := r.db.WithTx(db.WithTableHint(ctx, authorizationGrantsTable), func(ctx context.Context, tx pgx.Tx) error {
		before, err := findGrantForUpdateTx(ctx, tx, input)
		if err != nil && !errors.Is(err, ErrGrantNotFound) {
			return err
		}

		switch {
		case errors.Is(err, ErrGrantNotFound):
			created, createErr := insertGrantTx(ctx, tx, input)
			if createErr != nil {
				return createErr
			}
			result = MutationResult{Grant: created, Changed: true}
		case before.DesiredState == DesiredGranted && before.ProjectionStatus == ProjectionApplied:
			result = MutationResult{Grant: before, Changed: false}
			return nil
		default:
			updated, updateErr := restoreGrantTx(ctx, tx, before, input)
			if updateErr != nil {
				return updateErr
			}
			result = MutationResult{Grant: updated, Changed: true}
		}

		if err := enqueueProjectionTx(ctx, tx, result.Grant); err != nil {
			return err
		}
		return insertGrantAuditTx(ctx, tx, "grant", input.ActorUserID, input.Reason, before, result.Grant)
	})
	if err != nil {
		return MutationResult{}, err
	}
	return result, nil
}

func (r *Repository) RevokeGrant(ctx context.Context, input RevokeGrantInput) (MutationResult, error) {
	var result MutationResult
	err := r.db.WithTx(db.WithTableHint(ctx, authorizationGrantsTable), func(ctx context.Context, tx pgx.Tx) error {
		before, err := getGrantForUpdateTx(ctx, tx, input.GrantID)
		if err != nil {
			return err
		}
		if before.DesiredState == DesiredRevoked && before.ProjectionStatus == ProjectionApplied {
			result = MutationResult{Grant: before, Changed: false}
			return nil
		}

		if err := ensureSuperAdminCanLeaveAppliedTx(ctx, tx, before); err != nil {
			return err
		}

		updated, err := revokeGrantTx(ctx, tx, before, input)
		if err != nil {
			return err
		}
		result = MutationResult{Grant: updated, Changed: true}
		if err := enqueueProjectionTx(ctx, tx, updated); err != nil {
			return err
		}
		return insertGrantAuditTx(ctx, tx, "revoke", input.ActorUserID, input.Reason, before, updated)
	})
	if err != nil {
		return MutationResult{}, err
	}
	return result, nil
}

func (r *Repository) ReconcileGrant(ctx context.Context, input ReconcileGrantInput) (MutationResult, error) {
	var result MutationResult
	err := r.db.WithTx(db.WithTableHint(ctx, authorizationGrantsTable), func(ctx context.Context, tx pgx.Tx) error {
		before, err := getGrantForUpdateTx(ctx, tx, input.GrantID)
		if err != nil {
			return err
		}
		if err := ensureSuperAdminCanLeaveAppliedTx(ctx, tx, before); err != nil {
			return err
		}

		updated, err := reconcileGrantTx(ctx, tx, before, input)
		if err != nil {
			return err
		}
		result = MutationResult{Grant: updated, Changed: true}
		if err := enqueueProjectionTx(ctx, tx, updated); err != nil {
			return err
		}
		return insertGrantAuditTx(ctx, tx, "reconcile", input.ActorUserID, input.Reason, before, updated)
	})
	if err != nil {
		return MutationResult{}, err
	}
	return result, nil
}

func (r *Repository) GetGrant(ctx context.Context, grantID int64) (Grant, error) {
	ctx = db.WithTableHint(ctx, authorizationGrantsTable)
	grant, err := scanGrantWithSubjectOnly(r.db.QueryRow(ctx, `
		SELECT `+grantColumns+`,
		       u.username,
		       COALESCE(NULLIF(u.username, ''), u.casdoor_subject)
		FROM authorization_grants g
		JOIN users u ON u.id = g.subject_user_id
		WHERE g.id = $1
	`, grantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Grant{}, ErrGrantNotFound
	}
	if err != nil {
		return Grant{}, fmt.Errorf("get authorization grant: %w", err)
	}
	return grant, nil
}

func (r *Repository) ListGrants(ctx context.Context, filter ListGrantsFilter) (GrantList, error) {
	ctx = db.WithTableHint(ctx, authorizationGrantsTable)
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	args := make([]any, 0, 5)
	clauses := []string{"TRUE"}
	add := func(value any, clause string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}
	if filter.SubjectUserID != nil {
		add(*filter.SubjectUserID, "g.subject_user_id = $%d")
	}
	if filter.Role != nil {
		add(*filter.Role, "g.role = $%d")
	}
	if filter.DesiredState != nil {
		add(*filter.DesiredState, "g.desired_state = $%d")
	}
	if filter.Projection != nil {
		add(*filter.Projection, "g.projection_status = $%d")
	}
	args = append(args, filter.Limit, filter.Offset)
	limitPosition := len(args) - 1
	offsetPosition := len(args)

	rows, err := r.db.Query(ctx, `
		SELECT `+grantColumns+`,
		       u.username,
		       COALESCE(NULLIF(u.username, ''), u.casdoor_subject),
		       COUNT(*) OVER()
		FROM authorization_grants g
		JOIN users u ON u.id = g.subject_user_id
		WHERE `+strings.Join(clauses, " AND ")+`
		ORDER BY g.updated_at DESC, g.id DESC
		LIMIT $`+strconv.Itoa(limitPosition)+` OFFSET $`+strconv.Itoa(offsetPosition),
		args...,
	)
	if err != nil {
		return GrantList{}, fmt.Errorf("list authorization grants: %w", err)
	}
	defer rows.Close()

	items := make([]Grant, 0, filter.Limit)
	total := 0
	for rows.Next() {
		grant, scanErr := scanGrantWithSubject(rows, &total)
		if scanErr != nil {
			return GrantList{}, scanErr
		}
		items = append(items, grant)
	}
	if err := rows.Err(); err != nil {
		return GrantList{}, fmt.Errorf("list authorization grants rows: %w", err)
	}
	return GrantList{Items: items, Total: total}, nil
}

func (r *Repository) ResolveAccessSnapshot(ctx context.Context, casdoorSubject string) (AccessSnapshot, error) {
	ctx = db.WithTableHint(ctx, authorizationGrantsTable)
	var snapshot AccessSnapshot
	var verifiedStudent bool
	var freshmanProvisional bool
	err := r.db.QueryRow(ctx, `
		SELECT
			u.id,
			EXISTS (
				SELECT 1
				FROM user_profiles p
				WHERE p.user_id = u.id
				  AND p.verification_status = 'verified'
			),
			EXISTS (
				SELECT 1
				FROM user_verification_credentials c
				WHERE c.user_id = u.id
				  AND c.kind = 'freshman_material_manual'
				  AND c.revoked_at IS NULL
				  AND (c.expires_at IS NULL OR c.expires_at > NOW())
			)
		FROM users u
		WHERE u.casdoor_subject = $1
	`, strings.TrimSpace(casdoorSubject)).Scan(
		&snapshot.InternalUserID,
		&verifiedStudent,
		&freshmanProvisional,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccessSnapshot{}, ErrActorUserNotFound
	}
	if err != nil {
		return AccessSnapshot{}, fmt.Errorf("resolve authorization subject: %w", err)
	}

	roleSet := map[string]struct{}{"user": {}}
	if verifiedStudent {
		roleSet["verified_student"] = struct{}{}
	}
	if freshmanProvisional {
		roleSet["freshman_provisional"] = struct{}{}
	}
	scopes := make(map[string][]string)

	rows, err := r.db.Query(ctx, `
		SELECT role, school_id, section_id
		FROM authorization_grants
		WHERE subject_user_id = $1
		  AND desired_state = 'granted'
		  AND projection_status = 'applied'
		ORDER BY role, school_id, section_id
	`, snapshot.InternalUserID)
	if err != nil {
		return AccessSnapshot{}, fmt.Errorf("load active authorization grants: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		var schoolID *int64
		var sectionID *string
		if err := rows.Scan(&role, &schoolID, &sectionID); err != nil {
			return AccessSnapshot{}, fmt.Errorf("scan active authorization grant: %w", err)
		}
		roleSet[role] = struct{}{}
		switch {
		case schoolID != nil && sectionID == nil:
			scopes[role] = append(scopes[role], strconv.FormatInt(*schoolID, 10))
		case sectionID != nil:
			scopes[role] = append(scopes[role], *sectionID)
		}
	}
	if err := rows.Err(); err != nil {
		return AccessSnapshot{}, fmt.Errorf("active authorization grant rows: %w", err)
	}

	snapshot.Roles = make([]string, 0, len(roleSet))
	for role := range roleSet {
		snapshot.Roles = append(snapshot.Roles, role)
	}
	sort.Strings(snapshot.Roles)
	for role := range scopes {
		sort.Strings(scopes[role])
	}
	if len(scopes) > 0 {
		snapshot.RoleScopes = scopes
	}
	return snapshot, nil
}

func findGrantForUpdateTx(ctx context.Context, tx pgx.Tx, input CreateGrantInput) (Grant, error) {
	grant, err := scanGrant(tx.QueryRow(ctx, `
		SELECT `+grantColumns+`
		FROM authorization_grants g
		WHERE g.subject_user_id = $1
		  AND g.role = $2
		  AND g.school_id IS NOT DISTINCT FROM $3
		  AND g.section_id IS NOT DISTINCT FROM $4
		FOR UPDATE
	`, input.SubjectUserID, input.Role, input.SchoolID, input.SectionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Grant{}, ErrGrantNotFound
	}
	if err != nil {
		return Grant{}, fmt.Errorf("find authorization grant: %w", err)
	}
	return grant, nil
}

func getGrantForUpdateTx(ctx context.Context, tx pgx.Tx, grantID int64) (Grant, error) {
	grant, err := scanGrant(tx.QueryRow(ctx, `
		SELECT `+grantColumns+`
		FROM authorization_grants g
		WHERE g.id = $1
		FOR UPDATE
	`, grantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Grant{}, ErrGrantNotFound
	}
	if err != nil {
		return Grant{}, fmt.Errorf("lock authorization grant: %w", err)
	}
	return grant, nil
}

func insertGrantTx(ctx context.Context, tx pgx.Tx, input CreateGrantInput) (Grant, error) {
	grant, err := scanGrant(tx.QueryRow(ctx, `
		INSERT INTO authorization_grants (
			subject_user_id, role, school_id, section_id,
			desired_state, projection_status, revision, reason,
			created_by_user_id, updated_by_user_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			'granted', 'pending', 1, $5,
			$6, $6,
			NOW(), NOW()
		)
		RETURNING `+strings.ReplaceAll(grantColumns, "g.", "")+`
	`, input.SubjectUserID, input.Role, input.SchoolID, input.SectionID, input.Reason, input.ActorUserID))
	if err != nil {
		return Grant{}, fmt.Errorf("insert authorization grant: %w", err)
	}
	return grant, nil
}

func restoreGrantTx(ctx context.Context, tx pgx.Tx, before Grant, input CreateGrantInput) (Grant, error) {
	grant, err := scanGrant(tx.QueryRow(ctx, `
		UPDATE authorization_grants
		SET desired_state = 'granted',
		    projection_status = 'pending',
		    revision = revision + 1,
		    reason = $2,
		    updated_by_user_id = $3,
		    activated_at = NULL,
		    revoked_at = NULL,
		    projected_at = NULL,
		    last_error = NULL,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING `+strings.ReplaceAll(grantColumns, "g.", "")+`
	`, before.ID, input.Reason, input.ActorUserID))
	if err != nil {
		return Grant{}, fmt.Errorf("restore authorization grant: %w", err)
	}
	return grant, nil
}

func revokeGrantTx(ctx context.Context, tx pgx.Tx, before Grant, input RevokeGrantInput) (Grant, error) {
	grant, err := scanGrant(tx.QueryRow(ctx, `
		UPDATE authorization_grants
		SET desired_state = 'revoked',
		    projection_status = 'pending',
		    revision = revision + 1,
		    reason = $2,
		    updated_by_user_id = $3,
		    revoked_at = NOW(),
		    projected_at = NULL,
		    last_error = NULL,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING `+strings.ReplaceAll(grantColumns, "g.", "")+`
	`, before.ID, input.Reason, input.ActorUserID))
	if err != nil {
		return Grant{}, fmt.Errorf("revoke authorization grant: %w", err)
	}
	return grant, nil
}

func reconcileGrantTx(
	ctx context.Context,
	tx pgx.Tx,
	before Grant,
	input ReconcileGrantInput,
) (Grant, error) {
	grant, err := scanGrant(tx.QueryRow(ctx, `
		UPDATE authorization_grants
		SET projection_status = 'pending',
		    revision = revision + 1,
		    reason = $2,
		    updated_by_user_id = $3,
		    projected_at = NULL,
		    last_error = NULL,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING `+strings.ReplaceAll(grantColumns, "g.", "")+`
	`, before.ID, input.Reason, input.ActorUserID))
	if err != nil {
		return Grant{}, fmt.Errorf("reconcile authorization grant: %w", err)
	}
	return grant, nil
}

func ensureSuperAdminCanLeaveAppliedTx(ctx context.Context, tx pgx.Tx, grant Grant) error {
	if grant.Role != RoleSuperAdmin ||
		grant.DesiredState != DesiredGranted ||
		grant.ProjectionStatus != ProjectionApplied {
		return nil
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, superAdminMutationAdvisoryLock); err != nil {
		return fmt.Errorf("lock super admin mutation: %w", err)
	}
	var activeCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM authorization_grants
		WHERE role = 'super_admin'
		  AND desired_state = 'granted'
		  AND projection_status = 'applied'
	`).Scan(&activeCount); err != nil {
		return fmt.Errorf("count active super admins: %w", err)
	}
	if activeCount <= 1 {
		return ErrLastSuperAdmin
	}
	return nil
}

func enqueueProjectionTx(ctx context.Context, tx pgx.Tx, grant Grant) error {
	payload, err := json.Marshal(ProjectionPayload{
		GrantID:      grant.ID,
		Revision:     grant.Revision,
		DesiredState: grant.DesiredState,
	})
	if err != nil {
		return fmt.Errorf("marshal authorization projection: %w", err)
	}
	if err := outbox.UpsertJobTx(
		ctx,
		tx,
		outbox.StreamIAMAuthorizationGrantProjection,
		ProjectionJobType,
		projectionDedupeKey(grant.ID),
		payload,
	); err != nil {
		return fmt.Errorf("enqueue authorization projection: %w", err)
	}
	return nil
}

func insertGrantAuditTx(
	ctx context.Context,
	tx pgx.Tx,
	action string,
	actorUserID int64,
	reason string,
	before Grant,
	after Grant,
) error {
	eventID, err := id.New()
	if err != nil {
		return err
	}
	event := audit.EventFromContext(ctx, audit.Event{
		Type:          audit.EventType("iam.authorization_grant." + action + "_requested"),
		Category:      "admin_operation",
		ActorType:     "admin",
		UserID:        strconv.FormatInt(actorUserID, 10),
		ResourceType:  "authorization_grant",
		ResourceID:    strconv.FormatInt(after.ID, 10),
		ScopeSchoolID: nullableSchoolID(after.SchoolID),
		Action:        action,
		Result:        "pending",
		Reason:        reason,
		Before:        nullableGrantSnapshot(before),
		After:         grantSnapshot(after),
		Details: map[string]any{
			"target_user_id": after.SubjectUserID,
			"role":           after.Role,
			"revision":       after.Revision,
			"desired_state":  after.DesiredState,
			"section_id":     after.SectionID,
		},
	})
	beforeJSON, err := json.Marshal(event.Before)
	if err != nil {
		return fmt.Errorf("marshal authorization audit before: %w", err)
	}
	afterJSON, err := json.Marshal(event.After)
	if err != nil {
		return fmt.Errorf("marshal authorization audit after: %w", err)
	}
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		return fmt.Errorf("marshal authorization audit details: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (
			id, category, event_type, actor_type, actor_user_id, actor_username,
			action, resource_type, resource_id, scope_school_id,
			before_data, after_data, result, reason, trace_id, request_id,
			ip_address, user_agent, details, created_at
		) VALUES (
			$1, $2, $3, $4, $5, '',
			$6, $7, $8, NULLIF($9, ''),
			$10::jsonb, $11::jsonb, $12, $13, NULLIF($14, ''), NULLIF($15, ''),
			NULL, NULL, $16::jsonb, $17
		)
	`,
		eventID, event.Category, event.Type, event.ActorType, event.UserID,
		event.Action, event.ResourceType, event.ResourceID, event.ScopeSchoolID,
		beforeJSON, afterJSON, event.Result, event.Reason, event.TraceID, event.RequestID,
		detailsJSON, event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert authorization audit: %w", err)
	}
	return nil
}

func nullableSchoolID(schoolID *int64) string {
	if schoolID == nil {
		return ""
	}
	return strconv.FormatInt(*schoolID, 10)
}

func nullableGrantSnapshot(grant Grant) any {
	if grant.ID == 0 {
		return nil
	}
	return grantSnapshot(grant)
}

func grantSnapshot(grant Grant) map[string]any {
	return map[string]any{
		"id":                grant.ID,
		"subject_user_id":   grant.SubjectUserID,
		"role":              grant.Role,
		"school_id":         grant.SchoolID,
		"section_id":        grant.SectionID,
		"desired_state":     grant.DesiredState,
		"projection_status": grant.ProjectionStatus,
		"revision":          grant.Revision,
	}
}

func scanGrant(row pgx.Row) (Grant, error) {
	var grant Grant
	err := row.Scan(
		&grant.ID,
		&grant.SubjectUserID,
		&grant.Role,
		&grant.SchoolID,
		&grant.SectionID,
		&grant.DesiredState,
		&grant.ProjectionStatus,
		&grant.Revision,
		&grant.Reason,
		&grant.CreatedByUserID,
		&grant.UpdatedByUserID,
		&grant.ActivatedAt,
		&grant.RevokedAt,
		&grant.ProjectedAt,
		&grant.LastError,
		&grant.CreatedAt,
		&grant.UpdatedAt,
	)
	return grant, err
}

func scanGrantWithSubject(row pgx.Row, total *int) (Grant, error) {
	var grant Grant
	err := row.Scan(
		&grant.ID,
		&grant.SubjectUserID,
		&grant.Role,
		&grant.SchoolID,
		&grant.SectionID,
		&grant.DesiredState,
		&grant.ProjectionStatus,
		&grant.Revision,
		&grant.Reason,
		&grant.CreatedByUserID,
		&grant.UpdatedByUserID,
		&grant.ActivatedAt,
		&grant.RevokedAt,
		&grant.ProjectedAt,
		&grant.LastError,
		&grant.CreatedAt,
		&grant.UpdatedAt,
		&grant.SubjectUsername,
		&grant.SubjectDisplayName,
		total,
	)
	if err != nil {
		return Grant{}, fmt.Errorf("scan authorization grant list: %w", err)
	}
	return grant, nil
}

func scanGrantWithSubjectOnly(row pgx.Row) (Grant, error) {
	var grant Grant
	err := row.Scan(
		&grant.ID,
		&grant.SubjectUserID,
		&grant.Role,
		&grant.SchoolID,
		&grant.SectionID,
		&grant.DesiredState,
		&grant.ProjectionStatus,
		&grant.Revision,
		&grant.Reason,
		&grant.CreatedByUserID,
		&grant.UpdatedByUserID,
		&grant.ActivatedAt,
		&grant.RevokedAt,
		&grant.ProjectedAt,
		&grant.LastError,
		&grant.CreatedAt,
		&grant.UpdatedAt,
		&grant.SubjectUsername,
		&grant.SubjectDisplayName,
	)
	if err != nil {
		return Grant{}, fmt.Errorf("scan authorization grant: %w", err)
	}
	return grant, nil
}
