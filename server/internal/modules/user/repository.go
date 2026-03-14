package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

// Repository 用户数据访问层
type Repository struct {
	db *db.DB
}

// NewRepository 创建数据访问层
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// ---------- Identity ----------

// GetIdentityByUserID 根据用户ID获取实名认证记录
func (r *Repository) GetIdentityByUserID(ctx context.Context, userID int64) (*Identity, error) {
	var item Identity
	err := r.db.QueryRow(ctx, `
		SELECT user_id, doc_type, doc_number_enc, person_uid, real_name,
		       verified, verify_method, verified_at,
		       doc_photo_front, doc_photo_back, doc_photo_selfie,
		       rejection_reason, created_at, updated_at
		FROM user_identities
		WHERE user_id = $1
	`, userID).Scan(
		&item.UserID, &item.DocType, &item.DocNumberEnc, &item.PersonUID, &item.RealName,
		&item.Verified, &item.VerifyMethod, &item.VerifiedAt,
		&item.DocPhotoFront, &item.DocPhotoBack, &item.DocPhotoSelfie,
		&item.RejectionReason, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetIdentityByUserID: %w", err)
	}
	return &item, nil
}

// CreateIdentity 创建实名认证记录
func (r *Repository) CreateIdentity(ctx context.Context, identity *Identity) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_identities (
			user_id, doc_type, doc_number_enc, person_uid, real_name,
			verified, verify_method, verified_at,
			doc_photo_front, doc_photo_back, doc_photo_selfie,
			rejection_reason, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
	`, identity.UserID, identity.DocType, identity.DocNumberEnc, identity.PersonUID, identity.RealName,
		identity.Verified, identity.VerifyMethod, identity.VerifiedAt,
		identity.DocPhotoFront, identity.DocPhotoBack, identity.DocPhotoSelfie,
		identity.RejectionReason,
	)
	if err != nil {
		return fmt.Errorf("CreateIdentity: %w", err)
	}
	return nil
}

// UpdateIdentity 更新实名认证记录
func (r *Repository) UpdateIdentity(ctx context.Context, identity *Identity) error {
	_, err := r.db.Exec(ctx, `
		UPDATE user_identities SET
			doc_type = $2, doc_number_enc = $3, person_uid = $4, real_name = $5,
			verified = $6, verify_method = $7, verified_at = $8,
			doc_photo_front = $9, doc_photo_back = $10, doc_photo_selfie = $11,
			rejection_reason = $12, updated_at = NOW()
		WHERE user_id = $1
	`, identity.UserID, identity.DocType, identity.DocNumberEnc, identity.PersonUID, identity.RealName,
		identity.Verified, identity.VerifyMethod, identity.VerifiedAt,
		identity.DocPhotoFront, identity.DocPhotoBack, identity.DocPhotoSelfie,
		identity.RejectionReason,
	)
	if err != nil {
		return fmt.Errorf("UpdateIdentity: %w", err)
	}
	return nil
}

// ---------- Profile ----------

// GetProfileByUserID 根据用户ID获取学生认证档案
func (r *Repository) GetProfileByUserID(ctx context.Context, userID int64) (*Profile, error) {
	var item Profile
	var studentIDsJSON []byte
	err := r.db.QueryRow(ctx, `
		SELECT user_id, school_id, student_ids, active_student_id,
		       verification_status, verification_method,
		       phone, phone_verified, consent_given_at, verified_at,
		       created_at, updated_at
		FROM user_profiles
		WHERE user_id = $1
	`, userID).Scan(
		&item.UserID, &item.SchoolID, &studentIDsJSON, &item.ActiveStudentID,
		&item.VerificationStatus, &item.VerificationMethod,
		&item.Phone, &item.PhoneVerified, &item.ConsentGivenAt, &item.VerifiedAt,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetProfileByUserID: %w", err)
	}
	if studentIDsJSON != nil {
		if err := json.Unmarshal(studentIDsJSON, &item.StudentIDs); err != nil {
			return nil, fmt.Errorf("GetProfileByUserID unmarshal student_ids: %w", err)
		}
	}
	return &item, nil
}

// CreateProfile 创建学生认证档案
func (r *Repository) CreateProfile(ctx context.Context, profile *Profile) error {
	studentIDsJSON, err := json.Marshal(profile.StudentIDs)
	if err != nil {
		return fmt.Errorf("CreateProfile marshal student_ids: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO user_profiles (
			user_id, school_id, student_ids, active_student_id,
			verification_status, verification_method,
			phone, phone_verified, consent_given_at, verified_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
	`, profile.UserID, profile.SchoolID, studentIDsJSON, profile.ActiveStudentID,
		profile.VerificationStatus, profile.VerificationMethod,
		profile.Phone, profile.PhoneVerified, profile.ConsentGivenAt, profile.VerifiedAt,
	)
	if err != nil {
		return fmt.Errorf("CreateProfile: %w", err)
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
			school_id = $2, student_ids = $3, active_student_id = $4,
			verification_status = $5, verification_method = $6,
			phone = $7, phone_verified = $8, consent_given_at = $9, verified_at = $10,
			updated_at = NOW()
		WHERE user_id = $1
	`, profile.UserID, profile.SchoolID, studentIDsJSON, profile.ActiveStudentID,
		profile.VerificationStatus, profile.VerificationMethod,
		profile.Phone, profile.PhoneVerified, profile.ConsentGivenAt, profile.VerifiedAt,
	)
	if err != nil {
		return fmt.Errorf("UpdateProfile: %w", err)
	}
	return nil
}

// ---------- Identity / Profile 列表查询 ----------

// ListIdentitiesByStatus 分页查询实名认证记录（按状态筛选）
func (r *Repository) ListIdentitiesByStatus(ctx context.Context, status string, page, pageSize int) ([]Identity, int, error) {
	var qb strings.Builder
	qb.WriteString(`
		SELECT user_id, doc_type, doc_number_enc, person_uid, real_name,
		       verified, verify_method, verified_at,
		       doc_photo_front, doc_photo_back, doc_photo_selfie,
		       rejection_reason, created_at, updated_at,
		       COUNT(*) OVER() AS total
		FROM user_identities
		WHERE 1=1
	`)
	args := make([]any, 0, 4)
	argIdx := 1

	if status != "" {
		if status == "verified" {
			qb.WriteString(` AND verified = true`)
		} else if status == "unverified" {
			qb.WriteString(` AND verified = false`)
		}
	}

	qb.WriteString(` ORDER BY created_at DESC`)
	qb.WriteString(` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1))
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := r.db.Query(ctx, qb.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListIdentitiesByStatus: %w", err)
	}
	defer rows.Close()

	list := make([]Identity, 0, pageSize)
	var total int
	for rows.Next() {
		var item Identity
		if err := rows.Scan(
			&item.UserID, &item.DocType, &item.DocNumberEnc, &item.PersonUID, &item.RealName,
			&item.Verified, &item.VerifyMethod, &item.VerifiedAt,
			&item.DocPhotoFront, &item.DocPhotoBack, &item.DocPhotoSelfie,
			&item.RejectionReason, &item.CreatedAt, &item.UpdatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("ListIdentitiesByStatus scan: %w", err)
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListIdentitiesByStatus rows: %w", err)
	}
	return list, total, nil
}

// ListProfilesByStatus 分页查询学生认证档案（按状态和学校筛选）
func (r *Repository) ListProfilesByStatus(ctx context.Context, status string, schoolID string, page, pageSize int) ([]Profile, int, error) {
	var qb strings.Builder
	qb.WriteString(`
		SELECT user_id, school_id, student_ids, active_student_id,
		       verification_status, verification_method,
		       phone, phone_verified, consent_given_at, verified_at,
		       created_at, updated_at,
		       COUNT(*) OVER() AS total
		FROM user_profiles
		WHERE 1=1
	`)
	args := make([]any, 0, 4)
	argIdx := 1

	if status != "" {
		qb.WriteString(` AND verification_status = $` + strconv.Itoa(argIdx))
		args = append(args, status)
		argIdx++
	}
	if schoolID != "" {
		qb.WriteString(` AND school_id = $` + strconv.Itoa(argIdx))
		args = append(args, schoolID)
		argIdx++
	}

	qb.WriteString(` ORDER BY created_at DESC`)
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
		if err := rows.Scan(
			&item.UserID, &item.SchoolID, &studentIDsJSON, &item.ActiveStudentID,
			&item.VerificationStatus, &item.VerificationMethod,
			&item.Phone, &item.PhoneVerified, &item.ConsentGivenAt, &item.VerifiedAt,
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
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListProfilesByStatus rows: %w", err)
	}
	return list, total, nil
}

// ---------- SchoolConfig ----------

// GetSchoolConfig 获取学校认证配置
func (r *Repository) GetSchoolConfig(ctx context.Context, schoolID string) (*SchoolConfig, error) {
	var item SchoolConfig
	err := r.db.QueryRow(ctx, `
		SELECT school_id, school_name, verification_method, ldap_config,
		       academic_db_table, consent_text, manual_form_fields,
		       enabled, created_at, updated_at
		FROM school_configs
		WHERE school_id = $1
	`, schoolID).Scan(
		&item.SchoolID, &item.SchoolName, &item.VerificationMethod, &item.LDAPConfig,
		&item.AcademicDBTable, &item.ConsentText, &item.ManualFormFields,
		&item.Enabled, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetSchoolConfig: %w", err)
	}
	return &item, nil
}

// ListSchoolConfigs 获取所有启用的学校认证配置
func (r *Repository) ListSchoolConfigs(ctx context.Context) ([]SchoolConfig, error) {
	rows, err := r.db.Query(ctx, `
		SELECT school_id, school_name, verification_method, ldap_config,
		       academic_db_table, consent_text, manual_form_fields,
		       enabled, created_at, updated_at
		FROM school_configs
		WHERE enabled = true
		ORDER BY school_name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListSchoolConfigs: %w", err)
	}
	defer rows.Close()

	list := make([]SchoolConfig, 0, 10)
	for rows.Next() {
		var item SchoolConfig
		if err := rows.Scan(
			&item.SchoolID, &item.SchoolName, &item.VerificationMethod, &item.LDAPConfig,
			&item.AcademicDBTable, &item.ConsentText, &item.ManualFormFields,
			&item.Enabled, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ListSchoolConfigs scan: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// UpdateSchoolConfig 更新学校认证配置
func (r *Repository) UpdateSchoolConfig(ctx context.Context, config *SchoolConfig) error {
	_, err := r.db.Exec(ctx, `
		UPDATE school_configs SET
			school_name = $2, verification_method = $3, ldap_config = $4,
			academic_db_table = $5, consent_text = $6, manual_form_fields = $7,
			enabled = $8, updated_at = NOW()
		WHERE school_id = $1
	`, config.SchoolID, config.SchoolName, config.VerificationMethod, config.LDAPConfig,
		config.AcademicDBTable, config.ConsentText, config.ManualFormFields,
		config.Enabled,
	)
	if err != nil {
		return fmt.Errorf("UpdateSchoolConfig: %w", err)
	}
	return nil
}

// ---------- AcademicStudent ----------

// GetAcademicStudentByXH 根据学号查询教务系统学生记录
func (r *Repository) GetAcademicStudentByXH(ctx context.Context, xh string) (*AcademicStudent, error) {
	var item AcademicStudent
	err := r.db.QueryRow(ctx, `
		SELECT xh, xm, sfzjlxdm, sfzjh, yxdm, zydm, bjdm,
		       xznj, rxnj, pyccdm, xslbdm, sjh, dzxx,
		       xjztdm, sfzx, sfzj, synced_at
		FROM academic.buaa_students
		WHERE xh = $1
	`, xh).Scan(
		&item.XH, &item.XM, &item.SFZJLXDM, &item.SFZJH, &item.YXDM, &item.ZYDM, &item.BJDM,
		&item.XZNJ, &item.RXNJ, &item.PYCCDM, &item.XSLBDM, &item.SJH, &item.DZXX,
		&item.XJZTDM, &item.SFZX, &item.SFZJ, &item.SyncedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetAcademicStudentByXH: %w", err)
	}
	return &item, nil
}

// FindAcademicStudentsByPersonUID 根据证件号查询教务系统学生记录
func (r *Repository) FindAcademicStudentsByPersonUID(ctx context.Context, sfzjlxdm, sfzjh string) ([]AcademicStudent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT xh, xm, sfzjlxdm, sfzjh, yxdm, zydm, bjdm,
		       xznj, rxnj, pyccdm, xslbdm, sjh, dzxx,
		       xjztdm, sfzx, sfzj, synced_at
		FROM academic.buaa_students
		WHERE sfzjh = $1
	`, sfzjh)
	if err != nil {
		return nil, fmt.Errorf("FindAcademicStudentsByPersonUID: %w", err)
	}
	defer rows.Close()

	list := make([]AcademicStudent, 0, 4)
	for rows.Next() {
		var item AcademicStudent
		if err := rows.Scan(
			&item.XH, &item.XM, &item.SFZJLXDM, &item.SFZJH, &item.YXDM, &item.ZYDM, &item.BJDM,
			&item.XZNJ, &item.RXNJ, &item.PYCCDM, &item.XSLBDM, &item.SJH, &item.DZXX,
			&item.XJZTDM, &item.SFZX, &item.SFZJ, &item.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("FindAcademicStudentsByPersonUID scan: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ---------- SystemConfig ----------

// GetSystemConfig 获取系统配置项
func (r *Repository) GetSystemConfig(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRow(ctx, `
		SELECT value FROM system_configs WHERE key = $1
	`, key).Scan(&value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("GetSystemConfig: key %q not found", key)
		}
		return "", fmt.Errorf("GetSystemConfig: %w", err)
	}
	return value, nil
}

// ListSystemConfigs 获取所有系统配置项
func (r *Repository) ListSystemConfigs(ctx context.Context) ([]SystemConfig, error) {
	rows, err := r.db.Query(ctx, `
		SELECT key, value, description, updated_at
		FROM system_configs
		ORDER BY key ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListSystemConfigs: %w", err)
	}
	defer rows.Close()

	list := make([]SystemConfig, 0, 20)
	for rows.Next() {
		var item SystemConfig
		if err := rows.Scan(&item.Key, &item.Value, &item.Description, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ListSystemConfigs scan: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// UpdateSystemConfig 更新系统配置项
func (r *Repository) UpdateSystemConfig(ctx context.Context, key, value string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE system_configs SET value = $2, updated_at = NOW() WHERE key = $1
	`, key, value)
	if err != nil {
		return fmt.Errorf("UpdateSystemConfig: %w", err)
	}
	return nil
}

// ---------- Users ----------

// GetInternalUserID 根据外部ID获取内部用户ID
func (r *Repository) GetInternalUserID(ctx context.Context, externalID string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE external_id = $1`, externalID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("GetInternalUserID: %w", err)
	}
	return id, nil
}

// ---------- Admin SchoolConfig ----------

// ListAllSchoolConfigs 获取所有学校认证配置（含禁用，管理端用）
func (r *Repository) ListAllSchoolConfigs(ctx context.Context) ([]SchoolConfig, error) {
	rows, err := r.db.Query(ctx, `
		SELECT school_id, school_name, verification_method, ldap_config,
		       academic_db_table, consent_text, manual_form_fields,
		       enabled, created_at, updated_at
		FROM school_configs
		ORDER BY school_name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListAllSchoolConfigs: %w", err)
	}
	defer rows.Close()

	list := make([]SchoolConfig, 0, 10)
	for rows.Next() {
		var item SchoolConfig
		if err := rows.Scan(
			&item.SchoolID, &item.SchoolName, &item.VerificationMethod, &item.LDAPConfig,
			&item.AcademicDBTable, &item.ConsentText, &item.ManualFormFields,
			&item.Enabled, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ListAllSchoolConfigs scan: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}
