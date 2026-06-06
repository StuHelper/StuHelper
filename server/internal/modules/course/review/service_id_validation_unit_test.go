package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReviewService_CourseIDInputsRejectInvalidIDsBeforeDependencies(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	_, err := svc.GetCourseReviews(ctx, GetCourseReviewsParams{CourseID: 0})
	require.ErrorIs(t, err, ErrCourseNotFound)

	_, err = svc.GetBatchCourseReviews(ctx, GetBatchCourseReviewsParams{CourseIDs: []int64{1, -1}})
	require.ErrorIs(t, err, ErrCourseNotFound)

	exists, err := svc.CheckCourseExists(ctx, 0)
	require.ErrorIs(t, err, ErrCourseNotFound)
	require.False(t, exists)

	_, err = svc.GetCourseRatingStats(ctx, 0)
	require.ErrorIs(t, err, ErrCourseNotFound)

	_, err = svc.GetCourseTeachers(ctx, -1)
	require.ErrorIs(t, err, ErrCourseNotFound)

	_, err = svc.GetRatingTrend(ctx, 0)
	require.ErrorIs(t, err, ErrCourseNotFound)

	err = svc.AddFavorite(ctx, AddFavoriteParams{UserHash: "u-course-id", CourseID: 0})
	require.ErrorIs(t, err, ErrCourseNotFound)

	err = svc.RemoveFavorite(ctx, "u-course-id", -1)
	require.ErrorIs(t, err, ErrCourseNotFound)

	_, err = svc.GetFavoriteStatus(ctx, "u-course-id", 0)
	require.ErrorIs(t, err, ErrCourseNotFound)

	_, err = svc.SaveDraft(ctx, SaveDraftParams{
		UserHash: "u-course-id",
		CourseID: int64Ptr(0),
	})
	require.ErrorIs(t, err, ErrCourseNotFound)

	_, err = svc.PostReview(ctx, PostReviewParams{
		CourseID:             0,
		UserHash:             "u-course-id",
		AuthorInternalUserID: 1,
		Access:               fullReviewWriteAccess(1),
	})
	require.ErrorIs(t, err, ErrCourseNotFound)
}

func TestReviewService_TeacherIDInputsRejectInvalidIDsBeforeDependencies(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	_, err := svc.GetCourseReviews(ctx, GetCourseReviewsParams{
		CourseID:  1,
		TeacherID: int64Ptr(0),
	})
	require.ErrorIs(t, err, ErrTeacherNotFound)

	_, err = svc.GetTeacherRatingStats(ctx, -1)
	require.ErrorIs(t, err, ErrTeacherNotFound)

	_, err = svc.SaveDraft(ctx, SaveDraftParams{
		UserHash:  "u-teacher-id",
		CourseID:  int64Ptr(1),
		TeacherID: int64Ptr(0),
	})
	require.ErrorIs(t, err, ErrTeacherNotFound)

	_, err = svc.PostReview(ctx, PostReviewParams{
		CourseID:             1,
		TeacherID:            int64Ptr(0),
		UserHash:             "u-teacher-id",
		AuthorInternalUserID: 1,
		Access:               fullReviewWriteAccess(1),
	})
	require.ErrorIs(t, err, ErrTeacherNotFound)

	err = svc.UpdateTeacher(ctx, 0, "teacher", nil)
	require.ErrorIs(t, err, ErrTeacherNotFound)

	err = svc.DeleteTeacher(ctx, -1)
	require.ErrorIs(t, err, ErrTeacherNotFound)
}
