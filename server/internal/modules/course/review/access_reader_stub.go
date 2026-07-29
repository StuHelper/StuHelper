package review

import (
	"context"
	"errors"

	"github.com/StuHelper/StuHelper/server/internal/pkg/reviewaccess"
)

var errReviewAccessReaderNotConfigured = errors.New("review access reader is not configured")

type failClosedReviewAccessReader struct{}

func (failClosedReviewAccessReader) ListReviewAccessSchoolConfigs(context.Context) ([]reviewaccess.SchoolConfig, error) {
	return nil, errReviewAccessReaderNotConfigured
}

func (failClosedReviewAccessReader) ListReviewAccessSystemConfigs(context.Context) ([]reviewaccess.SystemConfig, error) {
	return nil, errReviewAccessReaderNotConfigured
}

func (failClosedReviewAccessReader) GetReviewAccessSubject(context.Context, string) (*reviewaccess.Subject, error) {
	return nil, errReviewAccessReaderNotConfigured
}

func normalizeReviewAccessReader(reader ReviewAccessReader) ReviewAccessReader {
	if reader != nil {
		return reader
	}
	return failClosedReviewAccessReader{}
}
