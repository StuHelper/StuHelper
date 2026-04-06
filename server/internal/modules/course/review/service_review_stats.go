package review

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (s *Service) refreshReviewTargetTx(ctx context.Context, tx pgx.Tx, courseID int64, teacherID *int64) error {
	courseIDs := []int64{courseID}
	teacherIDs := make([]int64, 0, 1)
	if teacherID != nil {
		teacherIDs = append(teacherIDs, *teacherID)
	}
	return s.refreshReviewTargetsTx(ctx, tx, courseIDs, teacherIDs)
}

func (s *Service) refreshReviewTargetsTx(ctx context.Context, tx pgx.Tx, courseIDs, teacherIDs []int64) error {
	seenCourses := make(map[int64]struct{}, len(courseIDs))
	for _, courseID := range courseIDs {
		if courseID <= 0 {
			continue
		}
		if _, ok := seenCourses[courseID]; ok {
			continue
		}
		seenCourses[courseID] = struct{}{}
		if err := s.repo.RefreshCourseRatingStatsTx(ctx, tx, courseID); err != nil {
			return err
		}
	}

	seenTeachers := make(map[int64]struct{}, len(teacherIDs))
	for _, teacherID := range teacherIDs {
		if teacherID <= 0 {
			continue
		}
		if _, ok := seenTeachers[teacherID]; ok {
			continue
		}
		seenTeachers[teacherID] = struct{}{}
		if err := s.repo.RefreshTeacherRatingStatsTx(ctx, tx, teacherID); err != nil {
			return err
		}
	}
	return nil
}
