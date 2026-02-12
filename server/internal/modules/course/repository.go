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

// CountCourses 统计课程总数（可按院系过滤）
func (r *Repository) CountCourses(ctx context.Context, departmentID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM courses WHERE ($1::bigint = 0 OR department_id = $1)",
		departmentID).Scan(&count)
	return count, err
}

// CountDepartments 统计院系总数
func (r *Repository) CountDepartments(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM departments").Scan(&count)
	return count, err
}

// ListCourses 获取课程列表（可按院系和分类过滤），使用窗口函数一次性返回数据和总数
func (r *Repository) ListCourses(ctx context.Context, departmentID int64, category string, limit, offset int) ([]Course, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.category, c.review_count,
			COUNT(*) OVER() AS total
		FROM courses c
		LEFT JOIN departments d ON d.id = c.department_id
		WHERE ($1::bigint = 0 OR c.department_id = $1)
		  AND ($4 = '' OR c.category = $4)
		ORDER BY c.name ASC
		LIMIT $2 OFFSET $3
	`, departmentID, limit, offset, category)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return r.scanCoursesWithTotal(rows)
}


// SearchCourses 搜索课程，使用窗口函数一次性返回数据和总数
func (r *Repository) SearchCourses(ctx context.Context, pattern string, limit, offset int) ([]Course, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.category, c.review_count,
			COUNT(*) OVER() AS total
		FROM courses c
		LEFT JOIN departments d ON d.id = c.department_id
		WHERE c.name ILIKE $1 ESCAPE '\' OR c.code ILIKE $1 ESCAPE '\'
		ORDER BY c.name ASC
		LIMIT $2 OFFSET $3
	`, pattern, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return r.scanCoursesWithTotal(rows)
}

// GetCourseByID 根据ID获取课程
func (r *Repository) GetCourseByID(ctx context.Context, id int64) (*Course, error) {
	var item Course
	err := r.db.QueryRow(ctx, `
		SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.category, c.review_count
		FROM courses c
		LEFT JOIN departments d ON d.id = c.department_id
		WHERE c.id = $1
	`, id).Scan(
		&item.ID, &item.SchoolID, &item.DepartmentID, &item.DepartmentName,
		&item.Code, &item.Name, &item.Credits, &item.Category, &item.ReviewCount,
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
			&item.Code, &item.Name, &item.Credits, &item.Category, &item.ReviewCount,
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

// scanCoursesWithTotal 扫描课程结果集（含 COUNT(*) OVER() 总数）
func (r *Repository) scanCoursesWithTotal(rows pgx.Rows) ([]Course, int, error) {
	list := make([]Course, 0)
	var total int
	for rows.Next() {
		var item Course
		if err := rows.Scan(
			&item.ID, &item.SchoolID, &item.DepartmentID, &item.DepartmentName,
			&item.Code, &item.Name, &item.Credits, &item.Category, &item.ReviewCount,
			&total,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListCourseCategories 获取课程分类列表
func (r *Repository) ListCourseCategories(ctx context.Context) ([]CourseCategory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, school_id, name, sort_order
		FROM course_categories
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]CourseCategory, 0)
	for rows.Next() {
		var item CourseCategory
		if err := rows.Scan(&item.ID, &item.SchoolID, &item.Name, &item.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
