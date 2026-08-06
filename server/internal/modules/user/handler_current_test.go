package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHandlerRequiresService(t *testing.T) {
	assert.PanicsWithValue(t, "user.NewHandler: service must not be nil", func() {
		NewHandler(nil)
	})
}
