// Package sms 提供短信发送能力，当前支持腾讯云短信 API。
//
// 用途：Zitadel 通用 HTTP SMS 提供商 → 本服务 → 腾讯云短信 API。
// Zitadel 配置方式：Action → 发送 HTTP 请求到本服务的 /internal/sms/send 端点。
package sms

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/observability"
)

// Config 腾讯云短信配置
type Config struct {
	SecretID    string // API SecretId
	SecretKey   string // API SecretKey
	AppID       string // 短信应用 SDKAppID
	SignName    string // 短信签名内容
	TemplateID  string // 短信模板 ID（验证码模板）
	Region      string // 地域，默认 ap-beijing
	InternalKey string // 内部调用鉴权密钥（防外部调用）
}

// Service 短信发送服务
type Service struct {
	cfg    Config
	client *http.Client
	logger *zap.Logger
}

// NewService 创建短信服务
func NewService(cfg Config, logger *zap.Logger) *Service {
	if cfg.Region == "" {
		cfg.Region = "ap-beijing"
	}
	return &Service{
		cfg:    cfg,
		client: observability.WrapHTTPClient(&http.Client{Timeout: 10 * time.Second}, "tencent_sms"),
		logger: logger,
	}
}

// SendRequest 内部 SMS 发送请求（Zitadel Action 调用）
type SendRequest struct {
	Phone   string `json:"phone"`   // 国际格式：+8613800138000
	Content string `json:"content"` // 验证码内容
}

// SendResponse 发送结果
type SendResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Send 发送短信验证码
func (s *Service) Send(ctx context.Context, phone, content string) (err error) {
	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest("tencent_sms", "send_sms", start, err)
	}()
	if s.cfg.SecretID == "" || s.cfg.SecretKey == "" {
		return fmt.Errorf("sms: Tencent Cloud credentials not configured")
	}

	// 腾讯云短信 API v3 请求
	// https://cloud.tencent.com/document/product/382/55981
	now := time.Now().UTC()
	timestamp := fmt.Sprintf("%d", now.Unix())
	params := map[string]any{
		"PhoneNumberSet":   []string{phone},
		"SmsSdkAppId":      s.cfg.AppID,
		"SignName":         s.cfg.SignName,
		"TemplateId":       s.cfg.TemplateID,
		"TemplateParamSet": []string{content},
	}
	payload, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("sms: marshal request: %w", err)
	}

	host := "sms.tencentcloudapi.com"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+host, strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("sms: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TC-Action", "SendSms")
	req.Header.Set("X-TC-Version", "2021-01-11")
	req.Header.Set("X-TC-Region", s.cfg.Region)
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("Authorization", s.signV3(host, now, timestamp, payload))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sms: send request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Response struct {
			SendStatusSet []struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"SendStatusSet"`
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return fmt.Errorf("sms: decode response: %w", err)
	}

	if e := result.Response.Error; e != nil {
		return fmt.Errorf("sms: API error %s: %s", e.Code, e.Message)
	}
	if len(result.Response.SendStatusSet) > 0 {
		status := result.Response.SendStatusSet[0]
		if status.Code != "Ok" {
			return fmt.Errorf("sms: send failed %s: %s", status.Code, status.Message)
		}
	}

	s.logger.Info("SMS sent successfully", zap.String("phone", maskPhone(phone)))
	return nil
}

// signV3 生成腾讯云 API v3 签名（TC3-HMAC-SHA256）
func (s *Service) signV3(host string, now time.Time, timestamp string, payload []byte) string {
	date := now.UTC().Format("2006-01-02")
	service := "sms"

	// 1. 拼接规范请求串
	canonicalRequest := fmt.Sprintf("POST\n/\n\ncontent-type:application/json\nhost:%s\n\ncontent-type;host\n%s",
		host, sha256Hex(payload))

	// 2. 拼接待签名字符串
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	stringToSign := fmt.Sprintf("TC3-HMAC-SHA256\n%s\n%s\n%s",
		timestamp, credentialScope, sha256Hex([]byte(canonicalRequest)))

	// 3. 计算签名
	secretDate := hmacSHA256([]byte("TC3"+s.cfg.SecretKey), []byte(date))
	secretService := hmacSHA256(secretDate, []byte(service))
	secretSigning := hmacSHA256(secretService, []byte("tc3_request"))
	signature := hex.EncodeToString(hmacSHA256(secretSigning, []byte(stringToSign)))

	return fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=content-type;host, Signature=%s",
		s.cfg.SecretID, credentialScope, signature)
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func maskPhone(phone string) string {
	if len(phone) <= 7 {
		return "***"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}
