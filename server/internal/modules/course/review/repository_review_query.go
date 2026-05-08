package review

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

const reviewCoreFieldsBaseQuery = `
	SELECT user_hash, course_id, COALESCE(teacher_id, 0), status
	FROM reviews
	WHERE id = $1
`

type reviewCoreFields struct {
	userHash  string
	courseID  int64
	teacherID *int64
	status    string
}

type reviewUpdateState struct {
	userHash  string
	courseID  int64
	teacherID *int64
	status    string
	title     string
	content   string
	grade     string
	ratings   ReviewRatings
}

type reviewCoreRow interface {
	Scan(dest ...any) error
}

func scanReviewCoreFields(row reviewCoreRow) (reviewCoreFields, error) {
	var (
		fields    reviewCoreFields
		teacherID int64
	)
	if err := row.Scan(&fields.userHash, &fields.courseID, &teacherID, &fields.status); err != nil {
		return reviewCoreFields{}, err
	}
	if teacherID != 0 {
		fields.teacherID = &teacherID
	}
	return fields, nil
}

func reviewCoreFieldsQuery(forUpdate bool) string {
	if forUpdate {
		return reviewCoreFieldsBaseQuery + " FOR UPDATE"
	}
	return reviewCoreFieldsBaseQuery
}

func (r *Repository) getReviewCoreFields(ctx context.Context, reviewID string, forUpdate bool) (reviewCoreFields, error) {
	return scanReviewCoreFields(r.db.QueryRow(ctx, reviewCoreFieldsQuery(forUpdate), reviewID))
}

func getReviewCoreFieldsTx(ctx context.Context, tx pgx.Tx, reviewID string, forUpdate bool) (reviewCoreFields, error) {
	return scanReviewCoreFields(tx.QueryRow(ctx, reviewCoreFieldsQuery(forUpdate), reviewID))
}

// GetReviewByID 根据ID获取已发布的评论（仅返回 published 状态）
// 包装 pgx.ErrNoRows 为业务层 sentinel error
func (r *Repository) GetReviewByID(ctx context.Context, reviewID string) (*Review, error) {
	var item Review
	err := r.db.QueryRow(ctx, `
		SELECT r.id, r.course_id, COALESCE(c.name, ''), r.teacher_id, COALESCE(t.name, ''), r.term_id,
		       r.user_hash,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.moderation_reason, r.created_at, r.updated_at
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE r.id = $1 AND r.status = 'published'
	`, reviewID).Scan(
		&item.ID, &item.CourseID, &item.CourseName, &item.TeacherID, &item.TeacherName,
		&item.TermID, &item.UserHash, &item.Title, &item.Content, &item.Grade, &item.Ratings,
		&item.LikeCount, &item.DislikeCount, &item.ReplyCount,
		&item.Status, &item.ModerationReason, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReviewNotFound
		}
		return nil, fmt.Errorf("GetReviewByID: %w", err)
	}
	return &item, nil
}

func (r *Repository) GetReviewUpdateStateTx(ctx context.Context, tx pgx.Tx, reviewID string) (reviewUpdateState, error) {
	var (
		state     reviewUpdateState
		teacherID int64
	)
	err := tx.QueryRow(ctx, `
		SELECT user_hash, course_id, COALESCE(teacher_id, 0), status, title, content, COALESCE(grade, ''), ratings
		FROM reviews
		WHERE id = $1
		FOR UPDATE
	`, reviewID).Scan(
		&state.userHash,
		&state.courseID,
		&teacherID,
		&state.status,
		&state.title,
		&state.content,
		&state.grade,
		&state.ratings,
	)
	if err != nil {
		return reviewUpdateState{}, err
	}
	if teacherID != 0 {
		state.teacherID = &teacherID
	}
	return state, nil
}

// UpdateParams 更新评论参数
type UpdateParams struct {
	ID          string
	Title       string
	Content     string
	Grade       string
	Ratings     []byte
	Status      string
	ContentFlag *string
}

// Update 更新评论
func (r *Repository) Update(ctx context.Context, tx pgx.Tx, p UpdateParams) error {
	_, err := tx.Exec(ctx, `
		UPDATE reviews SET title = $2, content = $3, grade = $4, ratings = $5,
			avg_rating = COALESCE((SELECT AVG(value::numeric) FROM jsonb_each_text($5) WHERE value ~ '^\d+(\.\d+)?$'), 0),
			status = $6, content_flag = $7, updated_at = NOW()
		WHERE id = $1
	`, p.ID, p.Title, p.Content, p.Grade, p.Ratings, p.Status, p.ContentFlag)
	return err
}

// SoftDeleteReview 软删除评论
func (r *Repository) SoftDeleteReview(ctx context.Context, tx pgx.Tx, reviewID string) error {
	_, err := tx.Exec(ctx, `UPDATE reviews SET status = 'deleted', updated_at = NOW() WHERE id = $1`, reviewID)
	return err
}

// UpdateReviewStatus 更新评论状态
func (r *Repository) UpdateReviewStatus(ctx context.Context, tx pgx.Tx, reviewID, status string) error {
	_, err := tx.Exec(ctx, `UPDATE reviews SET status = $2, updated_at = NOW() WHERE id = $1`, reviewID, status)
	return err
}

// DecrementCourseReviewCount 减少课程评论计数
func (r *Repository) DecrementCourseReviewCount(ctx context.Context, tx pgx.Tx, courseID int64) error {
	_, err := tx.Exec(ctx, `UPDATE courses SET review_count = GREATEST(review_count - 1, 0) WHERE id = $1`, courseID)
	return err
}

// GetReviewCourseIDTx 在事务内获取评论的课程ID
func (r *Repository) GetReviewCourseIDTx(ctx context.Context, tx pgx.Tx, reviewID string) (int64, error) {
	fields, err := getReviewCoreFieldsTx(ctx, tx, reviewID, false)
	if err != nil {
		return 0, err
	}
	return fields.courseID, nil
}

// GetCourseSchoolIDTx 在事务内获取课程所属学校 ID。
func (r *Repository) GetCourseSchoolIDTx(ctx context.Context, tx pgx.Tx, courseID int64) (int64, error) {
	var schoolID int64
	err := tx.QueryRow(ctx, `SELECT school_id FROM courses WHERE id = $1`, courseID).Scan(&schoolID)
	return schoolID, err
}

// GetReviewSchoolIDTx 在事务内获取评论所属学校 ID。
func (r *Repository) GetReviewSchoolIDTx(ctx context.Context, tx pgx.Tx, reviewID string) (int64, error) {
	var schoolID int64
	err := tx.QueryRow(ctx, `SELECT school_id FROM reviews WHERE id = $1`, reviewID).Scan(&schoolID)
	return schoolID, err
}

func (r *Repository) GetReviewSchoolID(ctx context.Context, reviewID string) (int64, error) {
	var schoolID int64
	err := r.db.QueryRow(ctx, `SELECT school_id FROM reviews WHERE id = $1`, reviewID).Scan(&schoolID)
	return schoolID, err
}

// GetReviewStatusAndCourseIDTx 在事务内获取评论状态和课程ID
func (r *Repository) GetReviewStatusAndCourseIDTx(ctx context.Context, tx pgx.Tx, reviewID string) (string, int64, error) {
	fields, err := getReviewCoreFieldsTx(ctx, tx, reviewID, true)
	if err != nil {
		return "", 0, err
	}
	return fields.status, fields.courseID, nil
}

// GetReviewStatusCourseTeacherTx 在事务内获取评论状态、课程ID和教师ID（带行锁）。
func (r *Repository) GetReviewStatusCourseTeacherTx(ctx context.Context, tx pgx.Tx, reviewID string) (string, int64, *int64, error) {
	fields, err := getReviewCoreFieldsTx(ctx, tx, reviewID, true)
	if err != nil {
		return "", 0, nil, err
	}
	return fields.status, fields.courseID, fields.teacherID, nil
}

// GetReviewOwnerCourseTeacherStatusTx 在事务内获取评论所有者、课程ID、教师ID和状态（带行锁）。
func (r *Repository) GetReviewOwnerCourseTeacherStatusTx(ctx context.Context, tx pgx.Tx, reviewID string) (string, int64, *int64, string, error) {
	fields, err := getReviewCoreFieldsTx(ctx, tx, reviewID, true)
	if err != nil {
		return "", 0, nil, "", err
	}
	return fields.userHash, fields.courseID, fields.teacherID, fields.status, nil
}

func (r *Repository) ListReviewSchoolIDs(ctx context.Context, reviewIDs []string) (map[string]int64, error) {
	if len(reviewIDs) == 0 {
		return map[string]int64{}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.school_id
		FROM reviews r
		WHERE r.id = ANY($1)
	`, reviewIDs)
	if err != nil {
		return nil, fmt.Errorf("ListReviewSchoolIDs: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64, len(reviewIDs))
	for rows.Next() {
		var (
			reviewID string
			schoolID int64
		)
		if err := rows.Scan(&reviewID, &schoolID); err != nil {
			return nil, fmt.Errorf("ListReviewSchoolIDs scan: %w", err)
		}
		result[reviewID] = schoolID
	}
	return result, rows.Err()
}

// ListDistinctReviewTargetIDsTx 在事务内获取一批评论关联的课程ID和教师ID集合。
func (r *Repository) ListDistinctReviewTargetIDsTx(ctx context.Context, tx pgx.Tx, reviewIDs []string) ([]int64, []int64, error) {
	if len(reviewIDs) == 0 {
		return nil, nil, nil
	}

	var (
		courseIDs  []int64
		teacherIDs []int64
	)
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(array_agg(DISTINCT course_id), '{}'::bigint[]),
		       COALESCE(array_agg(DISTINCT teacher_id) FILTER (WHERE teacher_id IS NOT NULL), '{}'::bigint[])
		FROM reviews
		WHERE id = ANY($1)
	`, reviewIDs).Scan(&courseIDs, &teacherIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("ListDistinctReviewTargetIDsTx: %w", err)
	}
	return courseIDs, teacherIDs, nil
}

// maxBatchCourseIDs 限制批量查询时允许携带的课程 ID 数量
const maxBatchCourseIDs = 20

// ListByMultipleCourses 批量获取多个课程的测评列表（消除 N+1 查询）
// 返回按 courseID 分组的结果 map
func (r *Repository) ListByMultipleCourses(ctx context.Context, courseIDs []int64, sort string, limit int) (map[int64][]Review, map[int64]int, error) {
	if len(courseIDs) == 0 {
		return nil, nil, nil
	}
	if len(courseIDs) > maxBatchCourseIDs {
		courseIDs = courseIDs[:maxBatchCourseIDs]
	}

	orderBy, ok := allowedSortOrders[sort]
	if !ok {
		orderBy = allowedSortOrders[SortTime]
	}

	// 使用 LATERAL join 高效获取每个课程的 top-N 测评
	rows, err := r.db.Query(ctx, `
		SELECT sub.course_id, sub.id, c.name, sub.teacher_id, t.name, sub.term_id,
		       sub.title, sub.content, sub.grade, sub.ratings,
		       sub.like_count, sub.dislike_count,
		       sub.reply_count,
		       sub.status, sub.moderation_reason, sub.created_at, sub.updated_at,
		       sub.total
		FROM unnest($1::bigint[]) AS cid(id)
		CROSS JOIN LATERAL (
			SELECT r.*, COUNT(*) OVER() AS total
			FROM reviews r
			WHERE r.course_id = cid.id AND r.status = 'published'
			ORDER BY `+orderBy+`
			LIMIT $2
		) sub
		LEFT JOIN courses c ON c.id = sub.course_id
		LEFT JOIN teachers t ON t.id = sub.teacher_id
	`, courseIDs, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("ListByMultipleCourses: %w", err)
	}
	defer rows.Close()

	reviewsMap := make(map[int64][]Review)
	totalsMap := make(map[int64]int)
	for rows.Next() {
		var courseID int64
		var item Review
		var total int
		if err := rows.Scan(
			&courseID,
			&item.ID, &item.CourseName, &item.TeacherID, &item.TeacherName,
			&item.TermID, &item.Title, &item.Content, &item.Grade, &item.Ratings,
			&item.LikeCount, &item.DislikeCount, &item.ReplyCount,
			&item.Status, &item.ModerationReason, &item.CreatedAt, &item.UpdatedAt,
			&total,
		); err != nil {
			return nil, nil, fmt.Errorf("ListByMultipleCourses scan: %w", err)
		}
		item.CourseID = courseID
		reviewsMap[courseID] = append(reviewsMap[courseID], item)
		totalsMap[courseID] = total
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("ListByMultipleCourses rows: %w", err)
	}

	return reviewsMap, totalsMap, nil
}

// allowedSortOrders 排序参数白名单（纵深防御：即使 handler 层已校验，Repository 层也独立校验）
// 列别名约定 — 所有别名均使用 "r." 前缀引用 reviews 表列，
// 与 ListByCourseWithSort / ListLatest 的 SELECT 别名一致。
// 修改此 map 时需同步检查对应查询的 SELECT 列别名。
var allowedSortOrders = map[string]string{
	SortTime:   "r.created_at DESC",
	SortLikes:  "r.like_count DESC, r.created_at DESC",
	SortRating: "r.avg_rating DESC, r.created_at DESC",
}

// ListByCourseWithSortParams 带排序筛选的评论列表参数
type ListByCourseWithSortParams struct {
	CourseID  int64
	Sort      string // time, likes, rating
	TermID    string
	TeacherID *int64
	Limit     int
	Offset    int
}

// ListByCourseWithSort 获取课程评论列表（支持排序和筛选，含总数）。
// 公开查询使用分离的 COUNT + 数据查询，避免 COUNT(*) OVER() 的全量窗口扫描开销。
func (r *Repository) ListByCourseWithSort(ctx context.Context, p ListByCourseWithSortParams) ([]Review, int, error) {
	// 构建公共 WHERE 子句
	baseWhere := ` WHERE r.course_id = $1 AND r.status = 'published'`
	args := []interface{}{p.CourseID}
	argIdx := 2

	if p.TermID != "" {
		baseWhere += ` AND r.term_id = $` + strconv.Itoa(argIdx)
		args = append(args, p.TermID)
		argIdx++
	}
	if p.TeacherID != nil {
		baseWhere += ` AND r.teacher_id = $` + strconv.Itoa(argIdx)
		args = append(args, *p.TeacherID)
		argIdx++
	}

	// 1) COUNT 查询
	countQuery := `SELECT COUNT(*) FROM reviews r` + baseWhere
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args[:argIdx-1]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("ListByCourseWithSort count: %w", err)
	}
	if total == 0 {
		return []Review{}, 0, nil
	}

	// 2) 数据查询
	var qb strings.Builder
	qb.WriteString(`
		SELECT r.id, r.course_id, COALESCE(c.name, ''), r.teacher_id, COALESCE(t.name, ''), r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.moderation_reason, r.created_at, r.updated_at
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
	`)
	qb.WriteString(baseWhere)

	// SQL 注入安全保证：orderBy 仅来自 allowedSortOrders 硬编码 map，不含用户输入
	orderBy, ok := allowedSortOrders[p.Sort]
	if !ok {
		orderBy = allowedSortOrders["time"]
	}
	qb.WriteString(` ORDER BY ` + orderBy)

	qb.WriteString(` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1))
	dataArgs := append(args, p.Limit, p.Offset) //nolint:gocritic // intentional append to new slice

	rows, err := r.db.Query(ctx, qb.String(), dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListByCourseWithSort data: %w", err)
	}
	defer rows.Close()

	list, err := scanReviews(rows)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// SearchReviewsQueryParams 搜索测评查询参数
type SearchReviewsQueryParams struct {
	Query        string
	DepartmentID int64
	TeacherName  string
	TermID       string
	Sort         string
	Limit        int
	Offset       int
}

// SearchReviews 搜索测评列表（支持课程名/课号、院系、教师、学期筛选）。
// 公开查询使用分离的 COUNT + 数据查询，避免 COUNT(*) OVER() 的全量窗口扫描开销。
func (r *Repository) SearchReviews(ctx context.Context, p SearchReviewsQueryParams) ([]Review, int, error) {
	// 构建公共 WHERE 子句片段和参数
	var whereParts strings.Builder
	args := make([]any, 0, 6)
	argIdx := 1
	if p.Query != "" {
		pattern := "%" + httputil.EscapeLikePattern(p.Query) + "%"
		whereParts.WriteString(` AND (c.name ILIKE $` + strconv.Itoa(argIdx) + ` ESCAPE '\' OR c.code ILIKE $` + strconv.Itoa(argIdx) + ` ESCAPE '\')`)
		args = append(args, pattern)
		argIdx++
	}
	if p.DepartmentID > 0 {
		whereParts.WriteString(` AND c.department_id = $` + strconv.Itoa(argIdx))
		args = append(args, p.DepartmentID)
		argIdx++
	}
	if p.TeacherName != "" {
		pattern := "%" + httputil.EscapeLikePattern(p.TeacherName) + "%"
		whereParts.WriteString(` AND t.name ILIKE $` + strconv.Itoa(argIdx) + ` ESCAPE '\'`)
		args = append(args, pattern)
		argIdx++
	}
	if p.TermID != "" {
		whereParts.WriteString(` AND r.term_id = $` + strconv.Itoa(argIdx))
		args = append(args, p.TermID)
		argIdx++
	}

	whereClause := whereParts.String()
	countArgs := make([]any, len(args))
	copy(countArgs, args)

	// 1) COUNT 查询
	countQuery := `SELECT COUNT(*) FROM reviews r LEFT JOIN courses c ON c.id = r.course_id LEFT JOIN teachers t ON t.id = r.teacher_id WHERE r.status = 'published'` + whereClause
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("SearchReviews count: %w", err)
	}
	if total == 0 {
		return []Review{}, 0, nil
	}

	// 2) 数据查询
	var qb strings.Builder
	qb.WriteString(`
		SELECT r.id, r.course_id, COALESCE(c.name, ''), r.teacher_id, COALESCE(t.name, ''), r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.moderation_reason, r.created_at, r.updated_at
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE r.status = 'published'
	`)
	qb.WriteString(whereClause)

	orderBy, ok := allowedSortOrders[p.Sort]
	if !ok {
		orderBy = allowedSortOrders[SortTime]
	}
	qb.WriteString(` ORDER BY ` + orderBy)
	qb.WriteString(` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1))
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.Query(ctx, qb.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("SearchReviews data: %w", err)
	}
	defer rows.Close()

	list, err := scanReviews(rows)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListByUserHash 获取用户的评论列表（含总数）
func (r *Repository) ListByUserHash(ctx context.Context, userHash string, limit, offset int) ([]Review, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.course_id, COALESCE(c.name, ''), r.teacher_id, COALESCE(t.name, ''), r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.moderation_reason, r.created_at, r.updated_at,
		       COUNT(*) OVER() AS total
		FROM reviews r
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE r.user_hash = $1 AND r.status != 'deleted'
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`, userHash, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanReviewsWithTotal(rows)
}

// ListVotedReviews 获取用户点赞/踩的评论列表（含总数）
func (r *Repository) ListVotedReviews(ctx context.Context, userHash, voteType string, limit, offset int) ([]Review, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.course_id, COALESCE(c.name, ''), r.teacher_id, COALESCE(t.name, ''), r.term_id,
		       r.title, r.content, r.grade, r.ratings,
		       r.like_count, r.dislike_count,
		       r.reply_count,
		       r.status, r.moderation_reason, r.created_at, r.updated_at,
		       COUNT(*) OVER() AS total
		FROM reviews r
		JOIN review_votes rv ON rv.review_id = r.id
		LEFT JOIN courses c ON c.id = r.course_id
		LEFT JOIN teachers t ON t.id = r.teacher_id
		WHERE rv.user_hash = $1 AND rv.vote_type = $2 AND r.status = 'published'
		ORDER BY rv.created_at DESC
		LIMIT $3 OFFSET $4
	`, userHash, voteType, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	return scanReviewsWithTotal(rows)
}
