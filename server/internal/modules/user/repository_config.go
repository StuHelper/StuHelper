package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/reviewaccess"
)

const selectSchoolConfigColumns = `
	school_id, school_name, verification_method, approval_policy, ldap_config,
	academic_db_table, consent_text, manual_form_fields,
	enabled, created_at, updated_at
`

func scanSchoolConfig(row interface{ Scan(dest ...any) error }) (*SchoolConfig, error) {
	var item SchoolConfig
	err := row.Scan(
		&item.SchoolID, &item.SchoolName, &item.VerificationMethod, &item.ApprovalPolicy, &item.LDAPConfig,
		&item.AcademicDBTable, &item.ConsentText, &item.ManualFormFields,
		&item.Enabled, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// GetSchoolConfig 获取学校认证配置
func (r *Repository) GetSchoolConfig(ctx context.Context, schoolID int64) (*SchoolConfig, error) {
	item, err := scanSchoolConfig(r.db.QueryRow(ctx, `
		SELECT `+selectSchoolConfigColumns+`
		FROM school_configs
		WHERE school_id = $1
	`, schoolID))
	if err != nil {
		return nil, fmt.Errorf("GetSchoolConfig: %w", err)
	}
	return item, nil
}

// ListSchoolConfigs 获取所有启用的学校认证配置
func (r *Repository) ListSchoolConfigs(ctx context.Context) ([]SchoolConfig, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+selectSchoolConfigColumns+`
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
		item, err := scanSchoolConfig(rows)
		if err != nil {
			return nil, fmt.Errorf("ListSchoolConfigs scan: %w", err)
		}
		if item != nil {
			list = append(list, *item)
		}
	}
	return list, rows.Err()
}

func (r *Repository) ListReviewAccessSchoolConfigs(ctx context.Context) ([]reviewaccess.SchoolConfig, error) {
	schools, err := r.ListSchoolConfigs(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]reviewaccess.SchoolConfig, 0, len(schools))
	for _, school := range schools {
		result = append(result, reviewaccess.SchoolConfig{SchoolID: school.SchoolID})
	}
	return result, nil
}

// ListAllSchoolConfigs 获取所有学校认证配置（含禁用，管理端用）
func (r *Repository) ListAllSchoolConfigs(ctx context.Context) ([]SchoolConfig, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+selectSchoolConfigColumns+`
		FROM school_configs
		ORDER BY school_name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListAllSchoolConfigs: %w", err)
	}
	defer rows.Close()

	list := make([]SchoolConfig, 0, 10)
	for rows.Next() {
		item, err := scanSchoolConfig(rows)
		if err != nil {
			return nil, fmt.Errorf("ListAllSchoolConfigs scan: %w", err)
		}
		if item != nil {
			list = append(list, *item)
		}
	}
	return list, rows.Err()
}

// UpdateSchoolConfig 更新学校认证配置
func (r *Repository) UpdateSchoolConfig(ctx context.Context, config *SchoolConfig) error {
	_, err := r.db.Exec(ctx, `
		UPDATE school_configs SET
			school_name = $2, verification_method = $3, approval_policy = $4, ldap_config = $5,
			academic_db_table = $6, consent_text = $7, manual_form_fields = $8,
			enabled = $9, updated_at = NOW()
		WHERE school_id = $1
	`, config.SchoolID, config.SchoolName, config.VerificationMethod, config.ApprovalPolicy, config.LDAPConfig,
		config.AcademicDBTable, config.ConsentText, config.ManualFormFields,
		config.Enabled,
	)
	if err != nil {
		return fmt.Errorf("UpdateSchoolConfig: %w", err)
	}
	return nil
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

func (r *Repository) ListReviewAccessSystemConfigs(ctx context.Context) ([]reviewaccess.SystemConfig, error) {
	configs, err := r.ListSystemConfigs(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]reviewaccess.SystemConfig, 0, len(configs))
	for _, config := range configs {
		result = append(result, reviewaccess.SystemConfig{
			Key:   config.Key,
			Value: config.Value,
		})
	}
	return result, nil
}

// UpdateSystemConfig 更新系统配置项
func (r *Repository) UpdateSystemConfig(ctx context.Context, key, value string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE system_configs SET value = $2, updated_at = NOW() WHERE key = $1
	`, key, value)
	if err != nil {
		return fmt.Errorf("UpdateSystemConfig: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSystemConfigNotFound
	}
	return nil
}
