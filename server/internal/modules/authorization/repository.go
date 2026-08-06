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

const (
	repositoryDefaultGrantListLimit = 50
	repositoryMaxGrantListLimit     = 200
)

const casdoorOrganizationAdminSyncReason = "Casdoor StuHelper organization administrator status synchronized"

var errConcurrentGrantInsert = errors.New("authorization grant was inserted concurrently")

type grantAuditActor struct {
	actorType string
	userID    string
}

func adminGrantAuditActor(userID int64) grantAuditActor {
	return grantAuditActor{
		actorType: "admin",
		userID:    strconv.FormatInt(userID, 10),
	}
}

var casdoorOrganizationAdminGrantAuditActor = grantAuditActor{
	actorType: "system",
	userID:    "casdoor-org-admin-sync",
}

var reconciliationGrantAuditActor = grantAuditActor{
	actorType: "system",
	userID:    "authorization-reconciliation",
}

const grantColumns = `
	g.id,
	g.subject_user_id,
	g.role,
	g.source,
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

		if errors.Is(err, ErrGrantNotFound) {
			created, createErr := insertGrantTx(ctx, tx, input)
			if errors.Is(createErr, errConcurrentGrantInsert) {
				before, createErr = findGrantForUpdateTx(ctx, tx, input)
			}
			if createErr != nil {
				return createErr
			}
			if before.ID == 0 {
				result = MutationResult{Grant: created, Changed: true}
			}
		}

		switch {
		case result.Changed:
			// The insert path already populated the mutation result.
		case before.DesiredState == DesiredGranted:
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
		return insertGrantAuditTx(
			ctx,
			tx,
			"grant",
			adminGrantAuditActor(input.ActorUserID),
			input.Reason,
			before,
			result.Grant,
		)
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
		if before.DesiredState == DesiredRevoked {
			result = MutationResult{Grant: before, Changed: false}
			return nil
		}

		updated, err := revokeGrantTx(ctx, tx, before, input)
		if err != nil {
			return err
		}
		result = MutationResult{Grant: updated, Changed: true}
		if err := enqueueProjectionTx(ctx, tx, updated); err != nil {
			return err
		}
		return insertGrantAuditTx(
			ctx,
			tx,
			"revoke",
			adminGrantAuditActor(input.ActorUserID),
			input.Reason,
			before,
			updated,
		)
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
		updated, err := reconcileGrantTx(ctx, tx, before, input)
		if err != nil {
			return err
		}
		result = MutationResult{Grant: updated, Changed: true}
		if err := enqueueProjectionTx(ctx, tx, updated); err != nil {
			return err
		}
		return insertGrantAuditTx(
			ctx,
			tx,
			"reconcile",
			adminGrantAuditActor(input.ActorUserID),
			input.Reason,
			before,
			updated,
		)
	})
	if err != nil {
		return MutationResult{}, err
	}
	return result, nil
}

func (r *Repository) ReconcileAll(ctx context.Context, input ReconcileAllInput) (int, error) {
	queued := 0
	err := r.db.WithTx(db.WithTableHint(ctx, authorizationGrantsTable), func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+grantColumns+`
			FROM authorization_grants g
			ORDER BY g.id
			FOR UPDATE
		`)
		if err != nil {
			return fmt.Errorf("lock authorization grants for rebuild: %w", err)
		}
		grants := make([]Grant, 0)
		for rows.Next() {
			grant, scanErr := scanGrant(rows)
			if scanErr != nil {
				rows.Close()
				return fmt.Errorf("scan authorization grant for rebuild: %w", scanErr)
			}
			grants = append(grants, grant)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("authorization rebuild rows: %w", err)
		}
		rows.Close()

		for _, before := range grants {
			after, err := reconcileGrantTx(ctx, tx, before, ReconcileGrantInput{
				GrantID:     before.ID,
				Reason:      input.Reason,
				ActorUserID: input.ActorUserID,
			})
			if err != nil {
				return err
			}
			if err := enqueueProjectionTx(ctx, tx, after); err != nil {
				return err
			}
			if err := insertGrantAuditTx(
				ctx,
				tx,
				"reconcile",
				adminGrantAuditActor(input.ActorUserID),
				input.Reason,
				before,
				after,
			); err != nil {
				return err
			}
			queued++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return queued, nil
}

func (r *Repository) ListGrantsForReconciliation(
	ctx context.Context,
	afterID int64,
	limit int,
) ([]Grant, error) {
	ctx = db.WithTableHint(ctx, authorizationGrantsTable)
	if afterID < 0 || limit <= 0 || limit > 500 {
		return nil, ErrInvalidGrant
	}
	rows, err := r.db.Query(ctx, `
		SELECT `+grantColumns+`
		FROM authorization_grants g
		WHERE g.id > $1
		ORDER BY g.id
		LIMIT $2
	`, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("list authorization grants for reconciliation: %w", err)
	}
	defer rows.Close()

	grants := make([]Grant, 0, limit)
	for rows.Next() {
		grant, scanErr := scanGrant(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan authorization grant for reconciliation: %w", scanErr)
		}
		grants = append(grants, grant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("authorization reconciliation rows: %w", err)
	}
	return grants, nil
}

func (r *Repository) ReconcileGrantsAsSystem(
	ctx context.Context,
	grantIDs []int64,
	reason string,
) (int, error) {
	if len(grantIDs) == 0 {
		return 0, nil
	}
	queued := 0
	err := r.db.WithTx(db.WithTableHint(ctx, authorizationGrantsTable), func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+grantColumns+`
			FROM authorization_grants g
			WHERE g.id = ANY($1::bigint[])
			ORDER BY g.id
			FOR UPDATE
		`, grantIDs)
		if err != nil {
			return fmt.Errorf("lock drifted authorization grants: %w", err)
		}
		grants := make([]Grant, 0, len(grantIDs))
		for rows.Next() {
			grant, scanErr := scanGrant(rows)
			if scanErr != nil {
				rows.Close()
				return fmt.Errorf("scan drifted authorization grant: %w", scanErr)
			}
			grants = append(grants, grant)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("drifted authorization grant rows: %w", err)
		}
		rows.Close()
		if len(grants) != len(grantIDs) {
			return ErrGrantNotFound
		}

		for _, before := range grants {
			after, err := reconcileGrantTx(ctx, tx, before, ReconcileGrantInput{
				GrantID: before.ID,
				Reason:  reason,
			})
			if err != nil {
				return err
			}
			if err := enqueueProjectionTx(ctx, tx, after); err != nil {
				return err
			}
			if err := insertGrantAuditTx(
				ctx,
				tx,
				"reconcile",
				reconciliationGrantAuditActor,
				reason,
				before,
				after,
			); err != nil {
				return err
			}
			queued++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return queued, nil
}

func (r *Repository) SyncCasdoorOrganizationAdmin(
	ctx context.Context,
	input CasdoorOrganizationAdminSyncInput,
) (MutationResult, error) {
	var result MutationResult
	err := r.db.WithTx(db.WithTableHint(ctx, authorizationGrantsTable), func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(
			ctx,
			`SELECT pg_advisory_xact_lock($1)`,
			superAdminMutationAdvisoryLock,
		); err != nil {
			return fmt.Errorf("lock Casdoor organization administrator sync: %w", err)
		}

		createInput := CreateGrantInput{
			SubjectUserID: input.SubjectUserID,
			Role:          RoleSuperAdmin,
			Reason:        casdoorOrganizationAdminSyncReason,
			Source:        GrantSourceCasdoorOrganizationAdmin,
		}
		before, findErr := findGrantForUpdateTx(ctx, tx, createInput)
		if findErr != nil && !errors.Is(findErr, ErrGrantNotFound) {
			return findErr
		}

		action := "provider_grant"
		if input.OrganizationAdmin {
			switch {
			case errors.Is(findErr, ErrGrantNotFound):
				created, createErr := insertGrantTx(ctx, tx, createInput)
				if createErr != nil {
					return createErr
				}
				result = MutationResult{Grant: created, Changed: true}
			case before.Source != GrantSourceCasdoorOrganizationAdmin:
				return ErrProviderManagedRole
			case before.DesiredState == DesiredGranted:
				if before.ProjectionStatus == ProjectionFailed {
					reconciled, reconcileErr := reconcileGrantTx(ctx, tx, before, ReconcileGrantInput{
						GrantID: before.ID,
						Reason:  casdoorOrganizationAdminSyncReason,
					})
					if reconcileErr != nil {
						return reconcileErr
					}
					result = MutationResult{Grant: reconciled, Changed: true}
					action = "provider_reconcile"
					break
				}
				result = MutationResult{Grant: before, Changed: false}
				return nil
			default:
				restored, restoreErr := restoreGrantTx(ctx, tx, before, createInput)
				if restoreErr != nil {
					return restoreErr
				}
				result = MutationResult{Grant: restored, Changed: true}
			}
		} else {
			action = "provider_revoke"
			if errors.Is(findErr, ErrGrantNotFound) {
				return nil
			}
			if before.Source != GrantSourceCasdoorOrganizationAdmin {
				return ErrProviderManagedRole
			}
			if before.DesiredState == DesiredRevoked {
				if before.ProjectionStatus == ProjectionFailed {
					reconciled, reconcileErr := reconcileGrantTx(ctx, tx, before, ReconcileGrantInput{
						GrantID: before.ID,
						Reason:  casdoorOrganizationAdminSyncReason,
					})
					if reconcileErr != nil {
						return reconcileErr
					}
					result = MutationResult{Grant: reconciled, Changed: true}
					action = "provider_reconcile"
				} else {
					result = MutationResult{Grant: before, Changed: false}
					return nil
				}
			} else {
				revoked, revokeErr := revokeGrantTx(ctx, tx, before, RevokeGrantInput{
					GrantID: before.ID,
					Reason:  casdoorOrganizationAdminSyncReason,
				})
				if revokeErr != nil {
					return revokeErr
				}
				result = MutationResult{Grant: revoked, Changed: true}
			}
		}

		if err := enqueueProjectionTx(ctx, tx, result.Grant); err != nil {
			return err
		}
		return insertGrantAuditTx(
			ctx,
			tx,
			action,
			casdoorOrganizationAdminGrantAuditActor,
			casdoorOrganizationAdminSyncReason,
			before,
			result.Grant,
		)
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
		       COALESCE(NULLIF(u.username, ''), 'user-' || u.id::text)
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
	filter.Limit = normalizeRepositoryGrantListLimit(filter.Limit)
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	args := make([]any, 0, 6)
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
	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM authorization_grants g
		JOIN users u ON u.id = g.subject_user_id
		WHERE `+strings.Join(clauses, " AND "), args...).Scan(&total); err != nil {
		return GrantList{}, fmt.Errorf("count authorization grants: %w", err)
	}
	items := make([]Grant, 0)
	if total == 0 || filter.Offset >= total {
		return GrantList{Items: items, Total: total}, nil
	}

	args = append(args, filter.Limit, filter.Offset)
	limitPosition := len(args) - 1
	offsetPosition := len(args)

	rows, err := r.db.Query(ctx, `
		SELECT `+grantColumns+`,
		       u.username,
		       COALESCE(NULLIF(u.username, ''), 'user-' || u.id::text)
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

	// Keep the response as a non-nil empty slice without deriving an allocation
	// size from request data. The SQL query still uses the repository-owned cap.
	for rows.Next() {
		grant, scanErr := scanGrantWithSubjectOnly(rows)
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

func normalizeRepositoryGrantListLimit(limit int) int {
	if limit <= 0 {
		return repositoryDefaultGrantListLimit
	}
	if limit > repositoryMaxGrantListLimit {
		return repositoryMaxGrantListLimit
	}
	return limit
}

func (r *Repository) ResolveAccessSnapshotByUserID(ctx context.Context, userID int64) (AccessSnapshot, error) {
	ctx = db.WithTableHint(ctx, authorizationGrantsTable)
	if userID <= 0 {
		return AccessSnapshot{}, ErrActorUserNotFound
	}
	snapshot := AccessSnapshot{InternalUserID: userID}
	var (
		userExists          bool
		verifiedStudent     bool
		freshmanProvisional bool
	)
	err := r.db.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM users u WHERE u.id = $1),
			EXISTS (
				SELECT 1
				FROM current_student_qualifying_credentials credential
				WHERE credential.user_id = $1
				  AND credential.credential_class = 'formal_student'
			),
			EXISTS (
				SELECT 1
				FROM current_student_qualifying_credentials credential
				WHERE credential.user_id = $1
				  AND credential.credential_class = 'temporary_freshman'
			)
	`, userID).Scan(&userExists, &verifiedStudent, &freshmanProvisional)
	if err != nil {
		return AccessSnapshot{}, fmt.Errorf("resolve authorization subject facts: %w", err)
	}
	if !userExists {
		return AccessSnapshot{}, ErrActorUserNotFound
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
		  AND activated_at IS NOT NULL
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
			subject_user_id, role, source, school_id, section_id,
			desired_state, projection_status, revision, reason,
			created_by_user_id, updated_by_user_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			'granted', 'pending', 1, $6,
			NULLIF($7, 0), NULLIF($7, 0),
			NOW(), NOW()
		)
		ON CONFLICT DO NOTHING
		RETURNING `+strings.ReplaceAll(grantColumns, "g.", "")+`
	`, input.SubjectUserID, input.Role, input.Source, input.SchoolID, input.SectionID, input.Reason, input.ActorUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Grant{}, errConcurrentGrantInsert
		}
		return Grant{}, fmt.Errorf("insert authorization grant: %w", err)
	}
	return grant, nil
}

func restoreGrantTx(ctx context.Context, tx pgx.Tx, before Grant, input CreateGrantInput) (Grant, error) {
	grant, err := scanGrant(tx.QueryRow(ctx, `
		UPDATE authorization_grants
		SET desired_state = 'granted',
		    source = $4,
		    projection_status = 'pending',
		    revision = revision + 1,
		    reason = $2,
		    updated_by_user_id = NULLIF($3, 0),
		    activated_at = NULL,
		    revoked_at = NULL,
		    projected_at = NULL,
		    last_error = NULL,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING `+strings.ReplaceAll(grantColumns, "g.", "")+`
	`, before.ID, input.Reason, input.ActorUserID, input.Source))
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
		    updated_by_user_id = NULLIF($3, 0),
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
		    updated_by_user_id = NULLIF($3, 0),
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
	actor grantAuditActor,
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
		ActorType:     actor.actorType,
		UserID:        actor.userID,
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
		"source":            grant.Source,
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
		&grant.Source,
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

func scanGrantWithSubjectOnly(row pgx.Row) (Grant, error) {
	var grant Grant
	err := row.Scan(
		&grant.ID,
		&grant.SubjectUserID,
		&grant.Role,
		&grant.Source,
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
