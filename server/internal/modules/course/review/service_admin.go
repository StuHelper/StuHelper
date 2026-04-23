package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/sanitizer"
)

// AdminUpdateReviewParams 管理员更新评论参数
type AdminUpdateReviewParams struct {
	ReviewID string
	Action   string
	Reason   string // 可选，hide 时记录屏蔽原因
	AdminID  string // 可选，hide/restore 时记录操作人
}

// validTransitions 定义合法的状态转移白名单
// key: action, value: 允许的源状态集合
var validTransitions = map[string]map[string]bool{
	"hide":    {StatusPublished: true, StatusPendingReview: true},
	"restore": {StatusHidden: true, StatusPendingReview: true},
	"delete":  {StatusPublished: true, StatusHidden: true, StatusPendingReview: true},
}

// AdminUpdateReviewResult 管理员更新评论结果
type AdminUpdateReviewResult struct {
	OldStatus string // 事务内读取的旧状态，用于审计日志
}

type adminReviewTransition struct {
	courseID  int64
	teacherID *int64
	newStatus string
	oldStatus string
}

// AdminUpdateReview 管理员更新评论，返回事务内读取的旧状态。
// restore 对 pending_review 表示审核通过并发布。
func (s *Service) AdminUpdateReview(ctx context.Context, params AdminUpdateReviewParams) (*AdminUpdateReviewResult, error) {
	transition, err := s.applyAdminUpdateReview(ctx, params)
	if err != nil {
		return nil, err
	}
	return &AdminUpdateReviewResult{OldStatus: transition.oldStatus}, nil
}

func (s *Service) applyAdminUpdateReview(ctx context.Context, params AdminUpdateReviewParams) (*adminReviewTransition, error) {
	var transition *adminReviewTransition
	err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		next, err := s.applyAdminReviewActionTx(ctx, tx, params)
		if err != nil {
			return err
		}
		transition = next
		if isPublicReviewStatus(next.oldStatus) || isPublicReviewStatus(next.newStatus) {
			return s.refreshReviewTargetTx(ctx, tx, next.courseID, next.teacherID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return transition, nil
}

func (s *Service) applyAdminReviewActionTx(
	ctx context.Context,
	tx pgx.Tx,
	params AdminUpdateReviewParams,
) (*adminReviewTransition, error) {
	currentStatus, courseID, teacherID, err := s.repo.GetReviewStatusCourseTeacherTx(ctx, tx, params.ReviewID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReviewNotFound
		}
		return nil, err
	}

	newStatus, err := validateAdminReviewTransition(params.Action, currentStatus)
	if err != nil {
		return nil, err
	}
	if err := s.executeAdminReviewActionTx(ctx, tx, params, currentStatus, courseID, newStatus); err != nil {
		return nil, err
	}
	return &adminReviewTransition{
		courseID:  courseID,
		teacherID: teacherID,
		newStatus: newStatus,
		oldStatus: currentStatus,
	}, nil
}

func validateAdminReviewTransition(action, currentStatus string) (string, error) {
	allowed, ok := validTransitions[action]
	if !ok {
		return "", ErrInvalidAction
	}
	if !allowed[currentStatus] {
		return "", fmt.Errorf("%w: cannot %s from %s", ErrInvalidTransition, action, currentStatus)
	}

	switch action {
	case "hide":
		return StatusHidden, nil
	case "restore":
		return StatusPublished, nil
	case "delete":
		return StatusDeleted, nil
	default:
		return "", ErrInvalidAction
	}
}

func (s *Service) executeAdminReviewActionTx(
	ctx context.Context,
	tx pgx.Tx,
	params AdminUpdateReviewParams,
	currentStatus string,
	courseID int64,
	newStatus string,
) error {
	switch params.Action {
	case "hide":
		return s.hideReviewByAdminTx(ctx, tx, params, currentStatus, courseID)
	case "restore":
		return s.restoreReviewByAdminTx(ctx, tx, params, currentStatus, courseID)
	case "delete":
		return s.deleteReviewByAdminTx(ctx, tx, params.ReviewID, currentStatus, courseID)
	default:
		return ErrInvalidAction
	}
}

func (s *Service) hideReviewByAdminTx(
	ctx context.Context,
	tx pgx.Tx,
	params AdminUpdateReviewParams,
	currentStatus string,
	courseID int64,
) error {
	if err := s.repo.UpdateReviewStatus(ctx, tx, params.ReviewID, StatusHidden); err != nil {
		return err
	}
	if params.AdminID != "" {
		if err := s.repo.ModerateReviewTx(ctx, tx, params.ReviewID, params.Reason, params.AdminID); err != nil {
			return err
		}
	}
	if isPublicReviewStatus(currentStatus) {
		return s.repo.DecrementCourseReviewCount(ctx, tx, courseID)
	}
	return nil
}

func (s *Service) restoreReviewByAdminTx(
	ctx context.Context,
	tx pgx.Tx,
	params AdminUpdateReviewParams,
	currentStatus string,
	courseID int64,
) error {
	if err := s.repo.UpdateReviewStatus(ctx, tx, params.ReviewID, StatusPublished); err != nil {
		return err
	}
	if currentStatus == StatusPendingReview {
		if err := s.repo.ClearContentFlagTx(ctx, tx, params.ReviewID, params.AdminID); err != nil {
			return err
		}
	} else if err := s.repo.ClearModerationTx(ctx, tx, params.ReviewID); err != nil {
		return err
	}
	if !isPublicReviewStatus(currentStatus) {
		return s.repo.IncrementCourseReviewCount(ctx, tx, courseID)
	}
	return nil
}

func (s *Service) deleteReviewByAdminTx(
	ctx context.Context,
	tx pgx.Tx,
	reviewID, currentStatus string,
	courseID int64,
) error {
	if err := s.repo.SoftDeleteReview(ctx, tx, reviewID); err != nil {
		return err
	}
	if isPublicReviewStatus(currentStatus) {
		return s.repo.DecrementCourseReviewCount(ctx, tx, courseID)
	}
	return nil
}

// AdminEditReviewParams 管理员编辑评论内容参数
type AdminEditReviewParams struct {
	ReviewID string
	Title    string
	Content  string
	Reason   string
	AdminID  string
}

// AdminEditReview 管理员编辑评论内容
func (s *Service) AdminEditReview(ctx context.Context, params AdminEditReviewParams) error {
	title := sanitizer.SanitizeTitle(params.Title)
	content := sanitizer.SanitizeText(params.Content)
	if strings.TrimSpace(title) == "" {
		return ErrTitleEmpty
	}
	if sanitizer.ContainsDangerousContent(title) || sanitizer.ContainsDangerousContent(content) {
		return ErrDangerousContent
	}
	if strings.TrimSpace(content) == "" {
		return ErrContentEmpty
	}

	checkResult, err := s.filter.CheckContent(ctx, title+" "+content)
	if err != nil {
		return err
	}

	decision, err := buildReviewModerationDecision(checkResult)
	if err != nil {
		return err
	}

	return s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 确认评论存在
		_, _, err := s.repo.GetReviewStatusAndCourseIDTx(ctx, tx, params.ReviewID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReviewNotFound
			}
			return err
		}
		return s.repo.AdminEditReviewContentTx(ctx, tx, params.ReviewID, title, content, params.Reason, params.AdminID, decision.ContentFlag)
	})
}

// ListAllReviewsParams 获取所有评论参数
type ListAllReviewsParams struct {
	Status    string
	Page      int
	PageSize  int
	SchoolIDs []int64
}

// ListAllReviews 获取所有评论（管理员）
func (s *Service) ListAllReviews(ctx context.Context, params ListAllReviewsParams) (*GetCourseReviewsResult, error) {
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListAllReviews(ctx, params.Status, pageSize, offset, params.SchoolIDs)
	if err != nil {
		return nil, err
	}

	return &GetCourseReviewsResult{List: list, Total: total}, nil
}

// BatchUpdateReviewsParams 批量更新评论参数
type BatchUpdateReviewsParams struct {
	IDs     []string
	Action  string // hide, restore, delete
	AdminID string
}

// BatchUpdateReviewsResult 批量更新评论结果
type BatchUpdateReviewsResult struct {
	Affected int64 `json:"affected"`
}

// BatchUpdateReviews 批量更新评论状态（管理员）
func (s *Service) BatchUpdateReviews(ctx context.Context, params BatchUpdateReviewsParams) (*BatchUpdateReviewsResult, error) {
	// 纵深防御：service 层也校验批量上限，防止绕过 handler 直接调用
	if len(params.IDs) > maxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds limit of %d", len(params.IDs), maxBatchSize)
	}
	if _, ok := validTransitions[params.Action]; !ok {
		return nil, ErrInvalidAction
	}
	ids := dedupeReviewIDs(params.IDs)

	var affected int64
	if err := s.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		courseIDs := make([]int64, 0, len(ids))
		teacherIDs := make([]int64, 0, len(ids))
		for _, reviewID := range ids {
			transition, err := s.applyAdminReviewActionTx(ctx, tx, AdminUpdateReviewParams{
				ReviewID: reviewID,
				Action:   params.Action,
				AdminID:  params.AdminID,
			})
			if err != nil {
				return err
			}
			affected++
			courseIDs = append(courseIDs, transition.courseID)
			if transition.teacherID != nil {
				teacherIDs = append(teacherIDs, *transition.teacherID)
			}
		}
		return s.refreshReviewTargetsTx(ctx, tx, courseIDs, teacherIDs)
	}); err != nil {
		return nil, err
	}

	return &BatchUpdateReviewsResult{Affected: affected}, nil
}

// BatchUpdateReviewsWithAudit 批量更新评论状态并记录审计日志（管理员）
// 封装审计日志记录，供 handler 层调用
func (s *Service) BatchUpdateReviewsWithAudit(ctx context.Context, params BatchUpdateReviewsParams, adminUserID, adminUsername string) (*BatchUpdateReviewsResult, error) {
	if params.AdminID == "" {
		params.AdminID = adminUserID
	}
	result, err := s.BatchUpdateReviews(ctx, params)
	if err != nil {
		return nil, err
	}

	// 记录批量操作审计日志
	audit.Log(audit.EventFromContext(ctx, audit.Event{
		Type:         audit.EventAdminBatchOp,
		ActorType:    "admin",
		UserID:       adminUserID,
		Username:     adminUsername,
		Resource:     "review",
		ResourceType: "review",
		ResourceID:   fmt.Sprintf("batch:%d_items", len(params.IDs)),
		Action:       "batch_" + params.Action,
		Result:       "success",
		Details: map[string]any{
			"ids":      params.IDs,
			"action":   params.Action,
			"affected": result.Affected,
		},
	}))

	return result, nil
}

func dedupeReviewIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

// LogOperationParams 记录操作日志参数
type LogOperationParams struct {
	AdminUserID   string
	AdminUsername string
	Action        string
	ResourceType  string
	ResourceID    string
	OldValue      any
	NewValue      any
	RequestID     string
	TraceID       string
	IPAddress     string
	UserAgent     string
}

// LogOperation 记录操作日志
func (s *Service) LogOperation(ctx context.Context, params LogOperationParams) error {
	var oldValue, newValue []byte
	var err error

	if params.OldValue != nil {
		oldValue, err = json.Marshal(params.OldValue)
		if err != nil {
			return err
		}
	}
	if params.NewValue != nil {
		newValue, err = json.Marshal(params.NewValue)
		if err != nil {
			return err
		}
	}

	return s.repo.CreateOperationLog(ctx, CreateOperationLogParams{
		AdminUserID:   params.AdminUserID,
		AdminUsername: params.AdminUsername,
		Action:        params.Action,
		ResourceType:  params.ResourceType,
		ResourceID:    params.ResourceID,
		OldValue:      oldValue,
		NewValue:      newValue,
		RequestID:     params.RequestID,
		TraceID:       params.TraceID,
		IPAddress:     params.IPAddress,
		UserAgent:     params.UserAgent,
	})
}

// GetOperationLogsParams 获取操作日志参数
type GetOperationLogsParams struct {
	Page     int
	PageSize int
}

// GetOperationLogsResult 获取操作日志结果
type GetOperationLogsResult struct {
	List  []AdminOperationLog
	Total int
}

// GetOperationLogs 获取操作日志列表
func (s *Service) GetOperationLogs(ctx context.Context, params GetOperationLogsParams) (*GetOperationLogsResult, error) {
	pageSize := httputil.ClampPageSize(params.PageSize)
	offset := httputil.SafeOffset(params.Page, pageSize)
	list, total, err := s.repo.ListOperationLogs(ctx, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &GetOperationLogsResult{List: list, Total: total}, nil
}

// StreamExportReviews 流式导出评论，逐行回调
func (s *Service) StreamExportReviews(ctx context.Context, status string, fn func(Review) error) error {
	return s.repo.ForEachReviewForExport(ctx, status, fn)
}

// CleanupOldOperationLogs 清理超过保留期的操作日志。
// 返回实际删除的行数。
func (s *Service) CleanupOldOperationLogs(ctx context.Context, retentionDays int) (int64, error) {
	return s.repo.CleanupOldOperationLogs(ctx, retentionDays)
}
