package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

// ListAdminTeachers 获取教师列表（管理员，含院系名和评论数）。
// 使用 CTE 预聚合评论数，避免 correlated subquery 对每行执行 COUNT。
func (r *Repository) ListAdminTeachers(ctx context.Context, search string, departmentID int64, limit, offset int) ([]AdminTeacher, int, error) {
	ctx = withDBTable(ctx, "teachers")
	var qb strings.Builder
	qb.WriteString(`
		WITH review_counts AS (
			SELECT teacher_id, COUNT(*) AS review_count
			FROM reviews
			WHERE status != 'deleted'
			GROUP BY teacher_id
		)
		SELECT t.id, t.name, t.department_id, d.name,
		       COALESCE(rc.review_count, 0) AS review_count,
		       t.created_at,
		       COUNT(*) OVER() AS total
		FROM teachers t
		LEFT JOIN departments d ON d.id = t.department_id
		LEFT JOIN review_counts rc ON rc.teacher_id = t.id
		WHERE 1=1
	`)

	args := make([]interface{}, 0, 4)
	argIdx := 1

	if search != "" {
		fmt.Fprintf(&qb, ` AND t.name ILIKE $%d`, argIdx)
		args = append(args, "%"+httputil.EscapeLikePattern(search)+"%")
		argIdx++
	}
	if departmentID > 0 {
		fmt.Fprintf(&qb, ` AND t.department_id = $%d`, argIdx)
		args = append(args, departmentID)
		argIdx++
	}

	fmt.Fprintf(&qb, ` ORDER BY t.id DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
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
	ctx = withDBTable(ctx, "teachers")
	var t AdminTeacher
	err := r.db.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO teachers (school_id, name, department_id)
			SELECT d.school_id, $1, d.id
			FROM departments d
			WHERE d.id = $2
			RETURNING id, name, department_id, created_at
		)
		SELECT i.id, i.name, i.department_id, d.name, i.created_at
		FROM inserted i
		LEFT JOIN departments d ON d.id = i.department_id
	`, name, departmentID).Scan(&t.ID, &t.Name, &t.DepartmentID, &t.DepartmentName, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("CreateTeacher: %w", err)
	}
	return &t, nil
}

// UpdateTeacher 更新教师
func (r *Repository) UpdateTeacher(ctx context.Context, id int64, name string, departmentID *int64) error {
	ctx = withDBTable(ctx, "teachers")
	result, err := r.db.Exec(ctx, `
		UPDATE teachers
		SET name = $2,
		    department_id = $3,
		    school_id = COALESCE((SELECT d.school_id FROM departments d WHERE d.id = $3), teachers.school_id)
		WHERE id = $1
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
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var lockedTeacherID int64
		if err := tx.QueryRow(ctx, `SELECT id FROM teachers WHERE id = $1 FOR UPDATE`, id).Scan(&lockedTeacherID); err != nil {
			if err == pgx.ErrNoRows {
				return pgx.ErrNoRows
			}
			return fmt.Errorf("DeleteTeacher lock: %w", err)
		}

		var reviewCount int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE teacher_id = $1 AND status != 'deleted'`, lockedTeacherID).Scan(&reviewCount); err != nil {
			return fmt.Errorf("DeleteTeacher check: %w", err)
		}
		if reviewCount > 0 {
			return fmt.Errorf("%w: %d associated reviews", ErrTeacherHasReviews, reviewCount)
		}

		result, err := tx.Exec(ctx, `DELETE FROM teachers WHERE id = $1`, lockedTeacherID)
		if err != nil {
			return fmt.Errorf("DeleteTeacher: %w", err)
		}
		if result.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}
