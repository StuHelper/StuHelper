package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	assert.PanicsWithValue(t, "storage.NewRepository: database must not be nil", func() {
		NewRepository(nil)
	})
}
