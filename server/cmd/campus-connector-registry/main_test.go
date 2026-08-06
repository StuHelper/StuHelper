package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadPublicKeyAcceptsEd25519PKIXPEM(t *testing.T) {
	t.Parallel()

	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	publicDER, err := x509.MarshalPKIXPublicKey(publicKey)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "signing.pub")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{
		Type: "PUBLIC KEY", Bytes: publicDER,
	}), 0o600))

	loaded, err := readPublicKey(path)
	require.NoError(t, err)
	require.Equal(t, []byte(publicKey), loaded)
}
