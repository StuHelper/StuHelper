package review

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Service) GetReviewStatus(ctx context.Context, reviewID string) (string, error) {
	status, err := s.repo.GetReviewStatus(ctx, reviewID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrReviewNotFound
	}
	return status, err
}

func (s *Service) ListReviewStatuses(ctx context.Context, reviewIDs []string) (map[string]string, error) {
	return s.repo.ListReviewStatuses(ctx, reviewIDs)
}
