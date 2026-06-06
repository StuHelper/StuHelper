package academics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHandlerPanicsWhenServiceNil(t *testing.T) {
	assert.PanicsWithValue(t, "academics.NewHandler: service must not be nil", func() {
		NewHandler(nil)
	})
}
