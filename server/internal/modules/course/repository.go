package course

import (
	"context"

	"github.com/jackc/pgx/v5"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

// Repository 课程数据访问层
type Repository struct {
	db *db.DB
}

// NewRepository 创建数据访问层
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// ListDepartments 获取院系列表
func (r *Repository) ListDepartments(ctx context.Context, category string) ([]Department, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, school_id, name, short_name, category, sort_order
		FROM departments
		WHERE ($1 = '' OR category = $1)
		ORDER BY sort_order ASC
	`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	departments := make([]Department, 0)
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.SchoolID, &d.Name, &d.ShortName, &d.Category, &d.SortOrder); err != nil {
			return nil, err
		}
		departments = append(departments, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return departments, nil
}

// CountCourses 统计课程总数
func (r *Repository) CountCourses(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM courses").Scan(&count)
	return count, err
}

// CountDepartments 统计院系总数
func (r *Repository) CountDepartments(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM departments").Scan(&count)
	return count, err
}

// ListCourses 获取课程列表
func (r *Repository) ListCourses(ctx context.Context, limit, offset int) ([]Course, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.review_count
		FROM courses c
		LEFT JOIN departments d ON d.id = c.department_id
		ORDER BY c.name ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCourses(rows)
}

// SearchCoursesCount 搜索课程计数
func (r *Repository) SearchCoursesCount(ctx context.Context, pattern string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM courses WHERE name ILIKE $1 ESCAPE '\' OR code ILIKE $1 ESCAPE '\'`,
		pattern).Scan(&count)
	return count, err
}

// SearchCourses 搜索课程
func (r *Repository) SearchCourses(ctx context.Context, pattern string, limit, offset int) ([]Course, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.review_count
		FROM courses c
		LEFT JOIN departments d ON d.id = c.department_id
		WHERE c.name ILIKE $1 ESCAPE '\' OR c.code ILIKE $1 ESCAPE '\'
		ORDER BY c.name ASC
		LIMIT $2 OFFSET $3
	`, pattern, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCourses(rows)
}

// GetCourseByID 根据ID获取课程
func (r *Repository) GetCourseByID(ctx context.Context, id int64) (*Course, error) {
	var item Course
	err := r.db.QueryRow(ctx, `
		SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.review_count
		FROM courses c
		LEFT JOIN departments d ON d.id = c.department_id
		WHERE c.id = $1
	`, id).Scan(
		&item.ID, &item.SchoolID, &item.DepartmentID, &item.DepartmentName,
		&item.Code, &item.Name, &item.Credits, &item.ReviewCount,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// scanCourses 扫描课程结果集
func (r *Repository) scanCourses(rows pgx.Rows) ([]Course, error) {
	list := make([]Course, 0)
	for rows.Next() {
		var item Course
		if err := rows.Scan(
			&item.ID, &item.SchoolID, &item.DepartmentID, &item.DepartmentName,
			&item.Code, &item.Name, &item.Credits, &item.ReviewCount,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
