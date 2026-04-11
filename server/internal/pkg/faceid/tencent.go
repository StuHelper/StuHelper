package faceid

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

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/metrics"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/observability"
)

const (
	hostTencentFaceID = "faceid.tencentcloudapi.com"
)

type Config struct {
	SecretID  string
	SecretKey string
}

type Client struct {
	cfg    Config
	client *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:    cfg,
		client: observability.WrapHTTPClient(&http.Client{Timeout: 10 * time.Second}, "tencent_faceid"),
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.cfg.SecretID != "" && c.cfg.SecretKey != ""
}

func (c *Client) VerifyMainlandID(ctx context.Context, idCard, name string) (matched bool, code, description string, err error) {
	start := time.Now()
	defer func() {
		metrics.ObserveExternalRequest("tencent_faceid", "id_card_verification", start, err)
	}()

	if !c.Enabled() {
		err = fmt.Errorf("faceid: Tencent Cloud credentials not configured")
		return false, "", "", err
	}

	now := time.Now().UTC()
	timestamp := fmt.Sprintf("%d", now.Unix())
	payload, err := json.Marshal(map[string]string{
		"IdCard": strings.TrimSpace(idCard),
		"Name":   strings.TrimSpace(name),
	})
	if err != nil {
		return false, "", "", fmt.Errorf("faceid: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+hostTencentFaceID, strings.NewReader(string(payload)))
	if err != nil {
		return false, "", "", fmt.Errorf("faceid: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TC-Action", "IdCardVerification")
	req.Header.Set("X-TC-Version", "2018-03-01")
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("Authorization", c.signV3(now, timestamp, payload))

	resp, err := c.client.Do(req)
	if err != nil {
		return false, "", "", fmt.Errorf("faceid: send request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("faceid: close response body: %w", closeErr)
		}
	}()

	var result struct {
		Response struct {
			Result      string `json:"Result"`
			Description string `json:"Description"`
			RequestID   string `json:"RequestId"`
			Error       *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return false, "", "", fmt.Errorf("faceid: decode response: %w", err)
	}
	if result.Response.Error != nil {
		return false, "", "", fmt.Errorf("faceid: API error %s: %s", result.Response.Error.Code, result.Response.Error.Message)
	}

	return result.Response.Result == "0", result.Response.Result, result.Response.Description, nil
}

func (c *Client) signV3(now time.Time, timestamp string, payload []byte) string {
	date := now.UTC().Format("2006-01-02")
	service := "faceid"

	canonicalRequest := fmt.Sprintf("POST\n/\n\ncontent-type:application/json\nhost:%s\n\ncontent-type;host\n%s",
		hostTencentFaceID, sha256Hex(payload))
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	stringToSign := fmt.Sprintf("TC3-HMAC-SHA256\n%s\n%s\n%s",
		timestamp, credentialScope, sha256Hex([]byte(canonicalRequest)))

	secretDate := hmacSHA256([]byte("TC3"+c.cfg.SecretKey), []byte(date))
	secretService := hmacSHA256(secretDate, []byte(service))
	secretSigning := hmacSHA256(secretService, []byte("tc3_request"))
	signature := hex.EncodeToString(hmacSHA256(secretSigning, []byte(stringToSign)))

	return fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=content-type;host, Signature=%s",
		c.cfg.SecretID, credentialScope, signature)
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
