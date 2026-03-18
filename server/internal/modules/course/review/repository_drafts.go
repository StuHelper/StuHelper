package review

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

// UpsertDraftParams 保存草稿参数
type UpsertDraftParams struct {
	UserHash  string
	CourseID  int64
	TeacherID *int64
	TermID    string
	Title     string
	Content   string
	Grade     string
	Ratings   []byte
}

// UpsertDraft 保存或更新草稿
func (r *Repository) UpsertDraft(ctx context.Context, p UpsertDraftParams) (*ReviewDraft, error) {
	var d ReviewDraft
	var xmax int64

	newID, err := id.New()
	if err != nil {
		return nil, fmt.Errorf("UpsertDraft generate id: %w", err)
	}

	err = r.db.QueryRow(ctx, `
		INSERT INTO review_drafts (id, user_hash, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (user_hash, course_id) DO UPDATE SET
			teacher_id = EXCLUDED.teacher_id,
			term_id = EXCLUDED.term_id,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			grade = EXCLUDED.grade,
			ratings = EXCLUDED.ratings,
			updated_at = NOW()
		RETURNING id, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at, xmax
	`, newID, p.UserHash, p.CourseID, p.TeacherID, p.TermID, p.Title, p.Content, p.Grade, p.Ratings).Scan(
		&d.ID, &d.CourseID, &d.TeacherID, &d.TermID, &d.Title, &d.Content, &d.Grade, &d.Ratings, &d.UpdatedAt, &xmax,
	)
	if err != nil {
		return nil, fmt.Errorf("UpsertDraft: %w", err)
	}

	_ = xmax
	return &d, nil
}

// GetDraft 获取草稿
func (r *Repository) GetDraft(ctx context.Context, userHash string, courseID int64) (*ReviewDraft, error) {
	var d ReviewDraft
	err := r.db.QueryRow(ctx, `
		SELECT id, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at
		FROM review_drafts
		WHERE user_hash = $1 AND course_id = $2
	`, userHash, courseID).Scan(
		&d.ID, &d.CourseID, &d.TeacherID, &d.TermID, &d.Title, &d.Content, &d.Grade, &d.Ratings, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, fmt.Errorf("GetDraft: %w", err)
	}
	return &d, nil
}

// DeleteDraft 删除草稿
func (r *Repository) DeleteDraft(ctx context.Context, userHash string, courseID int64) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM review_drafts WHERE user_hash = $1 AND course_id = $2
	`, userHash, courseID)
	if err != nil {
		return fmt.Errorf("DeleteDraft: %w", err)
	}
	return nil
}
