package app

import (
	"context"
	"strings"

	"github.com/StuHelper/StuHelper/server/internal/modules/studentverification"
)

type staticInboundEmailTargetResolver struct {
	targetAddress string
}

func (r staticInboundEmailTargetResolver) TargetAddress(context.Context, string) (string, error) {
	return r.targetAddress, nil
}

func newInboundEmailTargetResolver(enabled bool, targetAddress string) studentverification.InboundEmailTargetResolver {
	if !enabled || strings.TrimSpace(targetAddress) == "" {
		return nil
	}
	return staticInboundEmailTargetResolver{targetAddress: strings.TrimSpace(targetAddress)}
}
