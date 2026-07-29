package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/reviewaccess"
)

const selectSchoolConfigColumns = `
	sc.school_id, COALESCE(s.code, sc.school_id::text) AS school_code, sc.school_name,
	sc.verification_method, sc.approval_policy, sc.ldap_config,
	sc.academic_db_table, sc.consent_text, sc.manual_form_fields,
	sc.enabled, sc.created_at, sc.updated_at
`

const selectSchoolDirectoryConfigColumns = `
	s.id, s.code AS school_code, COALESCE(sc.school_name, s.name) AS school_name,
	COALESCE(sc.verification_method, 'manual') AS verification_method,
	COALESCE(sc.approval_policy, 'manual') AS approval_policy,
	sc.ldap_config, sc.academic_db_table, sc.consent_text, sc.manual_form_fields,
	COALESCE(sc.enabled, false) AS enabled,
	COALESCE(sc.created_at, s.created_at) AS created_at,
	COALESCE(sc.updated_at, s.created_at) AS updated_at
`

func scanSchoolConfig(row interface{ Scan(dest ...any) error }) (*SchoolConfig, error) {
	var item SchoolConfig
	err := row.Scan(
		&item.SchoolID, &item.SchoolCode, &item.SchoolName, &item.VerificationMethod, &item.ApprovalPolicy, &item.LDAPConfig,
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
	ctx = withDBTable(ctx, "school_configs")
	item, err := scanSchoolConfig(r.db.QueryRow(ctx, `
		SELECT `+selectSchoolDirectoryConfigColumns+`
		FROM schools s
		LEFT JOIN school_configs sc ON sc.school_id = s.id
		WHERE s.id = $1
	`, schoolID))
	if err != nil {
		return nil, fmt.Errorf("GetSchoolConfig: %w", err)
	}
	return item, nil
}

// ListSchoolConfigs 获取所有启用的学校认证配置
func (r *Repository) ListSchoolConfigs(ctx context.Context) ([]SchoolConfig, error) {
	ctx = withDBTable(ctx, "school_configs")
	rows, err := r.db.Query(ctx, `
		SELECT `+selectSchoolConfigColumns+`
		FROM school_configs sc
		LEFT JOIN schools s ON s.id = sc.school_id
		WHERE sc.enabled = true
		ORDER BY sc.school_name ASC
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
	ctx = withDBTable(ctx, "school_configs")
	rows, err := r.db.Query(ctx, `
		SELECT `+selectSchoolDirectoryConfigColumns+`
		FROM schools s
		LEFT JOIN school_configs sc ON sc.school_id = s.id
		ORDER BY s.name ASC, s.code ASC
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
	ctx = withDBTable(ctx, "school_configs")
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE schools
			SET name = $2
			WHERE id = $1
		`, config.SchoolID, config.SchoolName)
		if err != nil {
			return fmt.Errorf("UpdateSchoolConfig update school directory: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO school_configs (
				school_id, school_name, verification_method, approval_policy, ldap_config,
				academic_db_table, consent_text, manual_form_fields, enabled, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
			ON CONFLICT (school_id) DO UPDATE
			SET school_name = EXCLUDED.school_name,
				verification_method = EXCLUDED.verification_method,
				approval_policy = EXCLUDED.approval_policy,
				ldap_config = EXCLUDED.ldap_config,
				academic_db_table = EXCLUDED.academic_db_table,
				consent_text = EXCLUDED.consent_text,
				manual_form_fields = EXCLUDED.manual_form_fields,
				enabled = EXCLUDED.enabled,
				updated_at = NOW()
		`, config.SchoolID, config.SchoolName, config.VerificationMethod, config.ApprovalPolicy, config.LDAPConfig,
			config.AcademicDBTable, config.ConsentText, config.ManualFormFields,
			config.Enabled,
		)
		if err != nil {
			return fmt.Errorf("UpdateSchoolConfig: %w", err)
		}
		return nil
	})
}

// ListSystemConfigs 获取所有系统配置项
func (r *Repository) ListSystemConfigs(ctx context.Context) ([]SystemConfig, error) {
	ctx = withDBTable(ctx, "system_configs")
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
	ctx = withDBTable(ctx, "system_configs")
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
