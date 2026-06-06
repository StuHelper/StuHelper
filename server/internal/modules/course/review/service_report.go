package review

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sanitizer"
)

const maxReportDescriptionRunes = 500

// ReportReviewParams 举报评论参数
type ReportReviewParams struct {
	ReviewID               string
	UserHash               string
	ReporterInternalUserID int64
	Reason                 string
	Description            string
}

// ListReportsParams 获取举报列表参数
type ListReportsParams struct {
	Status    string
	Page      int
	PageSize  int
	SchoolIDs []int64
}

// ListReportsResult 获取举报列表结果
type ListReportsResult struct {
	List  []ReviewReport
	Total int
}

// ProcessReportParams 处理举报参数
type ProcessReportParams struct {
	ReportID   string
	Action     string
	Note       string
	ResolvedBy string
}

var validReportStatuses = map[string]bool{
	ReportStatusPending:  true,
	ReportStatusResolved: true,
	ReportStatusRejected: true,
	StatusAll:            true,
}

// ReportReview 举报评论，返回生成的举报 ID
func (s *Service) ReportReview(ctx context.Context, params ReportReviewParams) (string, error) {
	var err error
	params.UserHash, err = normalizeRequiredUserHash(params.UserHash)
	if err != nil {
		return "", err
	}
	if params.ReporterInternalUserID <= 0 {
		return "", ErrUserIdentityRequired
	}
	params.Reason = strings.TrimSpace(params.Reason)
	if !isValidReportReason(params.Reason) {
		return "", ErrInvalidAction
	}

	description, err := validateAndSanitizeReportDescription(params.Description)
	if err != nil {
		return "", err
	}
	params.Description = description

	var reportID string
	err = s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		exists, err := s.repo.ReviewExistsTx(ctx, tx, params.ReviewID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrReviewNotFound
		}

		reported, err := s.repo.ReportExistsTx(ctx, tx, params.ReviewID, params.UserHash)
		if err != nil {
			return err
		}
		if reported {
			return ErrAlreadyReported
		}
		schoolID, err := s.repo.GetReviewSchoolIDTx(ctx, tx, params.ReviewID)
		if err != nil {
			return err
		}

		reportID, err = s.repo.CreateReportTx(ctx, tx, CreateReportParams{
			ReviewID:     params.ReviewID,
			SchoolID:     schoolID,
			ReporterHash: params.UserHash,
			Reason:       params.Reason,
			Description:  params.Description,
		})
		if err != nil {
			if isUniqueConstraintViolation(err, "uq_review_reports_user") {
				return ErrAlreadyReported
			}
			return err
		}
		return s.enqueueReportFGASyncTx(ctx, tx, reportID, schoolID)
	})
	return reportID, err
}

func validateAndSanitizeReportDescription(description string) (string, error) {
	if utf8.RuneCountInString(description) > maxReportDescriptionRunes {
		return "", ErrContentTooLong
	}
	if sanitizer.ContainsDangerousContent(description) {
		return "", ErrDangerousContent
	}
	description = sanitizer.SanitizeText(description)
	if sanitizer.ContainsDangerousContent(description) {
		return "", ErrDangerousContent
	}
	return description, nil
}

// ListReports 获取举报列表
func (s *Service) ListReports(ctx context.Context, params ListReportsParams) (*ListReportsResult, error) {
	status := params.Status
	if !validReportStatuses[status] {
		status = StatusAll
	}
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListReports(ctx, status, pageSize, offset, params.SchoolIDs)
	if err != nil {
		return nil, err
	}

	return &ListReportsResult{List: list, Total: total}, nil
}

// ProcessReport 处理举报
func (s *Service) ProcessReport(ctx context.Context, params ProcessReportParams) error {
	switch params.Action {
	case "reject", "hide", "delete":
	default:
		return ErrInvalidAction
	}

	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		report, err := s.repo.GetReportByIDForUpdate(ctx, tx, params.ReportID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReportNotFound
			}
			return err
		}
		if report.Status != ReportStatusPending {
			return fmt.Errorf("%w: cannot %s report from %s", ErrInvalidTransition, params.Action, report.Status)
		}

		var (
			reportStatus string
			refresh      bool
			courseID     int64
			teacherID    *int64
		)
		switch params.Action {
		case "reject":
			reportStatus = ReportStatusRejected
		case "hide":
			reportStatus = ReportStatusResolved
			currentStatus, currentCourseID, currentTeacherID, err := s.repo.GetReviewStatusCourseTeacherTx(ctx, tx, report.ReviewID)
			if err != nil {
				return err
			}
			if err := s.repo.UpdateReviewStatus(ctx, tx, report.ReviewID, StatusHidden); err != nil {
				return err
			}
			if currentStatus == StatusPublished {
				if err := s.repo.DecrementCourseReviewCount(ctx, tx, currentCourseID); err != nil {
					return err
				}
				refresh = true
				courseID = currentCourseID
				teacherID = currentTeacherID
			}
		case "delete":
			reportStatus = ReportStatusResolved
			currentStatus, currentCourseID, currentTeacherID, err := s.repo.GetReviewStatusCourseTeacherTx(ctx, tx, report.ReviewID)
			if err != nil {
				return err
			}
			if err := s.repo.SoftDeleteReview(ctx, tx, report.ReviewID); err != nil {
				return err
			}
			if currentStatus == StatusPublished {
				if err := s.repo.DecrementCourseReviewCount(ctx, tx, currentCourseID); err != nil {
					return err
				}
				refresh = true
				courseID = currentCourseID
				teacherID = currentTeacherID
			}
		}
		if refresh {
			if err := s.refreshReviewTargetTx(ctx, tx, courseID, teacherID); err != nil {
				return err
			}
		}

		return s.repo.UpdateReport(ctx, tx, UpdateReportParams{
			ID:         params.ReportID,
			Status:     reportStatus,
			ResolvedBy: params.ResolvedBy,
			Note:       params.Note,
		})
	})
}
