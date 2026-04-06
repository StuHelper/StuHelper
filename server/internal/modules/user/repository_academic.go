package user

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

const defaultAcademicDBTable = "academic.buaa_students" // kept only for migration reference

var sqlIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var requiredAcademicColumns = []string{
	"xh",
	"xm",
	"sfzjlxdm",
	"sfzjh",
	"yxdm",
	"zydm",
	"bjdm",
	"xznj",
	"rxnj",
	"pyccdm",
	"xslbdm",
	"sjh",
	"dzxx",
	"xjztdm",
	"sfzx",
	"sfzj",
	"synced_at",
}

// GetAcademicStudentByXHFromTable 根据学号查询教务系统学生记录（指定学籍表）
func (r *Repository) GetAcademicStudentByXHFromTable(ctx context.Context, xh string, tableName string) (*AcademicStudent, error) {
	normalizedTable, err := normalizeAcademicDBTableName(&tableName)
	if err != nil {
		return nil, fmt.Errorf("GetAcademicStudentByXHFromTable: %w", err)
	}
	quotedTable, err := quoteAcademicDBTableName(normalizedTable)
	if err != nil {
		return nil, fmt.Errorf("GetAcademicStudentByXHFromTable: %w", err)
	}

	var item AcademicStudent
	query := fmt.Sprintf(`
		SELECT xh, xm, sfzjlxdm, sfzjh, yxdm, zydm, bjdm,
		       xznj, rxnj, pyccdm, xslbdm, sjh, dzxx,
		       xjztdm, sfzx, sfzj, synced_at
		FROM %s
		WHERE xh = $1
	`, quotedTable)
	err = r.db.QueryRow(ctx, query, xh).Scan(
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

// FindAcademicStudentsByPersonUIDFromTable 根据证件号查询教务系统学生记录（指定学籍表）
func (r *Repository) FindAcademicStudentsByPersonUIDFromTable(ctx context.Context, sfzjlxdm, sfzjh string, tableName string) ([]AcademicStudent, error) {
	normalizedTable, err := normalizeAcademicDBTableName(&tableName)
	if err != nil {
		return nil, fmt.Errorf("FindAcademicStudentsByPersonUIDFromTable: %w", err)
	}
	quotedTable, err := quoteAcademicDBTableName(normalizedTable)
	if err != nil {
		return nil, fmt.Errorf("FindAcademicStudentsByPersonUIDFromTable: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT xh, xm, sfzjlxdm, sfzjh, yxdm, zydm, bjdm,
		       xznj, rxnj, pyccdm, xslbdm, sjh, dzxx,
		       xjztdm, sfzx, sfzj, synced_at
		FROM %s
		WHERE sfzjh = $1
		  AND ($2 = '' OR sfzjlxdm = $2)
	`, quotedTable)
	rows, err := r.db.Query(ctx, query, sfzjh, strings.TrimSpace(sfzjlxdm))
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

// GetInternalUserID 根据外部ID获取内部用户ID
func (r *Repository) GetInternalUserID(ctx context.Context, externalID string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE external_id = $1`, externalID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("GetInternalUserID: %w", err)
	}
	return id, nil
}

// GetExternalID 根据内部用户 ID 获取 Zitadel 外部 ID（OIDC sub）
func (r *Repository) GetExternalID(ctx context.Context, userID int64) (string, error) {
	var externalID string
	err := r.db.QueryRow(ctx, `SELECT external_id FROM users WHERE id = $1`, userID).Scan(&externalID)
	if err != nil {
		return "", fmt.Errorf("GetExternalID: %w", err)
	}
	return externalID, nil
}

func normalizeAcademicDBTableName(raw *string) (string, error) {
	if raw == nil {
		return "", fmt.Errorf("academic table must be provided")
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return "", fmt.Errorf("academic table must be provided")
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("academic table must be in schema.table format")
	}

	schemaName := strings.TrimSpace(parts[0])
	tableName := strings.TrimSpace(parts[1])
	if !sqlIdentifierPattern.MatchString(schemaName) || !sqlIdentifierPattern.MatchString(tableName) {
		return "", fmt.Errorf("academic table contains invalid identifier characters")
	}

	return schemaName + "." + tableName, nil
}

func normalizeConfiguredAcademicDBTable(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}

	normalized, err := normalizeAcademicDBTableName(&trimmed)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func quoteAcademicDBTableName(normalized string) (string, error) {
	parts := strings.Split(normalized, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("academic table must be in schema.table format")
	}
	return pgx.Identifier{parts[0], parts[1]}.Sanitize(), nil
}

func (r *Repository) ValidateAcademicDBTable(ctx context.Context, tableName string) error {
	normalizedTable, err := normalizeAcademicDBTableName(&tableName)
	if err != nil {
		return err
	}

	parts := strings.Split(normalizedTable, ".")
	if len(parts) != 2 {
		return fmt.Errorf("academic table must be in schema.table format")
	}

	rows, err := r.db.Query(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
	`, parts[0], parts[1])
	if err != nil {
		return fmt.Errorf("ValidateAcademicDBTable query columns: %w", err)
	}
	defer rows.Close()

	columns := make(map[string]struct{}, len(requiredAcademicColumns))
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return fmt.Errorf("ValidateAcademicDBTable scan columns: %w", err)
		}
		columns[columnName] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("ValidateAcademicDBTable rows: %w", err)
	}
	if len(columns) == 0 {
		return fmt.Errorf("academic table does not exist")
	}

	missing := make([]string, 0, len(requiredAcademicColumns))
	for _, column := range requiredAcademicColumns {
		if _, ok := columns[column]; ok {
			continue
		}
		missing = append(missing, column)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("academic table missing required columns: %s", strings.Join(missing, ", "))
	}

	return nil
}
