package review

import (
	"context"
	"fmt"
)

// ListVoteTypes returns the current user's votes for a bounded page of reviews.
// It deliberately performs one batch query so list endpoints never fall into
// an N+1 query pattern.
func (r *Repository) ListVoteTypes(
	ctx context.Context,
	userHash string,
	reviewIDs []string,
) (map[string]string, error) {
	result := make(map[string]string, len(reviewIDs))
	if userHash == "" || len(reviewIDs) == 0 {
		return result, nil
	}

	ctx = withDBTable(ctx, "review_votes")
	rows, err := r.db.Query(ctx, `
		SELECT review_id, vote_type
		FROM review_votes
		WHERE user_hash = $1
		  AND review_id = ANY($2::varchar[])
	`, userHash, reviewIDs)
	if err != nil {
		return nil, fmt.Errorf("ListVoteTypes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var reviewID, voteType string
		if err := rows.Scan(&reviewID, &voteType); err != nil {
			return nil, fmt.Errorf("ListVoteTypes scan: %w", err)
		}
		result[reviewID] = voteType
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListVoteTypes rows: %w", err)
	}
	return result, nil
}
