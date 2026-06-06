package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	assert.PanicsWithValue(t, "resource.NewRepository: database must not be nil", func() {
		NewRepository(nil)
	})
}
