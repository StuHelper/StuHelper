package course

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

// ErrNotFound 表示数据库查询未找到记录的哨兵错误
var ErrNotFound = errors.New("record not found")

// Repository 课程数据访问层
type Repository struct {
	db *db.DB
}

// NewRepository 创建数据访问层
func NewRepository(database *db.DB) *Repository {
	if database == nil {
		panic("course.NewRepository: database must not be nil")
	}
	return &Repository{db: database}
}

func withDBTable(ctx context.Context, table string) context.Context {
	return db.WithTableHint(ctx, table)
}

// ListDepartments 获取院系列表
func (r *Repository) ListDepartments(ctx context.Context, category string) ([]Department, error) {
	ctx = withDBTable(ctx, "departments")
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

// ListTerms 获取学期列表
func (r *Repository) ListTerms(ctx context.Context) ([]Term, error) {
	ctx = withDBTable(ctx, "terms")
	rows, err := r.db.Query(ctx, `
		SELECT id, school_id, name, is_current
		FROM terms
		ORDER BY is_current DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	terms := make([]Term, 0)
	for rows.Next() {
		var item Term
		if err := rows.Scan(&item.ID, &item.SchoolID, &item.Name, &item.IsCurrent); err != nil {
			return nil, err
		}
		terms = append(terms, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return terms, nil
}

// CountCourses 统计课程总数（可按院系过滤）
func (r *Repository) CountCourses(ctx context.Context, departmentID int64) (int, error) {
	ctx = withDBTable(ctx, "courses")
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM courses WHERE ($1::bigint = 0 OR department_id = $1)",
		departmentID).Scan(&count)
	return count, err
}

// CountDepartments 统计院系总数
func (r *Repository) CountDepartments(ctx context.Context) (int, error) {
	ctx = withDBTable(ctx, "departments")
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM departments").Scan(&count)
	return count, err
}

// CountStats 单次查询同时获取课程数和院系数
func (r *Repository) CountStats(ctx context.Context) (courseCount, departmentCount int, err error) {
	ctx = withDBTable(ctx, "courses")
	err = r.db.QueryRow(ctx,
		"SELECT (SELECT COUNT(*) FROM courses), (SELECT COUNT(*) FROM departments)").
		Scan(&courseCount, &departmentCount)
	return
}

// ListCourses 获取课程列表（支持搜索、院系和分类过滤），使用窗口函数一次性返回数据和总数
func (r *Repository) ListCourses(ctx context.Context, query string, departmentID int64, category, sort string, limit, offset int) ([]Course, int, error) {
	ctx = withDBTable(ctx, "courses")
	orderBy := "c.name ASC"
	switch sort {
	case CourseSortCredits:
		orderBy = "c.credits DESC, c.name ASC"
	case CourseSortReviewCount:
		orderBy = "c.review_count DESC, c.name ASC"
	}

	pattern := "%" + httputil.EscapeLikePattern(query) + "%"
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.category, c.review_count,
			COUNT(*) OVER() AS total
		FROM courses c
		LEFT JOIN departments d ON d.id = c.department_id
		WHERE ($1 = '%%' OR c.name ILIKE $1 ESCAPE '\' OR c.code ILIKE $1 ESCAPE '\')
		  AND ($2::bigint = 0 OR c.department_id = $2)
		  AND ($5 = '' OR c.category = $5)
		ORDER BY `+orderBy+`
		LIMIT $3 OFFSET $4
	`, pattern, departmentID, limit, offset, category)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return r.scanCoursesWithTotal(rows)
}

// SearchCourses 搜索课程，使用窗口函数一次性返回数据和总数
func (r *Repository) SearchCourses(ctx context.Context, query string, limit, offset int) ([]Course, int, error) {
	ctx = withDBTable(ctx, "courses")
	pattern := "%" + httputil.EscapeLikePattern(query) + "%"
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
	ctx = withDBTable(ctx, "courses")
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
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("course id %d: %w", id, ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query course by id %d: %w", id, err)
	}
	return &item, nil
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
	ctx = withDBTable(ctx, "course_categories")
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

// FavoriteExists 检查单个课程是否被用户收藏
func (r *Repository) FavoriteExists(ctx context.Context, userHash string, courseID int64) (bool, error) {
	ctx = withDBTable(ctx, "course_favorites")
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM course_favorites WHERE user_hash = $1 AND course_id = $2)
	`, userHash, courseID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("FavoriteExists: %w", err)
	}
	return exists, nil
}

// BatchFavoritedCourseIDs 批量查询用户已收藏的课程 ID 集合
func (r *Repository) BatchFavoritedCourseIDs(ctx context.Context, userHash string, courseIDs []int64) (map[int64]bool, error) {
	if len(courseIDs) == 0 {
		return nil, nil
	}
	ctx = withDBTable(ctx, "course_favorites")
	rows, err := r.db.Query(ctx, `
		SELECT course_id FROM course_favorites
		WHERE user_hash = $1 AND course_id = ANY($2)
	`, userHash, courseIDs)
	if err != nil {
		return nil, fmt.Errorf("BatchFavoritedCourseIDs: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]bool, len(courseIDs))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("BatchFavoritedCourseIDs scan: %w", err)
		}
		result[id] = true
	}
	return result, rows.Err()
}

// ListCoursesGroupedByDepartment 一次查询获取所有课程，按院系分组返回。
// 数据库端按 department 排序，Go 侧分组，避免前端全量拉取 + 聚合。
func (r *Repository) ListCoursesGroupedByDepartment(ctx context.Context) ([]DepartmentGroup, error) {
	ctx = withDBTable(ctx, "courses")
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.category, c.review_count
		FROM courses c
		LEFT JOIN departments d ON d.id = c.department_id
		ORDER BY d.name ASC, c.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListCoursesGroupedByDepartment: %w", err)
	}
	defer rows.Close()

	var groups []DepartmentGroup
	groupIndex := make(map[int64]int) // departmentID → groups 下标

	for rows.Next() {
		var item Course
		if err := rows.Scan(
			&item.ID, &item.SchoolID, &item.DepartmentID, &item.DepartmentName,
			&item.Code, &item.Name, &item.Credits, &item.Category, &item.ReviewCount,
		); err != nil {
			return nil, fmt.Errorf("ListCoursesGroupedByDepartment scan: %w", err)
		}

		idx, exists := groupIndex[item.DepartmentID]
		if !exists {
			idx = len(groups)
			groupIndex[item.DepartmentID] = idx
			groups = append(groups, DepartmentGroup{
				DepartmentID:   item.DepartmentID,
				DepartmentName: item.DepartmentName,
				Courses:        make([]Course, 0, 16),
			})
		}
		groups[idx].Courses = append(groups[idx].Courses, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListCoursesGroupedByDepartment rows: %w", err)
	}
	return groups, nil
}
