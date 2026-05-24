package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

// RatingStatRow 评分统计行
type RatingStatRow struct {
	TermID       *string
	DimensionKey string
	AvgRating    float64
	RatingCount  int
	RatingDist   []byte
}

// ListRatingDimensions 获取评分维度列表
func (r *Repository) ListRatingDimensions(ctx context.Context) ([]RatingDimension, error) {
	ctx = withDBTable(ctx, "rating_dimensions")
	rows, err := r.db.Query(ctx, `
		SELECT id, school_id, key, name, description, sort_order, is_active, created_at, updated_at
		FROM rating_dimensions
		WHERE is_active = true
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dimensions := make([]RatingDimension, 0, 10)
	for rows.Next() {
		var d RatingDimension
		if err := rows.Scan(
			&d.ID, &d.SchoolID, &d.Key, &d.Name, &d.Description,
			&d.SortOrder, &d.IsActive, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		dimensions = append(dimensions, d)
	}
	return dimensions, rows.Err()
}

// GetDimensionNames 获取维度名称映射
func (r *Repository) GetDimensionNames(ctx context.Context) (map[string]string, error) {
	ctx = withDBTable(ctx, "rating_dimensions")
	rows, err := r.db.Query(ctx, `SELECT key, name FROM rating_dimensions WHERE is_active = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, name string
		if err := rows.Scan(&key, &name); err != nil {
			return nil, err
		}
		result[key] = name
	}
	return result, rows.Err()
}

// ListCourseRatingStats 获取课程评分统计
func (r *Repository) ListCourseRatingStats(ctx context.Context, courseID int64) ([]RatingStatRow, error) {
	ctx = withDBTable(ctx, "course_rating_stats")
	rows, err := r.db.Query(ctx, `
		SELECT term_id, dimension_key, avg_rating, rating_count, rating_dist
		FROM course_rating_stats
		WHERE course_id = $1
		ORDER BY term_id NULLS FIRST
	`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]RatingStatRow, 0, 20)
	for rows.Next() {
		var s RatingStatRow
		if err := rows.Scan(&s.TermID, &s.DimensionKey, &s.AvgRating, &s.RatingCount, &s.RatingDist); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// RefreshCourseRatingStats 在独立事务中刷新课程评分统计。
func (r *Repository) RefreshCourseRatingStats(ctx context.Context, courseID int64) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return r.RefreshCourseRatingStatsTx(ctx, tx, courseID)
	})
}

// RefreshCourseRatingStatsTx 在事务内刷新课程评分统计。
func (r *Repository) RefreshCourseRatingStatsTx(ctx context.Context, tx pgx.Tx, courseID int64) error {
	rows, err := tx.Query(ctx, `
		WITH base AS (
			SELECT r.term_id, d.key AS dimension_key,
				(r.ratings->>d.key)::numeric AS rating_num,
				(r.ratings->>d.key)::text AS rating_val
			FROM reviews r
			CROSS JOIN rating_dimensions d
			WHERE r.course_id = $1 AND r.status = 'published' AND d.is_active = true
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
	`, courseID)
	if err != nil {
		return fmt.Errorf("RefreshCourseRatingStatsTx query: %w", err)
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
			return fmt.Errorf("RefreshCourseRatingStatsTx scan: %w", err)
		}
		statRows = append(statRows, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("RefreshCourseRatingStatsTx rows iteration: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM course_rating_stats WHERE course_id = $1`, courseID); err != nil {
		return fmt.Errorf("RefreshCourseRatingStatsTx cleanup existing: %w", err)
	}
	if len(statRows) == 0 {
		return nil
	}

	args := make([]any, 0, len(statRows)*7)
	var query strings.Builder
	query.WriteString(`
		INSERT INTO course_rating_stats (
			id, course_id, term_id, dimension_key, avg_rating, rating_count, rating_dist, updated_at
		) VALUES
	`)
	for i, s := range statRows {
		newID, err := id.New()
		if err != nil {
			return fmt.Errorf("RefreshCourseRatingStatsTx generate id: %w", err)
		}
		if i > 0 {
			query.WriteString(",")
		}
		base := i * 7
		fmt.Fprintf(&query, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,NOW())", base+1, base+2, base+3, base+4, base+5, base+6, base+7)
		args = append(args, newID, courseID, s.termID, s.dimensionKey, s.avgRating, s.ratingCount, s.ratingDist)
	}

	if _, err := tx.Exec(ctx, query.String(), args...); err != nil {
		return fmt.Errorf("RefreshCourseRatingStatsTx insert: %w", err)
	}
	return nil
}
