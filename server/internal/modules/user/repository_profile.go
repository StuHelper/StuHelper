package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

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

// CreateProfile 创建学生认证档案
func (r *Repository) CreateProfile(ctx context.Context, profile *Profile) error {
	studentIDsJSON, err := json.Marshal(profile.StudentIDs)
	if err != nil {
		return fmt.Errorf("CreateProfile marshal student_ids: %w", err)
	}
	_, err = r.db.Exec(ctx, `
			INSERT INTO user_profiles (
				user_id, school_id, student_ids, active_student_id, manual_form_data,
				verification_status, verification_method, rejection_reason, reviewed_at,
				consent_given_at, verified_at,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		`, profile.UserID, profile.SchoolID, studentIDsJSON, profile.ActiveStudentID, profile.ManualFormData,
		profile.VerificationStatus, profile.VerificationMethod, profile.RejectionReason, profile.ReviewedAt,
		profile.ConsentGivenAt, profile.VerifiedAt,
	)
	if err != nil {
		return fmt.Errorf("CreateProfile: %w", err)
	}
	return nil
}

func (r *Repository) CreateProfileTx(ctx context.Context, tx pgx.Tx, profile *Profile) error {
	studentIDsJSON, err := json.Marshal(profile.StudentIDs)
	if err != nil {
		return fmt.Errorf("CreateProfileTx marshal student_ids: %w", err)
	}
	_, err = tx.Exec(ctx, `
			INSERT INTO user_profiles (
				user_id, school_id, student_ids, active_student_id, manual_form_data,
				verification_status, verification_method, rejection_reason, reviewed_at,
				consent_given_at, verified_at,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		`, profile.UserID, profile.SchoolID, studentIDsJSON, profile.ActiveStudentID, profile.ManualFormData,
		profile.VerificationStatus, profile.VerificationMethod, profile.RejectionReason, profile.ReviewedAt,
		profile.ConsentGivenAt, profile.VerifiedAt,
	)
	if err != nil {
		return fmt.Errorf("CreateProfileTx: %w", err)
	}
	return nil
}

// UpdateProfile 更新学生认证档案
func (r *Repository) UpdateProfile(ctx context.Context, profile *Profile) error {
	studentIDsJSON, err := json.Marshal(profile.StudentIDs)
	if err != nil {
		return fmt.Errorf("UpdateProfile marshal student_ids: %w", err)
	}
	_, err = r.db.Exec(ctx, `
			UPDATE user_profiles SET
				school_id = $2, student_ids = $3, active_student_id = $4, manual_form_data = $5,
				verification_status = $6, verification_method = $7, rejection_reason = $8, reviewed_at = $9,
				consent_given_at = $10, verified_at = $11,
				updated_at = NOW()
			WHERE user_id = $1
		`, profile.UserID, profile.SchoolID, studentIDsJSON, profile.ActiveStudentID, profile.ManualFormData,
		profile.VerificationStatus, profile.VerificationMethod, profile.RejectionReason, profile.ReviewedAt,
		profile.ConsentGivenAt, profile.VerifiedAt,
	)
	if err != nil {
		return fmt.Errorf("UpdateProfile: %w", err)
	}
	return nil
}

func (r *Repository) UpdateProfileTx(ctx context.Context, tx pgx.Tx, profile *Profile) error {
	studentIDsJSON, err := json.Marshal(profile.StudentIDs)
	if err != nil {
		return fmt.Errorf("UpdateProfileTx marshal student_ids: %w", err)
	}
	_, err = tx.Exec(ctx, `
			UPDATE user_profiles SET
				school_id = $2, student_ids = $3, active_student_id = $4, manual_form_data = $5,
				verification_status = $6, verification_method = $7, rejection_reason = $8, reviewed_at = $9,
				consent_given_at = $10, verified_at = $11,
				updated_at = NOW()
			WHERE user_id = $1
		`, profile.UserID, profile.SchoolID, studentIDsJSON, profile.ActiveStudentID, profile.ManualFormData,
		profile.VerificationStatus, profile.VerificationMethod, profile.RejectionReason, profile.ReviewedAt,
		profile.ConsentGivenAt, profile.VerifiedAt,
	)
	if err != nil {
		return fmt.Errorf("UpdateProfileTx: %w", err)
	}
	return nil
}

// ListProfilesByStatus 分页查询学生认证档案（按状态和学校筛选）
func (r *Repository) ListProfilesByStatus(ctx context.Context, status string, schoolID *int64, page, pageSize int) ([]Profile, int, error) {
	var qb strings.Builder
	qb.WriteString(`
			SELECT p.user_id, p.school_id, p.student_ids, p.active_student_id, p.manual_form_data,
			       p.verification_status, p.verification_method, p.rejection_reason, p.reviewed_at,
			       u.phone_enc, p.consent_given_at, p.verified_at,
			       p.created_at, p.updated_at,
			       COUNT(*) OVER() AS total
		FROM user_profiles p
		LEFT JOIN users u ON u.id = p.user_id
		WHERE 1=1
	`)
	args := make([]any, 0, 4)
	argIdx := 1

	if status != "" {
		qb.WriteString(` AND p.verification_status = $` + strconv.Itoa(argIdx))
		args = append(args, status)
		argIdx++
	}
	if schoolID != nil {
		qb.WriteString(` AND p.school_id = $` + strconv.Itoa(argIdx))
		args = append(args, *schoolID)
		argIdx++
	}

	qb.WriteString(` ORDER BY p.created_at DESC`)
	qb.WriteString(` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1))
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := r.db.Query(ctx, qb.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListProfilesByStatus: %w", err)
	}
	defer rows.Close()

	list := make([]Profile, 0, pageSize)
	var total int
	for rows.Next() {
		var item Profile
		var studentIDsJSON []byte
		var manualFormDataJSON []byte
		if err := rows.Scan(
			&item.UserID, &item.SchoolID, &studentIDsJSON, &item.ActiveStudentID, &manualFormDataJSON,
			&item.VerificationStatus, &item.VerificationMethod, &item.RejectionReason, &item.ReviewedAt,
			&item.PhoneEnc, &item.ConsentGivenAt, &item.VerifiedAt,
			&item.CreatedAt, &item.UpdatedAt,
			&total,
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
