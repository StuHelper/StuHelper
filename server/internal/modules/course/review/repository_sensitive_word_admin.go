package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

// ListSensitiveWords 分页获取敏感词列表
func (r *Repository) ListSensitiveWords(ctx context.Context, category, level string, limit, offset int) ([]SensitiveWord, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, category)
		argIdx++
	}
	if level != "" {
		conditions = append(conditions, fmt.Sprintf("level = $%d", argIdx))
		args = append(args, level)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, word, category, level, is_active, created_at,
		       COUNT(*) OVER() AS total
		FROM sensitive_words
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListSensitiveWords: %w", err)
	}
	defer rows.Close()

	var list []SensitiveWord
	var total int
	for rows.Next() {
		var w SensitiveWord
		if err := rows.Scan(&w.ID, &w.Word, &w.Category, &w.Level, &w.IsActive, &w.CreatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("ListSensitiveWords scan: %w", err)
		}
		list = append(list, w)
	}
	return list, total, rows.Err()
}

// CreateSensitiveWord 新增敏感词
func (r *Repository) CreateSensitiveWord(ctx context.Context, word, category, level string) (SensitiveWord, error) {
	newID, err := id.New()
	if err != nil {
		return SensitiveWord{}, fmt.Errorf("CreateSensitiveWord id: %w", err)
	}
	if category == "" {
		category = "general"
	}
	if level == "" {
		level = "block"
	}

	var w SensitiveWord
	err = r.db.QueryRow(ctx, `
		INSERT INTO sensitive_words (id, word, category, level)
		VALUES ($1, $2, $3, $4)
		RETURNING id, word, category, level, is_active, created_at
	`, newID, word, category, level).Scan(&w.ID, &w.Word, &w.Category, &w.Level, &w.IsActive, &w.CreatedAt)
	if err != nil {
		return SensitiveWord{}, fmt.Errorf("CreateSensitiveWord: %w", err)
	}
	return w, nil
}

// UpdateSensitiveWord 更新敏感词
func (r *Repository) UpdateSensitiveWord(ctx context.Context, wordID string, word, category, level *string, isActive *bool) error {
	var sets []string
	var args []interface{}
	argIdx := 1

	if word != nil {
		sets = append(sets, fmt.Sprintf("word = $%d", argIdx))
		args = append(args, *word)
		argIdx++
	}
	if category != nil {
		sets = append(sets, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *category)
		argIdx++
	}
	if level != nil {
		sets = append(sets, fmt.Sprintf("level = $%d", argIdx))
		args = append(args, *level)
		argIdx++
	}
	if isActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *isActive)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE sensitive_words SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	args = append(args, wordID)

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("UpdateSensitiveWord: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteSensitiveWord 删除敏感词
func (r *Repository) DeleteSensitiveWord(ctx context.Context, wordID string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM sensitive_words WHERE id = $1", wordID)
	if err != nil {
		return fmt.Errorf("DeleteSensitiveWord: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
