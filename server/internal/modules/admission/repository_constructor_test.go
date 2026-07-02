package admission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	assert.PanicsWithValue(t, "admission.NewRepository: database must not be nil", func() {
		NewRepository(nil, newTestAuthURLCipher(t))
	})
}
