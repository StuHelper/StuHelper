package email

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/observability"
)

const resendDefaultEndpoint = "https://api.resend.com/emails"

type ResendConfig struct {
	APIKey    string
	Endpoint  string
	From      string
	FromName  string
	ReplyTo   string
	Timeout   time.Duration
	UserAgent string
}

type ResendSender struct {
	cfg      ResendConfig
	from     mail.Address
	endpoint *url.URL
	client   *http.Client
}

func NewResendSender(cfg ResendConfig) (*ResendSender, error) {
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
	cfg.From = strings.TrimSpace(cfg.From)
	cfg.FromName = strings.TrimSpace(cfg.FromName)
	cfg.ReplyTo = strings.TrimSpace(cfg.ReplyTo)
	cfg.UserAgent = strings.TrimSpace(cfg.UserAgent)
	if cfg.APIKey == "" {
		return nil, errors.New("resend api key is required")
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = resendDefaultEndpoint
	}
	if cfg.From == "" {
		return nil, errors.New("email from is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "StuHelper"
	}
	from := mail.Address{Name: cfg.FromName, Address: cfg.From}
	if _, err := mail.ParseAddress(from.String()); err != nil {
		return nil, fmt.Errorf("parse resend sender: %w", err)
	}
	if cfg.ReplyTo != "" {
		if _, err := mail.ParseAddress(cfg.ReplyTo); err != nil {
			return nil, fmt.Errorf("parse resend reply-to: %w", err)
		}
	}
	endpoint, err := normalizeResendEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	return &ResendSender{
		cfg:      cfg,
		from:     from,
		endpoint: endpoint,
		client:   observability.WrapHTTPClient(&http.Client{Timeout: cfg.Timeout}, "resend"),
	}, nil
}

func normalizeResendEndpoint(endpoint string) (*url.URL, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		trimmed = resendDefaultEndpoint
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse resend endpoint: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("resend endpoint scheme must be http or https: %s", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, errors.New("resend endpoint host is required")
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/emails"
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed, nil
}

func (s *ResendSender) SendOTP(
	ctx context.Context,
	to string,
	subject string,
	code string,
	purpose string,
	schoolName string,
	expireMinutes int,
) (err error) {
	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest("resend", "send_email", start, err)
	}()
	if s == nil {
		return errors.New("resend sender is nil")
	}
	recipient, err := mail.ParseAddress(strings.TrimSpace(to))
	if err != nil {
		return fmt.Errorf("parse recipient: %w", err)
	}
	normalizedSubject := strings.TrimSpace(subject)
	if normalizedSubject == "" {
		return errors.New("email subject is required")
	}
	normalizedCode := strings.TrimSpace(code)
	if normalizedCode == "" {
		return errors.New("email otp code is required")
	}
	normalizedPurpose := strings.TrimSpace(purpose)
	if normalizedPurpose == "" {
		normalizedPurpose = "学校邮箱认证"
	}
	normalizedSchool := strings.TrimSpace(schoolName)
	if normalizedSchool == "" {
		normalizedSchool = "北京航空航天大学"
	}
	if expireMinutes <= 0 {
		expireMinutes = 5
	}

	requestBody := resendSendEmailRequest{
		From:    formatTencentSESFrom(s.from),
		To:      []string{recipient.Address},
		Subject: normalizedSubject,
		HTML:    buildOTPHTML(normalizedCode, normalizedPurpose, normalizedSchool, expireMinutes),
		Text:    buildOTPText(normalizedCode, normalizedPurpose, normalizedSchool, expireMinutes),
	}
	if s.cfg.ReplyTo != "" {
		requestBody.ReplyTo = []string{s.cfg.ReplyTo}
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("resend marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("resend create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", otpIdempotencyKey(recipient.Address, normalizedSubject, normalizedCode, normalizedPurpose, normalizedSchool))
	req.Header.Set("User-Agent", s.cfg.UserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend send request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("resend close response body: %w", closeErr)
		}
	}()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return fmt.Errorf("resend read response: %w", readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr resendErrorResponse
		if json.Unmarshal(body, &apiErr) == nil && strings.TrimSpace(apiErr.Message) != "" {
			return fmt.Errorf("resend API error %s: %s", firstNonEmpty(apiErr.Name, strconv.Itoa(apiErr.StatusCode)), apiErr.Message)
		}
		return fmt.Errorf("resend unexpected status: %d", resp.StatusCode)
	}
	var result resendSendEmailResponse
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("resend decode response: %w", err)
		}
	}
	if strings.TrimSpace(result.ID) == "" {
		return errors.New("resend response missing email id")
	}
	return nil
}

type resendSendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text,omitempty"`
	ReplyTo []string `json:"reply_to,omitempty"`
}

type resendSendEmailResponse struct {
	ID string `json:"id"`
}

type resendErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Name       string `json:"name"`
	Message    string `json:"message"`
}

func buildOTPHTML(code string, purpose string, schoolName string, expireMinutes int) string {
	escapedCode := html.EscapeString(code)
	escapedPurpose := html.EscapeString(purpose)
	escapedSchool := html.EscapeString(schoolName)
	escapedExpire := html.EscapeString(strconv.Itoa(expireMinutes))
	return strings.NewReplacer(
		"{{code}}", escapedCode,
		"{{purpose}}", escapedPurpose,
		"{{school_name}}", escapedSchool,
		"{{expire_minutes}}", escapedExpire,
	).Replace(`<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="x-apple-disable-message-reformatting">
    <title>学生认证验证码</title>
  </head>
  <body style="margin:0;padding:0;background:#f5f7fb;color:#172033;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','PingFang SC','Microsoft YaHei',Arial,sans-serif;">
    <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;background:#f5f7fb;margin:0;padding:24px 12px;">
      <tr>
        <td align="center">
          <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;max-width:560px;background:#ffffff;border:1px solid #e3e8f2;border-radius:8px;overflow:hidden;">
            <tr>
              <td style="padding:28px 32px 16px 32px;border-bottom:1px solid #edf1f7;">
                <div style="font-size:20px;line-height:28px;font-weight:700;color:#172033;">StuHelper</div>
                <div style="margin-top:6px;font-size:14px;line-height:22px;color:#667085;">{{school_name}} 学校邮箱认证</div>
              </td>
            </tr>
            <tr>
              <td style="padding:28px 32px 8px 32px;">
                <div style="font-size:18px;line-height:28px;font-weight:700;color:#172033;">你的验证码</div>
                <div style="margin-top:16px;padding:18px 20px;background:#f0f6ff;border:1px solid #b9d7ff;border-radius:8px;text-align:center;">
                  <span style="font-size:32px;line-height:40px;font-weight:800;letter-spacing:6px;color:#175cd3;font-family:Consolas,'SFMono-Regular','Roboto Mono',Menlo,monospace;">{{code}}</span>
                </div>
                <p style="margin:18px 0 0 0;font-size:15px;line-height:24px;color:#344054;">
                  你正在进行 <strong>{{purpose}}</strong>。请在 <strong>{{expire_minutes}}</strong> 分钟内回到 StuHelper 页面填写验证码。
                </p>
                <p style="margin:12px 0 0 0;font-size:14px;line-height:22px;color:#667085;">
                  为保护账号安全，请不要把验证码转发或告知他人。StuHelper 工作人员不会向你索要验证码。
                </p>
              </td>
            </tr>
            <tr>
              <td style="padding:16px 32px 28px 32px;">
                <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;background:#fff8eb;border:1px solid #fedf89;border-radius:8px;">
                  <tr>
                    <td style="padding:14px 16px;font-size:13px;line-height:20px;color:#7a4b00;">
                      如果这不是你本人发起的操作，请忽略本邮件。未填写验证码不会完成任何认证或绑定。
                    </td>
                  </tr>
                </table>
              </td>
            </tr>
            <tr>
              <td style="padding:18px 32px;background:#f8fafc;border-top:1px solid #edf1f7;font-size:12px;line-height:20px;color:#667085;">
                本邮件由 StuHelper 账号与认证系统自动发送，用于学校邮箱验证码校验。请勿直接回复本邮件。
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>
`)
}

func buildOTPText(code string, purpose string, schoolName string, expireMinutes int) string {
	return fmt.Sprintf(`学生认证验证码

学校：%s
用途：%s

你的验证码是：%s

请在 %d 分钟内回到 StuHelper 页面填写验证码。

为保护账号安全，请不要把验证码转发或告知他人。StuHelper 工作人员不会向你索要验证码。

如果这不是你本人发起的操作，请忽略本邮件。未填写验证码不会完成任何认证或绑定。

本邮件由 StuHelper 账号与认证系统自动发送，请勿直接回复。
`,
		schoolName,
		purpose,
		code,
		expireMinutes,
	)
}

func otpIdempotencyKey(to string, subject string, code string, purpose string, schoolName string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.ToLower(strings.TrimSpace(to)),
		strings.TrimSpace(subject),
		strings.TrimSpace(code),
		strings.TrimSpace(purpose),
		strings.TrimSpace(schoolName),
	}, "\x00")))
	return "student-email-otp-" + hex.EncodeToString(sum[:])[:32]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "Unknown"
}
