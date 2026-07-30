package review

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/errs"
	"github.com/StuHelper/StuHelper/server/internal/pkg/response"
)

var (
	reviewModerationErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrTitleEmpty, 400, "title cannot be empty"),
		response.MatchError(ErrTitleTooLong, 400, "title is too long", errs.ErrParamOutOfRange),
		response.MatchError(ErrDangerousContent, 400, "content contains potentially dangerous elements"),
		response.MatchError(ErrSensitiveContent, 400, "content contains sensitive words", errs.ErrSensitiveContent),
		response.MatchError(ErrModerationUnavailable, 503, "content moderation is temporarily unavailable"),
		response.MatchError(ErrContentEmpty, 400, "content cannot be empty", errs.ErrContentEmpty),
		response.MatchError(ErrContentTooShort, 400, "content is too short", errs.ErrParamOutOfRange),
		response.MatchError(ErrContentTooLong, 400, "content is too long", errs.ErrParamOutOfRange),
		response.MatchError(ErrReasonTooLong, 400, "reason is too long", errs.ErrParamOutOfRange),
	}
	reviewCourseLookupErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrCourseNotFound, 404, "course not found", errs.ErrCourseNotFound),
	}
	reviewTeacherLookupErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrTeacherNotFound, 404, "teacher not found", errs.ErrTeacherNotFound),
		{
			Match: func(err error) bool {
				return errors.Is(err, pgx.ErrNoRows)
			},
			Status:  404,
			Code:    errs.ErrTeacherNotFound,
			Message: "teacher not found",
		},
	}
	reviewTeacherAdminErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrTeacherNameInvalid, 400, "teacher name is invalid", errs.ErrInvalidParam),
		response.MatchError(ErrTeacherDepartmentRequired, 400, "teacher department is required", errs.ErrInvalidParam),
		response.MatchError(ErrTeacherDepartmentNotFound, 404, "teacher department not found", errs.ErrTeacherNotFound),
		response.MatchError(ErrTeacherHasReviews, 409, "teacher has associated reviews"),
	}
	reviewUserIdentityErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrUserIdentityRequired, 401, "missing authentication token", errs.ErrTokenMissing),
	}
	reviewWriteAccessErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrReviewWriteAccessDenied, 403, "review write access denied", errs.ErrAccessDenied),
	}
	reviewAdminIdentityErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrAdminIdentityRequired, 401, "missing authentication token", errs.ErrTokenMissing),
	}
	reviewWriteValidationErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrInvalidTermID, 400, "invalid term_id format, expected YYYY-S (e.g. 2024-1)"),
		response.MatchError(ErrRatingRequired, 400, "at least one rating dimension is required"),
		response.MatchError(ErrInvalidRating, 400, "rating must be between 1 and 5"),
		response.MatchError(ErrInvalidGrade, 400, "invalid grade"),
	}
	reviewNotFoundErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrReviewNotFound, 404, "review not found", errs.ErrReviewNotFound),
	}
	reviewReplyOwnerErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrReplyNotFound, 404, "reply not found", errs.ErrReplyNotFound),
		response.MatchError(ErrNotReplyOwner, 403, "you can only delete your own reply", errs.ErrNotReplyOwner),
	}
	reviewReportErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrAlreadyReported, 409, "you have already reported this review", errs.ErrAlreadyReported),
		response.MatchError(ErrReportNotFound, 404, "report not found", errs.ErrReportNotFound),
	}
	reviewAdminActionErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrInvalidAction, 400, "invalid action"),
		response.MatchError(ErrInvalidTransition, 400, "invalid status transition for this action", errs.ErrInvalidTransition),
	}
	reviewSensitiveWordErrorMappings = []response.ErrorMapping{
		response.MatchError(ErrSensitiveWordInvalid, 400, "sensitive word input is invalid", errs.ErrInvalidParam),
		{
			Match: func(err error) bool {
				return errors.Is(err, pgx.ErrNoRows)
			},
			Status:  404,
			Code:    errs.ErrSensitiveWordNotFound,
			Message: "sensitive word not found",
		},
	}
)

func respondToModerationError(c *gin.Context, err error) bool {
	return response.RespondMappedError(c, err, reviewModerationErrorMappings...)
}

func respondPostReviewError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewWriteAccessErrorMappings,
		reviewUserIdentityErrorMappings,
		reviewWriteValidationErrorMappings,
		reviewModerationErrorMappings,
		[]response.ErrorMapping{response.MatchError(ErrAlreadyReviewed, 409, "you have already reviewed this course", errs.ErrReviewExists)},
		reviewCourseLookupErrorMappings,
	)
}

func respondVoteReviewError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewNotFoundErrorMappings,
		reviewUserIdentityErrorMappings,
		[]response.ErrorMapping{
			response.MatchError(ErrInvalidAction, 400, "invalid vote type"),
		},
	)
}

func respondUpdateReviewError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewWriteAccessErrorMappings,
		reviewUserIdentityErrorMappings,
		reviewNotFoundErrorMappings,
		[]response.ErrorMapping{response.MatchError(ErrNotReviewOwner, 403, "you can only edit your own review", errs.ErrNotReviewOwner)},
		reviewWriteValidationErrorMappings,
		reviewModerationErrorMappings,
	)
}

func respondDeleteReviewError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewWriteAccessErrorMappings,
		reviewUserIdentityErrorMappings,
		reviewNotFoundErrorMappings,
		[]response.ErrorMapping{response.MatchError(ErrNotReviewOwner, 403, "you can only delete your own review", errs.ErrNotReviewOwner)},
	)
}

func respondReportReviewError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewUserIdentityErrorMappings,
		reviewNotFoundErrorMappings,
		reviewModerationErrorMappings,
		[]response.ErrorMapping{
			response.MatchError(ErrInvalidAction, 400, "invalid report reason"),
			response.MatchError(ErrAlreadyReported, 409, "you have already reported this review", errs.ErrAlreadyReported),
		},
	)
}

func respondCheckContentError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		[]response.ErrorMapping{response.MatchError(ErrModerationUnavailable, 503, "content moderation is temporarily unavailable")},
	)
}

func respondCreateReplyError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewUserIdentityErrorMappings,
		reviewNotFoundErrorMappings,
		reviewModerationErrorMappings,
	)
}

func respondDeleteReplyError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewUserIdentityErrorMappings,
		reviewReplyOwnerErrorMappings,
	)
}

func respondSaveDraftError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewUserIdentityErrorMappings,
		reviewCourseLookupErrorMappings,
		reviewTeacherLookupErrorMappings,
		reviewWriteValidationErrorMappings,
		reviewModerationErrorMappings,
	)
}

func respondGetDraftError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewUserIdentityErrorMappings,
	)
}

func respondProcessReportError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewAdminIdentityErrorMappings,
		reviewReportErrorMappings,
		reviewNotFoundErrorMappings,
		reviewAdminActionErrorMappings,
	)
}

func respondAdminUpdateReviewError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err, reviewAdminIdentityErrorMappings, reviewNotFoundErrorMappings, reviewAdminActionErrorMappings)
}

func respondAdminEditReviewError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err, reviewAdminIdentityErrorMappings, reviewNotFoundErrorMappings, reviewModerationErrorMappings)
}

func respondTeacherLookupError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err, reviewTeacherLookupErrorMappings)
}

func respondTeacherAdminError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewTeacherAdminErrorMappings,
		reviewTeacherLookupErrorMappings,
	)
}

func respondAddFavoriteError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err,
		reviewUserIdentityErrorMappings,
		reviewCourseLookupErrorMappings,
	)
}

func respondClearContentFlagError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err, reviewAdminIdentityErrorMappings, reviewNotFoundErrorMappings)
}

func respondSensitiveWordAdminError(c *gin.Context, err error) bool {
	return response.RespondMappedErrorGroups(c, err, reviewSensitiveWordErrorMappings)
}
