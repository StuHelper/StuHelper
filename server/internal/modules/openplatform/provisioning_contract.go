package openplatform

import "github.com/StuHelper/StuHelper/server/internal/pkg/tokenprobe"

type ProvisionedApplicationSpec struct {
	Organization         string
	Name                 string
	DisplayName          string
	Logo                 string
	HomepageURL          string
	Description          string
	ClientID             string
	ClientSecret         string
	RedirectURIs         []string
	GrantTypes           []string
	TokenFormat          string
	TokenFields          []string
	ExpireInHours        float64
	RefreshExpireInHours float64
}

type TokenMinimizationProbeResult = tokenprobe.TokenMinimizationProbeResult
type RuntimeTokenMinimizationProbeResult = tokenprobe.RuntimeTokenMinimizationProbeResult

func probeApplicationSpecTokenMinimization(
	spec ProvisionedApplicationSpec,
) (TokenMinimizationProbeResult, error) {
	return tokenprobe.ProbeApplicationTokenFieldsMinimized(spec.TokenFormat, spec.TokenFields)
}

func normalizeRuntimeTokenMinimizationProbeResult(
	result RuntimeTokenMinimizationProbeResult,
) (RuntimeTokenMinimizationProbeResult, error) {
	return tokenprobe.NormalizeRuntimeTokenMinimizationProbeResult(result)
}
