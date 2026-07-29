package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/id"
)

// RatingTrendItem 评分趋势项
type RatingTrendItem struct {
	TermID    string  `json:"termID"`
	TermName  string  `json:"termName"`
	AvgRating float64 `json:"avgRating"`
	Count     int     `json:"count"`
}

// GetRatingTrend 获取课程评分趋势
func (r *Repository) GetRatingTrend(ctx context.Context, courseID int64) ([]RatingTrendItem, error) {
	ctx = withDBTable(ctx, "reviews")
	rows, err := r.db.Query(ctx, `
		SELECT r.term_id, COALESCE(t.name, r.term_id) as term_name,
			AVG(r.avg_rating) as avg_rating,
			COUNT(*) as count
		FROM reviews r
		LEFT JOIN terms t ON r.term_id = t.id
		WHERE r.course_id = $1 AND r.status = 'published' AND r.term_id IS NOT NULL AND r.term_id != ''
		GROUP BY r.term_id, t.name
		ORDER BY r.term_id ASC
	`, courseID)
	if err != nil {
		return nil, fmt.Errorf("GetRatingTrend: %w", err)
	}
	defer rows.Close()

	list := make([]RatingTrendItem, 0, 16)
	for rows.Next() {
		var item RatingTrendItem
		if err := rows.Scan(&item.TermID, &item.TermName, &item.AvgRating, &item.Count); err != nil {
			return nil, fmt.Errorf("GetRatingTrend scan: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// HotCourse 热门课程
type HotCourse struct {
	CourseID    int64   `json:"courseID"`
	CourseName  string  `json:"courseName"`
	ReviewCount int     `json:"reviewCount"`
	AvgRating   float64 `json:"avgRating"`
}

// ListHotCourses 获取热门课程排行
// timeFilter 的值仅来自下方 switch 硬编码的 INTERVAL 字面量，不包含任何用户输入，无 SQL 注入风险
func (r *Repository) ListHotCourses(ctx context.Context, period string, limit int) ([]HotCourse, error) {
	ctx = withDBTable(ctx, "courses")
	// 安全保证：timeFilter 仅从 switch 硬编码值中选取，period 参数不直接拼入 SQL
	var timeFilter string
	switch period {
	case "week":
		timeFilter = "AND r.created_at >= NOW() - INTERVAL '7 days'"
	case "month":
		timeFilter = "AND r.created_at >= NOW() - INTERVAL '30 days'"
	default:
		timeFilter = ""
	}

	query := `
		SELECT c.id, c.name, COUNT(r.id) as review_count,
			COALESCE(AVG(r.avg_rating), 0) as avg_rating
		FROM courses c
		LEFT JOIN reviews r ON c.id = r.course_id AND r.status = 'published' ` + timeFilter + `
		GROUP BY c.id, c.name
		HAVING COUNT(r.id) > 0
		ORDER BY review_count DESC, avg_rating DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("ListHotCourses: %w", err)
	}
	defer rows.Close()

	list := make([]HotCourse, 0, limit)
	for rows.Next() {
		var item HotCourse
		if err := rows.Scan(&item.CourseID, &item.CourseName, &item.ReviewCount, &item.AvgRating); err != nil {
			return nil, fmt.Errorf("ListHotCourses scan: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ListCourseTeachers 获取课程的授课教师列表（含全局统计，CTE + JOIN 避免 IN 子查询性能问题）
func (r *Repository) ListCourseTeachers(ctx context.Context, courseID int64) ([]CourseTeacherStats, error) {
	ctx = withDBTable(ctx, "reviews")
	rows, err := r.db.Query(ctx, `
		WITH course_teachers AS (
			SELECT DISTINCT teacher_id
			FROM reviews
			WHERE course_id = $1 AND status = 'published' AND teacher_id IS NOT NULL
		)
		SELECT t.id, t.name, COALESCE(d.name, '') AS department_name,
			AVG(r.avg_rating) AS avg_rating,
			COUNT(DISTINCT r.course_id) AS course_count,
			COUNT(*) AS review_count
		FROM course_teachers ct
		JOIN reviews r ON r.teacher_id = ct.teacher_id AND r.status = 'published'
		JOIN teachers t ON t.id = ct.teacher_id
		LEFT JOIN departments d ON d.id = t.department_id
		GROUP BY t.id, t.name, d.name
		ORDER BY review_count DESC
	`, courseID)
	if err != nil {
		return nil, fmt.Errorf("ListCourseTeachers: %w", err)
	}
	defer rows.Close()

	list := make([]CourseTeacherStats, 0, 8)
	for rows.Next() {
		var s CourseTeacherStats
		if err := rows.Scan(&s.TeacherID, &s.TeacherName, &s.DepartmentName,
			&s.AvgRating, &s.CourseCount, &s.ReviewCount); err != nil {
			return nil, fmt.Errorf("ListCourseTeachers scan: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// GetTeacherName 获取教师名称
func (r *Repository) GetTeacherName(ctx context.Context, teacherID int64) (string, error) {
	ctx = withDBTable(ctx, "teachers")
	var name string
	err := r.db.QueryRow(ctx, `
		SELECT name FROM teachers WHERE id = $1
	`, teacherID).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("GetTeacherName: %w", err)
	}
	return name, nil
}

// GetTeacherInfo 获取教师基本信息（名称 + 院系名称）
func (r *Repository) GetTeacherInfo(ctx context.Context, teacherID int64) (name, departmentName string, err error) {
	ctx = withDBTable(ctx, "teachers")
	err = r.db.QueryRow(ctx, `
		SELECT t.name, COALESCE(d.name, '')
		FROM teachers t
		LEFT JOIN departments d ON d.id = t.department_id
		WHERE t.id = $1
	`, teacherID).Scan(&name, &departmentName)
	if err != nil {
		err = fmt.Errorf("GetTeacherInfo: %w", err)
	}
	return
}

// ListTeacherCourses 获取教师授课课程列表（含评分和评论数）
func (r *Repository) ListTeacherCourses(ctx context.Context, teacherID int64) ([]TeacherCourse, error) {
	ctx = withDBTable(ctx, "reviews")
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.name,
			AVG(CASE WHEN r.avg_rating > 0 THEN r.avg_rating END) AS avg_rating,
			COUNT(r.id) AS review_count
		FROM reviews r
		JOIN courses c ON c.id = r.course_id
		WHERE r.teacher_id = $1 AND r.status = 'published'
		GROUP BY c.id, c.name
		ORDER BY review_count DESC
	`, teacherID)
	if err != nil {
		return nil, fmt.Errorf("ListTeacherCourses: %w", err)
	}
	defer rows.Close()

	list := make([]TeacherCourse, 0, 8)
	for rows.Next() {
		var tc TeacherCourse
		if err := rows.Scan(&tc.ID, &tc.Name, &tc.AvgRating, &tc.ReviewCount); err != nil {
			return nil, fmt.Errorf("ListTeacherCourses scan: %w", err)
		}
		list = append(list, tc)
	}
	return list, rows.Err()
}

// GetTeacherReviewCount 获取教师的评论总数
func (r *Repository) GetTeacherReviewCount(ctx context.Context, teacherID int64) (int, error) {
	ctx = withDBTable(ctx, "reviews")
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM reviews
		WHERE teacher_id = $1 AND status = 'published'
	`, teacherID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("GetTeacherReviewCount: %w", err)
	}
	return count, nil
}

// GetTeacherRatingStats 获取教师评分统计
func (r *Repository) GetTeacherRatingStats(ctx context.Context, teacherID int64) ([]TeacherRatingStats, error) {
	ctx = withDBTable(ctx, "teacher_rating_stats")
	rows, err := r.db.Query(ctx, `
		SELECT id, teacher_id, term_id, dimension_key, avg_rating, rating_count, rating_dist, updated_at
		FROM teacher_rating_stats
		WHERE teacher_id = $1
		ORDER BY term_id NULLS FIRST, dimension_key
	`, teacherID)
	if err != nil {
		return nil, fmt.Errorf("GetTeacherRatingStats: %w", err)
	}
	defer rows.Close()

	list := make([]TeacherRatingStats, 0, 20)
	for rows.Next() {
		var s TeacherRatingStats
		if err := rows.Scan(
			&s.ID, &s.TeacherID, &s.TermID, &s.DimensionKey,
			&s.AvgRating, &s.RatingCount, &s.RatingDist, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("GetTeacherRatingStats scan: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// RefreshTeacherRatingStats 刷新教师评分统计（从 reviews 表聚合）。
func (r *Repository) RefreshTeacherRatingStats(ctx context.Context, teacherID int64) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return r.RefreshTeacherRatingStatsTx(ctx, tx, teacherID)
	})
}

// RefreshTeacherRatingStatsTx 在事务内刷新教师评分统计。
func (r *Repository) RefreshTeacherRatingStatsTx(ctx context.Context, tx pgx.Tx, teacherID int64) error {
	rows, err := tx.Query(ctx, `
		WITH base AS (
			SELECT r.term_id, d.key AS dimension_key,
				(r.ratings->>d.key)::numeric AS rating_num,
				(r.ratings->>d.key)::text AS rating_val
			FROM reviews r
			CROSS JOIN rating_dimensions d
			WHERE r.teacher_id = $1 AND r.status = 'published' AND d.is_active = true
				AND r.ratings ? d.key
		),
		stats AS (
			SELECT term_id, dimension_key,
				AVG(rating_num) AS avg_rating,
				COUNT(*) AS rating_count
			FROM base
			GROUP BY term_id, dimension_key
		),
		dist AS (
			SELECT term_id, dimension_key,
				jsonb_object_agg(rating_val, cnt) AS rating_dist
			FROM (
				SELECT term_id, dimension_key, rating_val, COUNT(*) AS cnt
				FROM base
				GROUP BY term_id, dimension_key, rating_val
			) sub
			GROUP BY term_id, dimension_key
		)
		SELECT s.term_id, s.dimension_key, s.avg_rating, s.rating_count,
			COALESCE(d.rating_dist, '{}'::jsonb)
		FROM stats s
		LEFT JOIN dist d ON s.term_id IS NOT DISTINCT FROM d.term_id AND s.dimension_key = d.dimension_key
	`, teacherID)
	if err != nil {
		return fmt.Errorf("RefreshTeacherRatingStatsTx query: %w", err)
	}
	defer rows.Close()

	type statRow struct {
		termID       *string
		dimensionKey string
		avgRating    float64
		ratingCount  int
		ratingDist   []byte
	}

	statRows := make([]statRow, 0, 20)
	for rows.Next() {
		var s statRow
		if err := rows.Scan(&s.termID, &s.dimensionKey, &s.avgRating, &s.ratingCount, &s.ratingDist); err != nil {
			return fmt.Errorf("RefreshTeacherRatingStatsTx scan: %w", err)
		}
		statRows = append(statRows, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("RefreshTeacherRatingStatsTx rows iteration: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM teacher_rating_stats WHERE teacher_id = $1`, teacherID); err != nil {
		return fmt.Errorf("RefreshTeacherRatingStatsTx cleanup existing: %w", err)
	}
	if len(statRows) == 0 {
		return nil
	}

	args := make([]any, 0, len(statRows)*7)
	var query strings.Builder
	query.WriteString(`
		INSERT INTO teacher_rating_stats (
			id, teacher_id, term_id, dimension_key, avg_rating, rating_count, rating_dist, updated_at
		) VALUES
	`)
	for i, s := range statRows {
		newID, err := id.New()
		if err != nil {
			return fmt.Errorf("RefreshTeacherRatingStatsTx generate id: %w", err)
		}
		if i > 0 {
			query.WriteString(",")
		}
		base := i * 7
		fmt.Fprintf(&query, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,NOW())", base+1, base+2, base+3, base+4, base+5, base+6, base+7)
		args = append(args, newID, teacherID, s.termID, s.dimensionKey, s.avgRating, s.ratingCount, s.ratingDist)
	}

	if _, err := tx.Exec(ctx, query.String(), args...); err != nil {
		return fmt.Errorf("RefreshTeacherRatingStatsTx insert: %w", err)
	}
	return nil
}

// ListActiveSensitiveWords 获取所有启用的敏感词
func (r *Repository) ListActiveSensitiveWords(ctx context.Context) ([]SensitiveWord, error) {
	ctx = withDBTable(ctx, "sensitive_words")
	rows, err := r.db.Query(ctx, `
		SELECT id, word, category, level, is_active, created_at
		FROM sensitive_words
		WHERE is_active = true
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("ListActiveSensitiveWords: %w", err)
	}
	defer rows.Close()

	list := make([]SensitiveWord, 0, 64)
	for rows.Next() {
		var w SensitiveWord
		if err := rows.Scan(&w.ID, &w.Word, &w.Category, &w.Level, &w.IsActive, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("ListActiveSensitiveWords scan: %w", err)
		}
		list = append(list, w)
	}
	return list, rows.Err()
}
