package review

import (
	"context"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
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
	rows, err := r.db.Query(ctx, `
		SELECT r.term_id, COALESCE(t.name, r.term_id) as term_name,
			AVG(r.avg_rating) as avg_rating,
			COUNT(*) as count
		FROM reviews r
		LEFT JOIN terms t ON r.term_id = t.id
		WHERE r.course_id = $1 AND r.status = 'published' AND r.term_id IS NOT NULL
		GROUP BY r.term_id, t.name
		ORDER BY r.term_id ASC
	`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []RatingTrendItem
	for rows.Next() {
		var item RatingTrendItem
		if err := rows.Scan(&item.TermID, &item.TermName, &item.AvgRating, &item.Count); err != nil {
			return nil, err
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
func (r *Repository) ListHotCourses(ctx context.Context, period string, limit int) ([]HotCourse, error) {
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
		return nil, err
	}
	defer rows.Close()

	var list []HotCourse
	for rows.Next() {
		var item HotCourse
		if err := rows.Scan(&item.CourseID, &item.CourseName, &item.ReviewCount, &item.AvgRating); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ListCourseTeachers 获取课程的授课教师列表（含全局统计）
func (r *Repository) ListCourseTeachers(ctx context.Context, courseID int64) ([]CourseTeacherStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.name, COALESCE(d.name, '') AS department_name,
			(SELECT AVG(r2.avg_rating) FROM reviews r2
			 WHERE r2.teacher_id = t.id AND r2.status = 'published') AS avg_rating,
			(SELECT COUNT(DISTINCT r3.course_id) FROM reviews r3
			 WHERE r3.teacher_id = t.id AND r3.status = 'published') AS course_count,
			(SELECT COUNT(*) FROM reviews r4
			 WHERE r4.teacher_id = t.id AND r4.status = 'published') AS review_count
		FROM (SELECT DISTINCT teacher_id FROM reviews
		      WHERE course_id = $1 AND status = 'published' AND teacher_id IS NOT NULL) rt
		JOIN teachers t ON t.id = rt.teacher_id
		LEFT JOIN departments d ON d.id = t.department_id
		ORDER BY review_count DESC
	`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []CourseTeacherStats
	for rows.Next() {
		var s CourseTeacherStats
		if err := rows.Scan(&s.TeacherID, &s.TeacherName, &s.DepartmentName,
			&s.AvgRating, &s.CourseCount, &s.ReviewCount); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// GetTeacherName 获取教师名称
func (r *Repository) GetTeacherName(ctx context.Context, teacherID int64) (string, error) {
	var name string
	err := r.db.QueryRow(ctx, `
		SELECT name FROM teachers WHERE id = $1
	`, teacherID).Scan(&name)
	return name, err
}

// GetTeacherInfo 获取教师基本信息（名称 + 院系名称）
func (r *Repository) GetTeacherInfo(ctx context.Context, teacherID int64) (name, departmentName string, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT t.name, COALESCE(d.name, '')
		FROM teachers t
		LEFT JOIN departments d ON d.id = t.department_id
		WHERE t.id = $1
	`, teacherID).Scan(&name, &departmentName)
	return
}

// ListTeacherCourses 获取教师授课课程列表（含评分和评论数）
func (r *Repository) ListTeacherCourses(ctx context.Context, teacherID int64) ([]TeacherCourse, error) {
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
		return nil, err
	}
	defer rows.Close()

	var list []TeacherCourse
	for rows.Next() {
		var tc TeacherCourse
		if err := rows.Scan(&tc.ID, &tc.Name, &tc.AvgRating, &tc.ReviewCount); err != nil {
			return nil, err
		}
		list = append(list, tc)
	}
	return list, rows.Err()
}

// GetTeacherReviewCount 获取教师的评论总数
func (r *Repository) GetTeacherReviewCount(ctx context.Context, teacherID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM reviews
		WHERE teacher_id = $1 AND status = 'published'
	`, teacherID).Scan(&count)
	return count, err
}

// GetTeacherRatingStats 获取教师评分统计
func (r *Repository) GetTeacherRatingStats(ctx context.Context, teacherID int64) ([]TeacherRatingStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, teacher_id, term_id, dimension_key, avg_rating, rating_count, rating_dist, updated_at
		FROM teacher_rating_stats
		WHERE teacher_id = $1
		ORDER BY term_id NULLS FIRST, dimension_key
	`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []TeacherRatingStats
	for rows.Next() {
		var s TeacherRatingStats
		if err := rows.Scan(
			&s.ID, &s.TeacherID, &s.TermID, &s.DimensionKey,
			&s.AvgRating, &s.RatingCount, &s.RatingDist, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// RefreshTeacherRatingStats 刷新教师评分统计（从 reviews 表聚合）
func (r *Repository) RefreshTeacherRatingStats(ctx context.Context, teacherID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 先聚合出统计数据
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
		return err
	}
	defer rows.Close()

	type statRow struct {
		termID       *string
		dimensionKey string
		avgRating    float64
		ratingCount  int
		ratingDist   []byte
	}
	var statRows []statRow
	for rows.Next() {
		var s statRow
		if err := rows.Scan(&s.termID, &s.dimensionKey, &s.avgRating, &s.ratingCount, &s.ratingDist); err != nil {
			return err
		}
		statRows = append(statRows, s)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// 逐行 upsert，使用 Go 端生成的 UUIDv7
	for _, s := range statRows {
		newID, err := id.New()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO teacher_rating_stats (id, teacher_id, term_id, dimension_key, avg_rating, rating_count, rating_dist, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
			ON CONFLICT (teacher_id, term_id, dimension_key)
			DO UPDATE SET
				avg_rating = EXCLUDED.avg_rating,
				rating_count = EXCLUDED.rating_count,
				rating_dist = EXCLUDED.rating_dist,
				updated_at = NOW()
		`, newID, teacherID, s.termID, s.dimensionKey, s.avgRating, s.ratingCount, s.ratingDist)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ListActiveSensitiveWords 获取所有启用的敏感词
func (r *Repository) ListActiveSensitiveWords(ctx context.Context) ([]SensitiveWord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, word, category, level, is_active, created_at
		FROM sensitive_words
		WHERE is_active = true
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []SensitiveWord
	for rows.Next() {
		var w SensitiveWord
		if err := rows.Scan(&w.ID, &w.Word, &w.Category, &w.Level, &w.IsActive, &w.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}
