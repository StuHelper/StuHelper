package studentverification

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

var ErrInboundWebhookUnauthorized = errors.New("inbound email webhook unauthorized")

type InboundEmailWebhookVerifier interface {
	Verify(timestamp string, eventID string, signature string, body []byte, now time.Time) error
}

type HMACInboundEmailWebhookVerifier struct {
	secret  []byte
	maxSkew time.Duration
}

func NewHMACInboundEmailWebhookVerifier(
	secret []byte,
	maxSkew time.Duration,
) (*HMACInboundEmailWebhookVerifier, error) {
	if len(secret) < 32 || maxSkew < time.Minute || maxSkew > 30*time.Minute {
		return nil, ErrInboundWebhookUnauthorized
	}
	return &HMACInboundEmailWebhookVerifier{
		secret: append([]byte(nil), secret...), maxSkew: maxSkew,
	}, nil
}

func (v *HMACInboundEmailWebhookVerifier) Verify(
	timestamp string,
	eventID string,
	signature string,
	body []byte,
	now time.Time,
) error {
	if v == nil || len(v.secret) == 0 || len(body) == 0 || len(body) > 64*1024 ||
		len(eventID) < 8 || len(eventID) > 500 || strings.ContainsAny(eventID, "\r\n") {
		return ErrInboundWebhookUnauthorized
	}
	unixSeconds, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return ErrInboundWebhookUnauthorized
	}
	signedAt := time.Unix(unixSeconds, 0).UTC()
	delta := now.Sub(signedAt)
	if delta < 0 {
		delta = -delta
	}
	if delta > v.maxSkew {
		return ErrInboundWebhookUnauthorized
	}
	provided := strings.TrimSpace(signature)
	if !strings.HasPrefix(provided, "v1=") {
		return ErrInboundWebhookUnauthorized
	}
	providedBytes, err := hex.DecodeString(strings.TrimPrefix(provided, "v1="))
	if err != nil || len(providedBytes) != sha256.Size {
		return ErrInboundWebhookUnauthorized
	}
	mac := hmac.New(sha256.New, v.secret)
	_, _ = mac.Write([]byte(strings.TrimSpace(timestamp)))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(eventID))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write(body)
	if !hmac.Equal(providedBytes, mac.Sum(nil)) {
		return ErrInboundWebhookUnauthorized
	}
	return nil
}
