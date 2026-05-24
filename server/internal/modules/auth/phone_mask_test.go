package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskPhone(t *testing.T) {
	assert.Equal(t, "138****8000", maskPhone("13800138000"))
	assert.Equal(t, "***", maskPhone("12345"))
}
