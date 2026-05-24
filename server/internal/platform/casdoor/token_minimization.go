package casdoor

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

var ErrTokenMinimizationProbeFailed = errors.New("casdoor: token minimization probe failed")

type TokenMinimizationProbeResult struct {
	InspectedClaims []string
	BusinessClaims  []string
}

type TokenMinimizationError struct {
	Claims []string
}

func (e TokenMinimizationError) Error() string {
	if len(e.Claims) == 0 {
		return ErrTokenMinimizationProbeFailed.Error()
	}
	return fmt.Sprintf("%s: forbidden business claims: %s",
		ErrTokenMinimizationProbeFailed,
		strings.Join(e.Claims, ", "),
	)
}

func (e TokenMinimizationError) Unwrap() error {
	return ErrTokenMinimizationProbeFailed
}

func ProbeApplicationSpecTokenMinimization(spec ApplicationSpec) (TokenMinimizationProbeResult, error) {
	if spec.TokenFormat != "JWT-Custom" {
		return TokenMinimizationProbeResult{}, fmt.Errorf(
			"%w: Casdoor token format must be JWT-Custom for minimized third-party tokens (got %s)",
			ErrTokenMinimizationProbeFailed,
			spec.TokenFormat,
		)
	}
	fields, err := normalizeExplicitTokenFields(spec.TokenFields)
	if err != nil {
		return TokenMinimizationProbeResult{}, fmt.Errorf("%w: %v", ErrTokenMinimizationProbeFailed, err)
	}
	return ProbeTokenClaimsMinimized(fields)
}

func ProbeTokenClaimsMinimized(claims []string) (TokenMinimizationProbeResult, error) {
	inspected := normalizedClaimKeys(claims)
	forbidden := forbiddenBusinessClaims(inspected)
	result := TokenMinimizationProbeResult{
		InspectedClaims: inspected,
		BusinessClaims:  forbidden,
	}
	if len(forbidden) > 0 {
		return result, TokenMinimizationError{Claims: forbidden}
	}
	return result, nil
}

func ProbeJWTClaimsMinimized(raw string) (TokenMinimizationProbeResult, error) {
	payload, err := decodeJWTPayload(raw)
	if err != nil {
		return TokenMinimizationProbeResult{}, err
	}
	claims := make([]string, 0, len(payload))
	for key := range payload {
		claims = append(claims, key)
	}
	return ProbeTokenClaimsMinimized(claims)
}

func normalizeExplicitTokenFields(fields []string) ([]string, error) {
	if fields == nil {
		return nil, errors.New("casdoor: token fields must be explicit")
	}
	return normalizeList("token field", fields)
}

func normalizedClaimKeys(claims []string) []string {
	seen := make(map[string]struct{}, len(claims))
	result := make([]string, 0, len(claims))
	for _, claim := range claims {
		key := canonicalClaimKey(claim)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func forbiddenBusinessClaims(claims []string) []string {
	forbiddenSet := map[string]struct{}{
		"phone":                         {},
		"phone_number":                  {},
		"phone_verified":                {},
		"phone_number_verified":         {},
		"password":                      {},
		"password_salt":                 {},
		"password_type":                 {},
		"identity_verified":             {},
		"identity_type":                 {},
		"student_verified":              {},
		"school":                        {},
		"school_id":                     {},
		"school_name":                   {},
		"qq":                            {},
		"qq_binding":                    {},
		"stuhelper_identity_verified":   {},
		"stuhelper_identity_type":       {},
		"stuhelper_student_verified":    {},
		"stuhelper_student_school":      {},
		"stuhelper_student_school_id":   {},
		"stuhelper_student_school_name": {},
	}
	result := make([]string, 0)
	for _, claim := range claims {
		if _, ok := forbiddenSet[claim]; ok {
			result = append(result, claim)
		}
	}
	return result
}

func canonicalClaimKey(raw string) string {
	key := strings.TrimSpace(raw)
	var builder strings.Builder
	for i, r := range key {
		if i > 0 && unicode.IsUpper(r) {
			builder.WriteByte('_')
		}
		builder.WriteRune(unicode.ToLower(r))
	}
	key = builder.String()
	key = strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(key)
	for strings.Contains(key, "__") {
		key = strings.ReplaceAll(key, "__", "_")
	}
	return strings.Trim(key, "_")
}

func decodeJWTPayload(raw string) (map[string]any, error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%w: invalid JWT compact serialization", ErrTokenMinimizationProbeFailed)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: decode JWT payload: %v", ErrTokenMinimizationProbeFailed, err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("%w: parse JWT payload: %v", ErrTokenMinimizationProbeFailed, err)
	}
	return claims, nil
}
