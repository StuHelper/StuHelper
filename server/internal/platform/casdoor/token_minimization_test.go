package casdoor

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeApplicationSpecTokenMinimizationAllowsOIDCProfileClaims(t *testing.T) {
	spec := validApplicationSpec()
	spec.TokenFields = []string{"sub", "iss", "aud", "preferred_username", "name", "picture", "email", "auth_time"}

	result, err := ProbeApplicationSpecTokenMinimization(spec)

	require.NoError(t, err)
	assert.Empty(t, result.BusinessClaims)
	assert.Contains(t, result.InspectedClaims, "preferred_username")
}

func TestProbeApplicationSpecTokenMinimizationRejectsBusinessClaims(t *testing.T) {
	spec := validApplicationSpec()
	spec.TokenFields = []string{"sub", "phone", "studentVerified", "school.name"}

	result, err := ProbeApplicationSpecTokenMinimization(spec)

	require.ErrorIs(t, err, ErrTokenMinimizationProbeFailed)
	var minimizationErr TokenMinimizationError
	require.True(t, errors.As(err, &minimizationErr))
	assert.Equal(t, []string{"phone", "school_name", "student_verified"}, result.BusinessClaims)
	assert.Equal(t, []string{"phone", "school_name", "student_verified"}, minimizationErr.Claims)
}

func TestProbeApplicationSpecTokenMinimizationRequiresExplicitTokenFields(t *testing.T) {
	spec := validApplicationSpec()
	spec.TokenFields = nil

	_, err := ProbeApplicationSpecTokenMinimization(spec)

	require.ErrorIs(t, err, ErrTokenMinimizationProbeFailed)
	assert.Contains(t, err.Error(), "token fields must be explicit")
}

func TestProbeApplicationSpecTokenMinimizationRejectsNonCustomTokenFormat(t *testing.T) {
	spec := validApplicationSpec()
	spec.TokenFormat = "JWT"
	spec.TokenFields = []string{}

	_, err := ProbeApplicationSpecTokenMinimization(spec)

	require.ErrorIs(t, err, ErrTokenMinimizationProbeFailed)
	assert.Contains(t, err.Error(), "token format must be JWT-Custom")
}

func TestProbeJWTClaimsMinimizedRejectsRuntimeBusinessClaim(t *testing.T) {
	raw := unsignedJWT(t, map[string]any{
		"iss":                        "https://sso.example.com",
		"sub":                        "user-1",
		"stuhelper_student_verified": true,
	})

	result, err := ProbeJWTClaimsMinimized(raw)

	require.ErrorIs(t, err, ErrTokenMinimizationProbeFailed)
	assert.Equal(t, []string{"stuhelper_student_verified"}, result.BusinessClaims)
}

func TestNormalizeRuntimeTokenMinimizationProbeResultRejectsBusinessClaim(t *testing.T) {
	result, err := NormalizeRuntimeTokenMinimizationProbeResult(RuntimeTokenMinimizationProbeResult{
		TokenClaims: map[string][]string{
			"id_token": {"iss", "sub", "phoneVerified", "passwordSalt"},
		},
	})

	require.ErrorIs(t, err, ErrTokenMinimizationProbeFailed)
	var minimizationErr TokenMinimizationError
	require.True(t, errors.As(err, &minimizationErr))
	assert.Equal(t, []string{"password_salt", "phone_verified"}, result.BusinessClaims)
	assert.Equal(t, []string{"iss", "password_salt", "phone_verified", "sub"}, result.InspectedClaims)
}

func TestNormalizeRuntimeTokenMinimizationProbeResultRequiresClaims(t *testing.T) {
	result, err := NormalizeRuntimeTokenMinimizationProbeResult(RuntimeTokenMinimizationProbeResult{})

	require.ErrorIs(t, err, ErrTokenMinimizationProbeFailed)
	assert.Empty(t, result.InspectedClaims)
	assert.Contains(t, err.Error(), "did not return inspected claims")
}

func TestCommandRuntimeTokenProberNormalizesJSONEvidence(t *testing.T) {
	script := filepath.Join(t.TempDir(), "probe")
	require.NoError(t, os.WriteFile(script, []byte(`#!/usr/bin/env bash
set -euo pipefail
[[ "${CASDOOR_ISSUER}" == "https://sso.example.com" ]]
[[ "${CASDOOR_TOKEN_PROBE_CLIENT_ID}" == "runtime-client" ]]
[[ "${CASDOOR_TOKEN_PROBE_CLIENT_SECRET}" == "runtime-secret" ]]
[[ "${CASDOOR_TOKEN_PROBE_REDIRECT_URI}" == "https://client.example.com/callback" ]]
[[ "${CASDOOR_TOKEN_PROBE_SCOPE}" == "openid" ]]
[[ "${CASDOOR_TOKEN_PROBE_OUTPUT}" == "json" ]]
cat >/dev/null
printf '%s\n' '{"method":"authorization_code","tokenClaims":{"id_token":["sub","iss"],"access_token":["sub"]},"metadata":{"source":"test"}}'
`), 0o700))
	prober, err := NewCommandRuntimeTokenProber(RuntimeTokenProbeCommandConfig{
		Command: script,
		Issuer:  "https://sso.example.com",
		Timeout: time.Second,
	})
	require.NoError(t, err)

	result, err := prober.ProbeTokenMinimization(context.Background(), ApplicationSpec{
		Name:         "runtime-app",
		ClientID:     "runtime-client",
		ClientSecret: "runtime-secret",
		RedirectURIs: []string{"https://client.example.com/callback"},
	})

	require.NoError(t, err)
	assert.Equal(t, "authorization_code", result.Method)
	assert.Equal(t, []string{"iss", "sub"}, result.InspectedClaims)
	assert.Empty(t, result.BusinessClaims)
	assert.Equal(t, []string{"iss", "sub"}, result.TokenClaims["id_token"])
	assert.Equal(t, "test", result.Metadata["source"])
}

func TestCreateApplicationRejectsBusinessTokenField(t *testing.T) {
	client := newTestClient(t, &fakeApplicationAPI{addOK: true})
	spec := validApplicationSpec()
	spec.TokenFields = []string{"phone_verified"}

	err := client.CreateApplication(t.Context(), spec)

	require.ErrorIs(t, err, ErrTokenMinimizationProbeFailed)
}

func unsignedJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "none", "typ": "JWT"})
	require.NoError(t, err)
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	return strings.Join([]string{
		base64.RawURLEncoding.EncodeToString(header),
		base64.RawURLEncoding.EncodeToString(payload),
		"",
	}, ".")
}
