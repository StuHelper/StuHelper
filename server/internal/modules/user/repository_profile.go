package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
)

type rowScanner interface {
	Scan(dest ...any) error
}

type execFn func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)

const insertProfileSQL = `
			INSERT INTO user_profiles (
				user_id, school_id, student_ids, active_student_id, manual_form_data,
				verification_status, verification_method, rejection_reason, reviewed_at,
				consent_given_at, verified_at,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		`

const updateProfileSQL = `
			UPDATE user_profiles SET
				school_id = $2, student_ids = $3, active_student_id = $4, manual_form_data = $5,
				verification_status = $6, verification_method = $7, rejection_reason = $8, reviewed_at = $9,
				consent_given_at = $10, verified_at = $11,
				updated_at = NOW()
			WHERE user_id = $1
		`

func saveProfile(ctx context.Context, exec execFn, profile *Profile, sql, op string) error {
	studentIDsJSON, err := json.Marshal(profile.StudentIDs)
	if err != nil {
		return fmt.Errorf("%s marshal student_ids: %w", op, err)
	}
	_, err = exec(ctx, sql,
		profile.UserID, profile.SchoolID, studentIDsJSON, profile.ActiveStudentID, profile.ManualFormData,
		profile.VerificationStatus, profile.VerificationMethod, profile.RejectionReason, profile.ReviewedAt,
		profile.ConsentGivenAt, profile.VerifiedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

const selectProfileByUserIDSQL = `
			SELECT p.user_id, p.school_id, p.student_ids, p.active_student_id, p.manual_form_data,
			       p.verification_status, p.verification_method, p.rejection_reason, p.reviewed_at,
			       u.phone_enc, p.consent_given_at, p.verified_at,
			       p.created_at, p.updated_at
			FROM user_profiles p
			LEFT JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $1
		`

func scanProfileRow(row pgx.Row) (*Profile, error) {
	var item Profile
	var studentIDsJSON []byte
	var manualFormDataJSON []byte
	err := row.Scan(
		&item.UserID, &item.SchoolID, &studentIDsJSON, &item.ActiveStudentID, &manualFormDataJSON,
		&item.VerificationStatus, &item.VerificationMethod, &item.RejectionReason, &item.ReviewedAt,
		&item.PhoneEnc, &item.ConsentGivenAt, &item.VerifiedAt,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if studentIDsJSON != nil {
		if err := json.Unmarshal(studentIDsJSON, &item.StudentIDs); err != nil {
			return nil, fmt.Errorf("unmarshal student_ids: %w", err)
		}
	}
	item.ManualFormData = manualFormDataJSON
	item.PhoneVerified = len(item.PhoneEnc) > 0
	return &item, nil
}

// GetProfileByUserID 根据用户ID获取学生认证档案
func (r *Repository) GetProfileByUserID(ctx context.Context, userID int64) (*Profile, error) {
	ctx = withDBTable(ctx, "user_profiles")
	item, err := scanProfileRow(r.db.QueryRow(ctx, selectProfileByUserIDSQL, userID))
	if err != nil {
		return nil, fmt.Errorf("GetProfileByUserID: %w", err)
	}
	return item, nil
}

func (r *Repository) GetProfileByUserIDTx(ctx context.Context, tx pgx.Tx, userID int64) (*Profile, error) {
	item, err := scanProfileRow(tx.QueryRow(ctx, selectProfileByUserIDSQL, userID))
	if err != nil {
		return nil, fmt.Errorf("GetProfileByUserIDTx: %w", err)
	}
	return item, nil
}

func (r *Repository) GetProfileByUserIDForUpdateTx(ctx context.Context, tx pgx.Tx, userID int64) (*Profile, error) {
	ctx = withDBTable(ctx, "user_profiles")
	var lockedUserID int64
	if err := tx.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, userID).Scan(&lockedUserID); err != nil {
		return nil, fmt.Errorf("GetProfileByUserIDForUpdateTx lock user: %w", err)
	}
	item, err := scanProfileRow(tx.QueryRow(ctx, selectProfileByUserIDSQL+` FOR UPDATE OF p`, userID))
	if err != nil {
		return nil, fmt.Errorf("GetProfileByUserIDForUpdateTx: %w", err)
	}
	return item, nil
}

// CreateProfile 创建学生认证档案
func (r *Repository) CreateProfile(ctx context.Context, profile *Profile) error {
	ctx = withDBTable(ctx, "user_profiles")
	return saveProfile(ctx, r.db.Exec, profile, insertProfileSQL, "CreateProfile")
}

func (r *Repository) CreateProfileTx(ctx context.Context, tx pgx.Tx, profile *Profile) error {
	return saveProfile(ctx, tx.Exec, profile, insertProfileSQL, "CreateProfileTx")
}

// UpdateProfile 更新学生认证档案
func (r *Repository) UpdateProfile(ctx context.Context, profile *Profile) error {
	ctx = withDBTable(ctx, "user_profiles")
	return saveProfile(ctx, r.db.Exec, profile, updateProfileSQL, "UpdateProfile")
}

func (r *Repository) UpdateProfileTx(ctx context.Context, tx pgx.Tx, profile *Profile) error {
	return saveProfile(ctx, tx.Exec, profile, updateProfileSQL, "UpdateProfileTx")
}

// ListProfilesByStatus 分页查询学生认证档案（按状态和学校筛选）
func (r *Repository) ListProfilesByStatus(ctx context.Context, status string, schoolID *int64, page, pageSize int) ([]Profile, int, error) {
	ctx = withDBTable(ctx, "user_profiles")
	pageSize = httputil.ClampPageSize(pageSize)
	offset := httputil.SafeOffset(page, pageSize)
	args := make([]any, 0, 4)
	argIdx := 1
	whereClause := ""

	if status != "" {
		whereClause += ` WHERE p.verification_status = $` + strconv.Itoa(argIdx)
		args = append(args, status)
		argIdx++
	}
	if schoolID != nil {
		if whereClause == "" {
			whereClause += ` WHERE p.school_id = $` + strconv.Itoa(argIdx)
		} else {
			whereClause += ` AND p.school_id = $` + strconv.Itoa(argIdx)
		}
		args = append(args, *schoolID)
		argIdx++
	}

	countQuery := `
		SELECT COUNT(*)
		FROM user_profiles p
		LEFT JOIN users u ON u.id = p.user_id
	` + whereClause
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("ListProfilesByStatus count: %w", err)
	}
	if total == 0 {
		return []Profile{}, 0, nil
	}

	args = append(args, pageSize, offset)
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.school_id, p.student_ids, p.active_student_id, p.manual_form_data,
		       p.verification_status, p.verification_method, p.rejection_reason, p.reviewed_at,
		       u.phone_enc, p.consent_given_at, p.verified_at,
		       p.created_at, p.updated_at
		FROM user_profiles p
		LEFT JOIN users u ON u.id = p.user_id
	`+whereClause+`
		ORDER BY p.created_at DESC
		LIMIT $`+strconv.Itoa(argIdx)+` OFFSET $`+strconv.Itoa(argIdx+1), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListProfilesByStatus data: %w", err)
	}
	defer rows.Close()

	list := make([]Profile, 0)
	for rows.Next() {
		var item Profile
		var studentIDsJSON []byte
		var manualFormDataJSON []byte
		if err := rows.Scan(
			&item.UserID, &item.SchoolID, &studentIDsJSON, &item.ActiveStudentID, &manualFormDataJSON,
			&item.VerificationStatus, &item.VerificationMethod, &item.RejectionReason, &item.ReviewedAt,
			&item.PhoneEnc, &item.ConsentGivenAt, &item.VerifiedAt,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("ListProfilesByStatus scan: %w", err)
		}
		if studentIDsJSON != nil {
			if err := json.Unmarshal(studentIDsJSON, &item.StudentIDs); err != nil {
				return nil, 0, fmt.Errorf("ListProfilesByStatus unmarshal student_ids: %w", err)
			}
		}
		item.ManualFormData = manualFormDataJSON
		item.PhoneVerified = len(item.PhoneEnc) > 0
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListProfilesByStatus rows: %w", err)
	}
	return list, total, nil
}
