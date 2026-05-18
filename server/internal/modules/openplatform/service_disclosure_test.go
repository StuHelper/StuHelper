package openplatform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildConsentURLUsesConnectRoute(t *testing.T) {
	assert.Equal(t, "/connect/consent?token=abc", buildConsentURL("", "abc"))
	assert.Equal(t,
		"https://stuhelper.com/connect/consent?token=abc",
		buildConsentURL("https://stuhelper.com/", "abc"),
	)
}
