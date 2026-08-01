package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeHTTPServerShutdown struct {
	shutdownErr   error
	closeErr      error
	shutdownCalls int
	closeCalls    int
}

func (s *fakeHTTPServerShutdown) Shutdown(context.Context) error {
	s.shutdownCalls++
	return s.shutdownErr
}

func (s *fakeHTTPServerShutdown) Close() error {
	s.closeCalls++
	return s.closeErr
}

func TestShutdownHTTPServerGracefulSuccessDoesNotForceClose(t *testing.T) {
	srv := &fakeHTTPServerShutdown{}

	err := shutdownHTTPServer(context.Background(), srv)

	require.NoError(t, err)
	assert.Equal(t, 1, srv.shutdownCalls)
	assert.Zero(t, srv.closeCalls)
}

func TestShutdownHTTPServerForcesCloseAfterGracefulFailure(t *testing.T) {
	shutdownErr := context.DeadlineExceeded
	srv := &fakeHTTPServerShutdown{shutdownErr: shutdownErr}

	err := shutdownHTTPServer(context.Background(), srv)

	require.ErrorIs(t, err, shutdownErr)
	assert.Contains(t, err.Error(), "graceful HTTP shutdown")
	assert.Equal(t, 1, srv.shutdownCalls)
	assert.Equal(t, 1, srv.closeCalls)
}

func TestShutdownHTTPServerPreservesGracefulAndForceCloseErrors(t *testing.T) {
	shutdownErr := context.DeadlineExceeded
	closeErr := errors.New("close listener")
	srv := &fakeHTTPServerShutdown{
		shutdownErr: shutdownErr,
		closeErr:    closeErr,
	}

	err := shutdownHTTPServer(context.Background(), srv)

	require.ErrorIs(t, err, shutdownErr)
	require.ErrorIs(t, err, closeErr)
	assert.Equal(t, 1, srv.shutdownCalls)
	assert.Equal(t, 1, srv.closeCalls)
}
