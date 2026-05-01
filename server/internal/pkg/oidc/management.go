// Package oidc 的 management.go 提供 Zitadel Management API v2 客户端。
// 用于角色同步：学生认证通过后添加 project role，撤销时移除。
package oidc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
)

// ManagementClient Zitadel Management API 客户端
type ManagementClient struct {
	baseURL   string
	pat       string
	projectID string
	orgID     string
	client    *http.Client
}

var ErrManagementPATNotConfigured = errors.New("zitadel management PAT is not configured")

// NewManagementClient 创建 Management API 客户端。
// pat 为空时返回 nil（未配置，角色同步禁用）。
func NewManagementClient(cfg config.CasdoorConfig) *ManagementClient {
	if cfg.ManagementPAT == "" {
		return nil
	}
	return &ManagementClient{
		baseURL:   cfg.Issuer,
		pat:       cfg.ManagementPAT,
		projectID: cfg.ProjectID,
		orgID:     cfg.OrgID,
		client:    newOIDCHTTPClient(cfg, "zitadel_management"),
	}
}

// GrantProjectRole 为用户添加 Project Role Grant。
// userID 是 Zitadel 用户 ID（即 OIDC sub claim）。
func (m *ManagementClient) GrantProjectRole(ctx context.Context, userID, role string) error {
	// Zitadel v2 API: POST /management/v1/users/{userId}/grants
	endpoint := fmt.Sprintf("%s/management/v1/users/%s/grants", m.baseURL, url.PathEscape(userID))
	body := map[string]any{
		"projectId": m.projectID,
		"roleKeys":  []string{role},
	}
	return m.doRequest(ctx, http.MethodPost, endpoint, body)
}

// RevokeProjectRole 移除用户的 Project Role Grant。
// 由于 Zitadel API 需要 grantID，此方法先查询再删除。
func (m *ManagementClient) RevokeProjectRole(ctx context.Context, userID, role string) error {
	// 1. 查询用户的 grants
	grantID, err := m.findUserGrant(ctx, userID, role)
	if err != nil {
		return fmt.Errorf("find grant for revoke: %w", err)
	}
	if grantID == "" {
		return nil // 没有此 grant，无需移除
	}

	// 2. 删除 grant
	endpoint := fmt.Sprintf("%s/management/v1/users/%s/grants/%s", m.baseURL, url.PathEscape(userID), url.PathEscape(grantID))
	return m.doRequest(ctx, http.MethodDelete, endpoint, nil)
}

// findUserGrant 查找用户在当前 project 中指定 role 的 grant ID
func (m *ManagementClient) findUserGrant(ctx context.Context, userID, role string) (_ string, err error) {
	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest("zitadel_management", "find_user_grant", start, err)
	}()
	endpoint := fmt.Sprintf("%s/management/v1/users/%s/grants/_search", m.baseURL, url.PathEscape(userID))
	body := map[string]any{
		"queries": []map[string]any{
			{"projectIdQuery": map[string]string{"projectId": m.projectID}},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	m.setHeaders(req)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close grant search response body: %w", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return "", fmt.Errorf("search grants: read error response: %w", readErr)
		}
		return "", fmt.Errorf("search grants: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result []struct {
			ID       string   `json:"id"`
			RoleKeys []string `json:"roleKeys"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	for _, grant := range result.Result {
		for _, rk := range grant.RoleKeys {
			if rk == role {
				return grant.ID, nil
			}
		}
	}
	return "", nil
}

func (m *ManagementClient) doRequest(ctx context.Context, method, endpoint string, body any) (err error) {
	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest("zitadel_management", strings.ToLower(method), start, err)
	}()
	var reqBody io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	m.setHeaders(req)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode >= 300 {
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return fmt.Errorf("read error response: %w", readErr)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (m *ManagementClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.pat)
	if m.orgID != "" {
		req.Header.Set("x-zitadel-orgid", m.orgID)
	}
}

// BuildRoleSyncFunc 创建角色同步回调函数，适配 user.RoleSyncFunc 签名。
// mgmt 为 nil 时返回仅记日志的 stub。
func BuildRoleSyncFunc(mgmt *ManagementClient, getUserExternalID func(ctx context.Context, userID int64) (string, error)) func(ctx context.Context, userID int64, role string, approved bool) error {
	if mgmt == nil {
		return func(_ context.Context, userID int64, role string, approved bool) error {
			action := "grant"
			if !approved {
				action = "revoke"
			}
			logger.L().Info("role sync skipped (Zitadel Management PAT not configured)",
				zap.Int64("user_id", userID),
				zap.String("role", role),
				zap.String("action", action),
			)
			return fmt.Errorf("%w: role=%s action=%s user_id=%d", ErrManagementPATNotConfigured, role, action, userID)
		}
	}

	return func(ctx context.Context, userID int64, role string, approved bool) error {
		externalID, err := getUserExternalID(ctx, userID)
		if err != nil {
			return fmt.Errorf("get external ID: %w", err)
		}

		if approved {
			err = mgmt.GrantProjectRole(ctx, externalID, role)
		} else {
			err = mgmt.RevokeProjectRole(ctx, externalID, role)
		}
		if err != nil {
			return fmt.Errorf("sync role via Zitadel management: %w", err)
		}

		logger.L().Info("role sync completed",
			zap.Int64("user_id", userID),
			zap.String("role", role),
			zap.Bool("approved", approved),
		)
		return nil
	}
}
