package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminMFAMiddlewaresSkippedInDevelopment(t *testing.T) {
	require.Empty(t, adminMFAMiddlewares("development", nil))
}

func TestAdminMFAMiddlewaresSkippedOutsideDevelopment(t *testing.T) {
	require.Empty(t, adminMFAMiddlewares("production", nil))
}
