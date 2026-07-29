package objectstorage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyErrorKind_UsesMessageFallbacks(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ErrorKindNetwork, classifyErrorKind(errors.New("dial tcp: lookup object-storage: no such host")))
	assert.Equal(t, ErrorKindNotFound, classifyErrorKind(errors.New("NoSuchKey: object not found")))
	assert.Equal(t, ErrorKindPermission, classifyErrorKind(errors.New("AccessDenied: forbidden")))
	assert.Equal(t, ErrorKindAuthentication, classifyErrorKind(errors.New("SignatureDoesNotMatch")))
}

func TestIsKind(t *testing.T) {
	t.Parallel()

	err := wrapError("put_object", "bucket/key", errors.New("AccessDenied"))
	assert.True(t, IsKind(err, ErrorKindPermission))
	assert.False(t, IsKind(err, ErrorKindNotFound))
}
