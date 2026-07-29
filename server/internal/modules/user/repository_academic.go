package user

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/phoneutil"
)

var sqlIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var requiredAcademicColumns = []string{
	"xh",
	"xm",
	"sfzjlxdm",
	"sfzjh_enc",
	"sfzjh_hash",
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
	ctx = withDBTable(ctx, "academic_students")
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
		SELECT xh, xm, sfzjlxdm, sfzjh_enc, sfzjh_hash, yxdm, zydm, bjdm,
		       xznj, rxnj, pyccdm, xslbdm, sjh, dzxx,
		       xjztdm, sfzx, sfzj, synced_at
		FROM %s
		WHERE xh = $1
	`, quotedTable)
	err = r.db.QueryRow(ctx, query, xh).Scan(
		&item.XH, &item.XM, &item.SFZJLXDM, &item.SFZJHEnc, &item.SFZJHHash, &item.YXDM, &item.ZYDM, &item.BJDM,
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
	ctx = withDBTable(ctx, "academic_students")
	normalizedTable, err := normalizeAcademicDBTableName(&tableName)
	if err != nil {
		return nil, fmt.Errorf("FindAcademicStudentsByPersonUIDFromTable: %w", err)
	}
	quotedTable, err := quoteAcademicDBTableName(normalizedTable)
	if err != nil {
		return nil, fmt.Errorf("FindAcademicStudentsByPersonUIDFromTable: %w", err)
	}
	docHash, err := r.hashDocumentLookup(sfzjh)
	if err != nil {
		return nil, fmt.Errorf("FindAcademicStudentsByPersonUIDFromTable: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT xh, xm, sfzjlxdm, sfzjh_enc, sfzjh_hash, yxdm, zydm, bjdm,
		       xznj, rxnj, pyccdm, xslbdm, sjh, dzxx,
		       xjztdm, sfzx, sfzj, synced_at
		FROM %s
		WHERE (
		       ($1 <> '' AND sfzjh_hash = $1)
		    OR (sfzjh_hash IS NULL AND sfzjh_enc = convert_to($2, 'UTF8'))
		)
		  AND ($3 = '' OR sfzjlxdm = $3)
	`, quotedTable)
	rows, err := r.db.Query(ctx, query, docHash, sfzjh, strings.TrimSpace(sfzjlxdm))
	if err != nil {
		return nil, fmt.Errorf("FindAcademicStudentsByPersonUID: %w", err)
	}
	defer rows.Close()

	list := make([]AcademicStudent, 0, 4)
	for rows.Next() {
		var item AcademicStudent
		if err := rows.Scan(
			&item.XH, &item.XM, &item.SFZJLXDM, &item.SFZJHEnc, &item.SFZJHHash, &item.YXDM, &item.ZYDM, &item.BJDM,
			&item.XZNJ, &item.RXNJ, &item.PYCCDM, &item.XSLBDM, &item.SJH, &item.DZXX,
			&item.XJZTDM, &item.SFZX, &item.SFZJ, &item.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("FindAcademicStudentsByPersonUID scan: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (r *Repository) hashDocumentLookup(docNumber string) (string, error) {
	trimmed := strings.TrimSpace(docNumber)
	if trimmed == "" || len(r.hmacKey) == 0 {
		return "", nil
	}
	hash, err := phoneutil.HashLookupWithKey(trimmed, r.hmacKey)
	if err != nil {
		return "", fmt.Errorf("hash document lookup: %w", err)
	}
	return hash, nil
}

// GetInternalUserID 根据外部ID获取内部用户ID
func (r *Repository) GetInternalUserID(ctx context.Context, casdoorSubject string) (int64, error) {
	ctx = withDBTable(ctx, "users")
	var id int64
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE casdoor_subject = $1`, casdoorSubject).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("GetInternalUserID: %w", err)
	}
	return id, nil
}

// GetCasdoorSubject resolves users.id back to the current Casdoor subject for role sync.
func (r *Repository) GetCasdoorSubject(ctx context.Context, userID int64) (string, error) {
	ctx = withDBTable(ctx, "users")
	var casdoorSubject string
	err := r.db.QueryRow(ctx, `SELECT casdoor_subject FROM users WHERE id = $1`, userID).Scan(&casdoorSubject)
	if err != nil {
		return "", fmt.Errorf("GetCasdoorSubject: %w", err)
	}
	return casdoorSubject, nil
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
	ctx = withDBTable(ctx, "information_schema.columns")
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
