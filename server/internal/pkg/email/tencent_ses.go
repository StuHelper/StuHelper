package email

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

const (
	tencentSESDefaultEndpoint = "ses.tencentcloudapi.com"
	tencentSESDefaultRegion   = "ap-guangzhou"
	tencentSESService         = "ses"
	tencentSESVersion         = "2020-10-02"
	tencentSESActionSendEmail = "SendEmail"
)

type TencentSESConfig struct {
	SecretID       string
	SecretKey      string
	Region         string
	Endpoint       string
	From           string
	FromName       string
	ReplyTo        string
	TemplateID     int64
	DefaultPurpose string
	DefaultSchool  string
	DefaultExpire  int
	Timeout        time.Duration
}

type TencentSESSender struct {
	cfg      TencentSESConfig
	from     mail.Address
	endpoint *url.URL
	host     string
	client   *http.Client
	now      func() time.Time
}

func NewTencentSESSender(cfg TencentSESConfig) (*TencentSESSender, error) {
	cfg.SecretID = strings.TrimSpace(cfg.SecretID)
	cfg.SecretKey = strings.TrimSpace(cfg.SecretKey)
	cfg.Region = strings.TrimSpace(cfg.Region)
	cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
	cfg.From = strings.TrimSpace(cfg.From)
	cfg.FromName = strings.TrimSpace(cfg.FromName)
	cfg.ReplyTo = strings.TrimSpace(cfg.ReplyTo)
	cfg.DefaultPurpose = strings.TrimSpace(cfg.DefaultPurpose)
	cfg.DefaultSchool = strings.TrimSpace(cfg.DefaultSchool)
	if cfg.SecretID == "" {
		return nil, errors.New("tencent ses secret id is required")
	}
	if cfg.SecretKey == "" {
		return nil, errors.New("tencent ses secret key is required")
	}
	if cfg.Region == "" {
		cfg.Region = tencentSESDefaultRegion
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = tencentSESDefaultEndpoint
	}
	if cfg.TemplateID <= 0 {
		return nil, fmt.Errorf("tencent ses template id must be greater than 0: %d", cfg.TemplateID)
	}
	if cfg.DefaultExpire <= 0 {
		cfg.DefaultExpire = 5
	}
	if cfg.DefaultPurpose == "" {
		cfg.DefaultPurpose = "学校邮箱认证"
	}
	if cfg.DefaultSchool == "" {
		cfg.DefaultSchool = "北京航空航天大学"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	from := mail.Address{Name: cfg.FromName, Address: cfg.From}
	if _, err := mail.ParseAddress(from.String()); err != nil {
		return nil, fmt.Errorf("parse sender: %w", err)
	}
	if cfg.ReplyTo != "" {
		if _, err := mail.ParseAddress(cfg.ReplyTo); err != nil {
			return nil, fmt.Errorf("parse reply-to: %w", err)
		}
	}
	endpoint, host, err := normalizeTencentSESEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	return &TencentSESSender{
		cfg:      cfg,
		from:     from,
		endpoint: endpoint,
		host:     host,
		client:   observability.WrapHTTPClient(&http.Client{Timeout: cfg.Timeout}, "tencent_ses"),
		now:      time.Now,
	}, nil
}

func normalizeTencentSESEndpoint(endpoint string) (*url.URL, string, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		trimmed = tencentSESDefaultEndpoint
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, "", fmt.Errorf("parse tencent ses endpoint: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, "", fmt.Errorf("tencent ses endpoint scheme must be http or https: %s", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, "", errors.New("tencent ses endpoint host is required")
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed, parsed.Host, nil
}

func (s *TencentSESSender) SendOTP(
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
		metrics.ObserveExternalRequest("tencent_ses", "send_email", start, err)
	}()
	if s == nil {
		return errors.New("tencent ses sender is nil")
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
	if strings.TrimSpace(purpose) == "" {
		purpose = s.cfg.DefaultPurpose
	}
	if strings.TrimSpace(schoolName) == "" {
		schoolName = s.cfg.DefaultSchool
	}
	if expireMinutes <= 0 {
		expireMinutes = s.cfg.DefaultExpire
	}

	templateData, err := json.Marshal(map[string]string{
		"code":           normalizedCode,
		"expire_minutes": strconv.Itoa(expireMinutes),
		"purpose":        strings.TrimSpace(purpose),
		"school_name":    strings.TrimSpace(schoolName),
	})
	if err != nil {
		return fmt.Errorf("tencent ses marshal template data: %w", err)
	}
	requestBody := tencentSESSendEmailRequest{
		FromEmailAddress: formatTencentSESFrom(s.from),
		Destination:      []string{recipient.Address},
		Template: tencentSESTemplate{
			TemplateID:   s.cfg.TemplateID,
			TemplateData: string(templateData),
		},
		Subject: normalizedSubject,
	}
	if s.cfg.ReplyTo != "" {
		requestBody.ReplyToAddresses = s.cfg.ReplyTo
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("tencent ses marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("tencent ses create request: %w", err)
	}
	now := s.now().UTC()
	timestamp := strconv.FormatInt(now.Unix(), 10)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TC-Action", tencentSESActionSendEmail)
	req.Header.Set("X-TC-Version", tencentSESVersion)
	req.Header.Set("X-TC-Region", s.cfg.Region)
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("Authorization", s.signV3(now, timestamp, payload))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("tencent ses send request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("tencent ses close response body: %w", closeErr)
		}
	}()

	var result tencentSESAPIResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return fmt.Errorf("tencent ses decode response: %w", err)
	}
	if apiErr := result.Response.Error; apiErr != nil {
		return fmt.Errorf("tencent ses API error %s: %s", apiErr.Code, apiErr.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("tencent ses unexpected status: %d", resp.StatusCode)
	}
	return nil
}

func formatTencentSESFrom(from mail.Address) string {
	if strings.TrimSpace(from.Name) == "" {
		return from.Address
	}
	return strings.TrimSpace(from.Name) + " <" + from.Address + ">"
}

func (s *TencentSESSender) signV3(now time.Time, timestamp string, payload []byte) string {
	date := now.UTC().Format("2006-01-02")
	canonicalURI := s.endpoint.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalRequest := fmt.Sprintf(
		"POST\n%s\n\ncontent-type:application/json\nhost:%s\n\ncontent-type;host\n%s",
		canonicalURI,
		s.host,
		sha256Hex(payload),
	)
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, tencentSESService)
	stringToSign := fmt.Sprintf(
		"TC3-HMAC-SHA256\n%s\n%s\n%s",
		timestamp,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	)
	secretDate := hmacSHA256([]byte("TC3"+s.cfg.SecretKey), []byte(date))
	secretService := hmacSHA256(secretDate, []byte(tencentSESService))
	secretSigning := hmacSHA256(secretService, []byte("tc3_request"))
	signature := hex.EncodeToString(hmacSHA256(secretSigning, []byte(stringToSign)))
	return fmt.Sprintf(
		"TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=content-type;host, Signature=%s",
		s.cfg.SecretID,
		credentialScope,
		signature,
	)
}

type tencentSESSendEmailRequest struct {
	FromEmailAddress string             `json:"FromEmailAddress"`
	ReplyToAddresses string             `json:"ReplyToAddresses,omitempty"`
	Destination      []string           `json:"Destination"`
	Template         tencentSESTemplate `json:"Template"`
	Subject          string             `json:"Subject"`
}

type tencentSESTemplate struct {
	TemplateID   int64  `json:"TemplateID"`
	TemplateData string `json:"TemplateData"`
}

type tencentSESAPIResponse struct {
	Response struct {
		Error *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
		RequestID string `json:"RequestId"`
		MessageID string `json:"MessageId"`
	} `json:"Response"`
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
