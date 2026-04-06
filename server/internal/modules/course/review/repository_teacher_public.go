package review

import (
	"context"
	"fmt"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

// ListPublicTeachers 获取公开教师列表（支持搜索、院系过滤、多排序）
func (r *Repository) ListPublicTeachers(ctx context.Context, search string, departmentID *int64, sort string, limit, offset int) ([]TeacherSummary, int, error) {
	var qb strings.Builder
	qb.WriteString(`
		SELECT t.id, t.name, COALESCE(d.name, '') AS department_name,
			AVG(r.avg_rating) AS avg_rating,
			COUNT(DISTINCT r.course_id) AS course_count,
			COUNT(r.id) AS review_count,
			COUNT(*) OVER() AS total
		FROM teachers t
		LEFT JOIN departments d ON d.id = t.department_id
		LEFT JOIN reviews r ON r.teacher_id = t.id AND r.status = 'published'
		WHERE 1=1
	`)

	args := make([]interface{}, 0, 4)
	argIdx := 1

	if search != "" {
		fmt.Fprintf(&qb, ` AND t.name ILIKE $%d`, argIdx)
		args = append(args, "%"+httputil.EscapeLikePattern(search)+"%")
		argIdx++
	}
	if departmentID != nil && *departmentID > 0 {
		fmt.Fprintf(&qb, ` AND t.department_id = $%d`, argIdx)
		args = append(args, *departmentID)
		argIdx++
	}

	qb.WriteString(` GROUP BY t.id, t.name, d.name`)

	// 排序：仅使用硬编码的 ORDER BY 子句，sort 参数不直接拼入 SQL
	switch sort {
	case "rating":
		qb.WriteString(` ORDER BY avg_rating DESC NULLS LAST, review_count DESC`)
	case "name":
		qb.WriteString(` ORDER BY t.name ASC`)
	default: // "reviews" 或其他
		qb.WriteString(` ORDER BY review_count DESC, avg_rating DESC NULLS LAST`)
	}

	fmt.Fprintf(&qb, ` LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, qb.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListPublicTeachers: %w", err)
	}
	defer rows.Close()

	list := make([]TeacherSummary, 0, limit)
	var total int
	for rows.Next() {
		var t TeacherSummary
		if err := rows.Scan(&t.TeacherID, &t.TeacherName, &t.DepartmentName,
			&t.AvgRating, &t.CourseCount, &t.ReviewCount, &total); err != nil {
			return nil, 0, fmt.Errorf("ListPublicTeachers scan: %w", err)
		}
		list = append(list, t)
	}
	return list, total, rows.Err()
}

// ListHotTeachers 获取热门教师列表（按评论数+评分排序）
func (r *Repository) ListHotTeachers(ctx context.Context, limit int) ([]TeacherSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.name, COALESCE(d.name, '') AS department_name,
			AVG(r.avg_rating) AS avg_rating,
			COUNT(DISTINCT r.course_id) AS course_count,
			COUNT(r.id) AS review_count
		FROM teachers t
		LEFT JOIN departments d ON d.id = t.department_id
		JOIN reviews r ON r.teacher_id = t.id AND r.status = 'published'
		GROUP BY t.id, t.name, d.name
		HAVING COUNT(r.id) > 0
		ORDER BY review_count DESC, avg_rating DESC NULLS LAST
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("ListHotTeachers: %w", err)
	}
	defer rows.Close()

	list := make([]TeacherSummary, 0, limit)
	for rows.Next() {
		var t TeacherSummary
		if err := rows.Scan(&t.TeacherID, &t.TeacherName, &t.DepartmentName,
			&t.AvgRating, &t.CourseCount, &t.ReviewCount); err != nil {
			return nil, fmt.Errorf("ListHotTeachers scan: %w", err)
		}
		list = append(list, t)
	}
	return list, rows.Err()
}
