package review

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

// UpsertDraftParams 保存草稿参数
type UpsertDraftParams struct {
	UserHash  string
	CourseID  *int64
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

	newID, err := id.New()
	if err != nil {
		return nil, fmt.Errorf("UpsertDraft generate id: %w", err)
	}

	err = scanDraftRow(r.db.QueryRow(ctx, `
		INSERT INTO review_drafts (id, user_hash, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (user_hash) DO UPDATE SET
			course_id = EXCLUDED.course_id,
			teacher_id = EXCLUDED.teacher_id,
			term_id = EXCLUDED.term_id,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			grade = EXCLUDED.grade,
			ratings = EXCLUDED.ratings,
			updated_at = NOW()
		RETURNING id, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at
	`, newID, p.UserHash, p.CourseID, p.TeacherID, nullableString(p.TermID), p.Title, p.Content, p.Grade, p.Ratings), &d)
	if err != nil {
		return nil, fmt.Errorf("UpsertDraft: %w", err)
	}

	return &d, nil
}

// GetDraft 获取草稿
func (r *Repository) GetDraft(ctx context.Context, userHash string) (*ReviewDraft, error) {
	var d ReviewDraft
	err := scanDraftRow(r.db.QueryRow(ctx, `
		SELECT id, course_id, teacher_id, term_id, title, content, grade, ratings, updated_at
		FROM review_drafts
		WHERE user_hash = $1
	`, userHash), &d)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, fmt.Errorf("GetDraft: %w", err)
	}
	return &d, nil
}

// DeleteDraft 删除草稿
func (r *Repository) DeleteDraft(ctx context.Context, userHash string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM review_drafts WHERE user_hash = $1
	`, userHash)
	if err != nil {
		return fmt.Errorf("DeleteDraft: %w", err)
	}
	return nil
}

type draftScanner interface {
	Scan(dest ...any) error
}

func scanDraftRow(row draftScanner, d *ReviewDraft) error {
	var courseID pgtype.Int8
	var termID pgtype.Text
	if err := row.Scan(
		&d.ID,
		&courseID,
		&d.TeacherID,
		&termID,
		&d.Title,
		&d.Content,
		&d.Grade,
		&d.Ratings,
		&d.UpdatedAt,
	); err != nil {
		return err
	}
	if courseID.Valid {
		value := courseID.Int64
		d.CourseID = &value
	}
	if termID.Valid {
		d.TermID = termID.String
	}
	return nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
