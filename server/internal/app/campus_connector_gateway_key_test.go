package app

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadSnapshotPrivateKeyFileAcceptsPEMAndLegacyBase64(t *testing.T) {
	t.Parallel()

	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	pemPath := filepath.Join(t.TempDir(), "snapshot.key")
	require.NoError(t, os.WriteFile(pemPath, pem.EncodeToMemory(&pem.Block{
		Type: "PRIVATE KEY", Bytes: privateDER,
	}), 0o600))
	loadedPEM, err := readSnapshotPrivateKeyFile(pemPath)
	require.NoError(t, err)
	require.Equal(t, privateKey.Bytes(), loadedPEM)

	legacyPath := filepath.Join(t.TempDir(), "snapshot-base64.key")
	require.NoError(t, os.WriteFile(
		legacyPath,
		[]byte(base64.RawStdEncoding.EncodeToString(privateKey.Bytes())),
		0o600,
	))
	loadedLegacy, err := readSnapshotPrivateKeyFile(legacyPath)
	require.NoError(t, err)
	require.Equal(t, privateKey.Bytes(), loadedLegacy)
}

func TestReadSnapshotPrivateKeyFileRejectsWrongPEMCurve(t *testing.T) {
	t.Parallel()

	privateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "wrong-curve.key")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{
		Type: "PRIVATE KEY", Bytes: privateDER,
	}), 0o600))

	_, err = readSnapshotPrivateKeyFile(path)
	require.Error(t, err)
}
