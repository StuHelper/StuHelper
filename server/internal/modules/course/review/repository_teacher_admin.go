package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

// ListAdminTeachers 获取教师列表（管理员，含院系名和评论数）
func (r *Repository) ListAdminTeachers(ctx context.Context, search string, departmentID int64, limit, offset int) ([]AdminTeacher, int, error) {
	var qb strings.Builder
	qb.WriteString(`
		SELECT t.id, t.name, t.department_id, d.name,
		       (SELECT COUNT(*) FROM reviews rv WHERE rv.teacher_id = t.id AND rv.status != 'deleted') AS review_count,
		       t.created_at,
		       COUNT(*) OVER() AS total
		FROM teachers t
		LEFT JOIN departments d ON d.id = t.department_id
		WHERE 1=1
	`)

	args := make([]interface{}, 0, 4)
	argIdx := 1

	if search != "" {
		qb.WriteString(fmt.Sprintf(` AND t.name ILIKE $%d`, argIdx))
		args = append(args, "%"+httputil.EscapeLikePattern(search)+"%")
		argIdx++
	}
	if departmentID > 0 {
		qb.WriteString(fmt.Sprintf(` AND t.department_id = $%d`, argIdx))
		args = append(args, departmentID)
		argIdx++
	}

	qb.WriteString(fmt.Sprintf(` ORDER BY t.id DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1))
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, qb.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListAdminTeachers: %w", err)
	}
	defer rows.Close()

	list := make([]AdminTeacher, 0, limit)
	var total int
	for rows.Next() {
		var t AdminTeacher
		if err := rows.Scan(&t.ID, &t.Name, &t.DepartmentID, &t.DepartmentName, &t.ReviewCount, &t.CreatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("ListAdminTeachers scan: %w", err)
		}
		list = append(list, t)
	}
	return list, total, rows.Err()
}

// CreateTeacher 创建教师
func (r *Repository) CreateTeacher(ctx context.Context, name string, departmentID *int64) (*AdminTeacher, error) {
	var t AdminTeacher
	err := r.db.QueryRow(ctx, `
		INSERT INTO teachers (name, department_id) VALUES ($1, $2)
		RETURNING id, name, department_id, created_at
	`, name, departmentID).Scan(&t.ID, &t.Name, &t.DepartmentID, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("CreateTeacher: %w", err)
	}

	// 查询院系名
	if t.DepartmentID != nil {
		_ = r.db.QueryRow(ctx, `SELECT name FROM departments WHERE id = $1`, *t.DepartmentID).Scan(&t.DepartmentName)
	}
	return &t, nil
}

// UpdateTeacher 更新教师
func (r *Repository) UpdateTeacher(ctx context.Context, id int64, name string, departmentID *int64) error {
	result, err := r.db.Exec(ctx, `
		UPDATE teachers SET name = $2, department_id = $3 WHERE id = $1
	`, id, name, departmentID)
	if err != nil {
		return fmt.Errorf("UpdateTeacher: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteTeacher 删除教师（检查是否有关联评论）
func (r *Repository) DeleteTeacher(ctx context.Context, id int64) error {
	var reviewCount int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE teacher_id = $1 AND status != 'deleted'`, id).Scan(&reviewCount); err != nil {
		return fmt.Errorf("DeleteTeacher check: %w", err)
	}
	if reviewCount > 0 {
		return fmt.Errorf("teacher has %d associated reviews, cannot delete", reviewCount)
	}

	result, err := r.db.Exec(ctx, `DELETE FROM teachers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("DeleteTeacher: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
