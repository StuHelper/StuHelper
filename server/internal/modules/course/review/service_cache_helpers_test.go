package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDimensionNames_ReturnsCachedMapOrNil(t *testing.T) {
	svc := &Service{}
	assert.Nil(t, svc.getDimensionNames())

	want := map[string]string{"teaching": "教学质量"}
	svc.dimensionCache.Store(want)
	assert.Equal(t, want, svc.getDimensionNames())
}
