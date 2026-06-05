package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/ldap"
	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
)

func TestNewLDAPAuthClientRejectsInvalidConfig(t *testing.T) {
	_, err := newLDAPAuthClient(user.LDAPConfig{})

	require.Error(t, err)
}

func TestNormalizeLDAPClientErrorMapsInvalidUID(t *testing.T) {
	err := normalizeLDAPClientError(errors.Join(errors.New("wrapped"), ldap.ErrInvalidUID))

	require.ErrorIs(t, err, user.ErrLDAPFailed)
	require.False(t, errors.Is(err, ldap.ErrInvalidUID), "ldap module error must not leak through adapter")
}
