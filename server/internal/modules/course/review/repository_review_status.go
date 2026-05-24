package review

import (
	"context"
	"fmt"
)

func (r *Repository) GetReviewStatus(ctx context.Context, reviewID string) (string, error) {
	ctx = withDBTable(ctx, "reviews")
	var status string
	err := r.db.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1`, reviewID).Scan(&status)
	return status, err
}

func (r *Repository) ListReviewStatuses(ctx context.Context, reviewIDs []string) (map[string]string, error) {
	if len(reviewIDs) == 0 {
		return map[string]string{}, nil
	}
	ctx = withDBTable(ctx, "reviews")

	rows, err := r.db.Query(ctx, `SELECT id, status FROM reviews WHERE id = ANY($1)`, reviewIDs)
	if err != nil {
		return nil, fmt.Errorf("ListReviewStatuses: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string, len(reviewIDs))
	for rows.Next() {
		var reviewID string
		var status string
		if err := rows.Scan(&reviewID, &status); err != nil {
			return nil, fmt.Errorf("ListReviewStatuses scan: %w", err)
		}
		result[reviewID] = status
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListReviewStatuses rows: %w", err)
	}
	return result, nil
}
