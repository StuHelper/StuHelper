package review

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sanitizer"
)

// ReportReviewParams 举报评论参数
type ReportReviewParams struct {
	ReviewID    string
	UserHash    string
	Reason      string
	Description string
}

// ListReportsParams 获取举报列表参数
type ListReportsParams struct {
	Status   string
	Page     int
	PageSize int
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
	if strings.TrimSpace(params.Reason) == "" {
		return "", ErrInvalidAction
	}

	if sanitizer.ContainsDangerousContent(params.Description) {
		return "", ErrDangerousContent
	}
	params.Description = sanitizer.SanitizeText(params.Description)

	var reportID string
	err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
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

		reportID, err = s.repo.CreateReportTx(ctx, tx, CreateReportParams{
			ReviewID:     params.ReviewID,
			ReporterHash: params.UserHash,
			Reason:       params.Reason,
			Description:  params.Description,
		})
		return err
	})
	return reportID, err
}

// ListReports 获取举报列表
func (s *Service) ListReports(ctx context.Context, params ListReportsParams) (*ListReportsResult, error) {
	status := params.Status
	if !validReportStatuses[status] {
		status = StatusAll
	}
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListReports(ctx, status, pageSize, offset)
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
