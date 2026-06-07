package openplatform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type typedNilRuntimeProber struct{}

func (*typedNilRuntimeProber) ProbeTokenMinimization(
	context.Context,
	ProvisionedApplicationSpec,
) (RuntimeTokenMinimizationProbeResult, error) {
	return RuntimeTokenMinimizationProbeResult{}, nil
}

func TestWithRuntimeTokenProbeTreatsTypedNilAsUnconfigured(t *testing.T) {
	var prober *typedNilRuntimeProber
	service := &Service{}

	WithRuntimeTokenProbe(prober, false)(service)

	assert.Nil(t, service.tokenProber)
	assert.False(t, service.tokenProbeRequired)
}

func TestWithRuntimeTokenProbeKeepsRequiredFlagForTypedNil(t *testing.T) {
	var prober *typedNilRuntimeProber
	service := &Service{}

	WithRuntimeTokenProbe(prober, true)(service)

	assert.Nil(t, service.tokenProber)
	assert.True(t, service.tokenProbeRequired)
}
