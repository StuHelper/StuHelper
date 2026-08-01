package authorization

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextProjectionReconciliationDelayTargetsDaily0320(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	tests := []struct {
		name string
		now  time.Time
		want time.Duration
	}{
		{
			name: "before window",
			now:  time.Date(2026, 7, 31, 3, 19, 0, 0, location),
			want: time.Minute,
		},
		{
			name: "at window",
			now:  time.Date(2026, 7, 31, 3, 20, 0, 0, location),
			want: 24 * time.Hour,
		},
		{
			name: "after window",
			now:  time.Date(2026, 7, 31, 4, 20, 0, 0, location),
			want: 23 * time.Hour,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, nextProjectionReconciliationDelay(test.now))
		})
	}
}

func TestStartBackgroundJobsRegistersProjectionAndReconciliation(t *testing.T) {
	service := &Service{
		repo:       &Repository{},
		projection: newFakeProjectionClient(),
	}
	names := make([]string, 0, 2)
	service.StartBackgroundJobs(
		context.Background(),
		func(name string, _ func(context.Context)) {
			names = append(names, name)
		},
	)
	require.ElementsMatch(t, []string{
		"authorization grant projection worker",
		"authorization grant projection reconciliation",
	}, names)
}
