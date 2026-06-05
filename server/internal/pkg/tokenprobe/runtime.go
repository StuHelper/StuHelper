package tokenprobe

import (
	"fmt"
	"strings"
)

type RuntimeTokenMinimizationProbeResult struct {
	Method          string              `json:"method,omitempty"`
	Issuer          string              `json:"issuer,omitempty"`
	InspectedClaims []string            `json:"inspectedClaims,omitempty"`
	BusinessClaims  []string            `json:"businessClaims,omitempty"`
	TokenClaims     map[string][]string `json:"tokenClaims,omitempty"`
	Metadata        map[string]any      `json:"metadata,omitempty"`
}

func NormalizeRuntimeTokenMinimizationProbeResult(
	result RuntimeTokenMinimizationProbeResult,
) (RuntimeTokenMinimizationProbeResult, error) {
	result.Method = strings.TrimSpace(result.Method)
	if result.Method == "" {
		result.Method = "authorization_code"
	}
	result.Issuer = strings.TrimSpace(result.Issuer)
	if result.Metadata == nil {
		result.Metadata = map[string]any{}
	}
	claims := append([]string(nil), result.InspectedClaims...)
	if result.TokenClaims == nil {
		result.TokenClaims = map[string][]string{}
	}
	for tokenType, tokenClaims := range result.TokenClaims {
		normalized := normalizedClaimKeys(tokenClaims)
		result.TokenClaims[tokenType] = normalized
		claims = append(claims, normalized...)
	}
	normalizedBusinessClaims := forbiddenBusinessClaims(normalizedClaimKeys(result.BusinessClaims))
	claims = append(claims, normalizedBusinessClaims...)

	probeResult, err := ProbeTokenClaimsMinimized(claims)
	result.InspectedClaims = probeResult.InspectedClaims
	result.BusinessClaims = mergeClaimKeys(probeResult.BusinessClaims, normalizedBusinessClaims)
	if len(result.InspectedClaims) == 0 {
		return result, fmt.Errorf("%w: runtime code-flow probe did not return inspected claims",
			ErrTokenMinimizationProbeFailed)
	}
	if len(result.BusinessClaims) > 0 {
		return result, TokenMinimizationError{Claims: result.BusinessClaims}
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func mergeClaimKeys(values ...[]string) []string {
	merged := make([]string, 0)
	for _, value := range values {
		merged = append(merged, value...)
	}
	return normalizedClaimKeys(merged)
}
