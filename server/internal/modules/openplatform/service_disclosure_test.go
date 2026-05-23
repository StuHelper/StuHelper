package openplatform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildConsentURLUsesIdentityRoute(t *testing.T) {
	assert.Equal(t, "/consent?token=abc", buildConsentURL("", "abc"))
	assert.Equal(t,
		"https://id.stuhelper.com/consent?token=abc",
		buildConsentURL("https://id.stuhelper.com/", "abc"),
	)
}
