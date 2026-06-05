package casdoor

import "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/tokenprobe"

var ErrTokenMinimizationProbeFailed = tokenprobe.ErrTokenMinimizationProbeFailed

type TokenMinimizationProbeResult = tokenprobe.TokenMinimizationProbeResult
type TokenMinimizationError = tokenprobe.TokenMinimizationError
type RuntimeTokenMinimizationProbeResult = tokenprobe.RuntimeTokenMinimizationProbeResult

func ProbeApplicationSpecTokenMinimization(spec ApplicationSpec) (TokenMinimizationProbeResult, error) {
	return tokenprobe.ProbeApplicationTokenFieldsMinimized(spec.TokenFormat, spec.TokenFields)
}

func normalizeExplicitTokenFields(fields []string) ([]string, error) {
	return tokenprobe.NormalizeExplicitTokenFields(fields)
}

func ProbeTokenClaimsMinimized(claims []string) (TokenMinimizationProbeResult, error) {
	return tokenprobe.ProbeTokenClaimsMinimized(claims)
}

func ProbeJWTClaimsMinimized(raw string) (TokenMinimizationProbeResult, error) {
	return tokenprobe.ProbeJWTClaimsMinimized(raw)
}

func NormalizeRuntimeTokenMinimizationProbeResult(
	result RuntimeTokenMinimizationProbeResult,
) (RuntimeTokenMinimizationProbeResult, error) {
	return tokenprobe.NormalizeRuntimeTokenMinimizationProbeResult(result)
}
