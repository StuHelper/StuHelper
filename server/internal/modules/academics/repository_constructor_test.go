package academics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	assert.PanicsWithValue(t, "academics.NewRepository: database must not be nil", func() {
		NewRepository(nil)
	})
}
