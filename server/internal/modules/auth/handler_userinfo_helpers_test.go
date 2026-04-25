package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserInfoHelpers(t *testing.T) {
	assert.Nil(t, nullableString("   "))
	assert.Equal(t, "avatar", *nullableString("avatar"))

	snapshot := buildAccessSnapshotForRoles([]string{"super_admin"}, nil)
	assert.NotEmpty(t, snapshot.Capabilities)
	assert.NotEmpty(t, snapshot.GlobalCapabilities)
}
