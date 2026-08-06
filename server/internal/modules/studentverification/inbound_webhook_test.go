package studentverification

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHMACInboundEmailWebhookVerifierAuthenticatesBodyEventAndTimestamp(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	verifier, err := NewHMACInboundEmailWebhookVerifier(secret, 5*time.Minute)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	timestamp := strconv.FormatInt(now.Unix(), 10)
	eventID := "provider-event-12345678"
	body := []byte(`{"subject":"opaque"}`)
	signature := signInboundWebhookTest(secret, timestamp, eventID, body)

	require.NoError(t, verifier.Verify(timestamp, eventID, signature, body, now))
	assert.ErrorIs(t, verifier.Verify(timestamp, eventID, signature, append(body, ' '), now), ErrInboundWebhookUnauthorized)
	assert.ErrorIs(t, verifier.Verify(timestamp, eventID+"-changed", signature, body, now), ErrInboundWebhookUnauthorized)
	assert.ErrorIs(t, verifier.Verify(timestamp, eventID, signature, body, now.Add(6*time.Minute)), ErrInboundWebhookUnauthorized)
}

func TestHMACInboundEmailWebhookVerifierRejectsMalformedInputs(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	verifier, err := NewHMACInboundEmailWebhookVerifier(secret, 5*time.Minute)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	body := []byte(`{"subject":"opaque"}`)

	assert.ErrorIs(t, verifier.Verify("not-a-timestamp", "provider-event", "v1=00", body, now), ErrInboundWebhookUnauthorized)
	assert.ErrorIs(t, verifier.Verify(strconv.FormatInt(now.Unix(), 10), "short", "v1=00", body, now), ErrInboundWebhookUnauthorized)
}

func signInboundWebhookTest(secret []byte, timestamp string, eventID string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(timestamp + "\n" + eventID + "\n"))
	_, _ = mac.Write(body)
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}
