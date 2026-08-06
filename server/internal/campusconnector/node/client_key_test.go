package node

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadConnectorKeyAcceptsStandardPEMAndLegacyBase64(t *testing.T) {
	t.Parallel()

	_, signingPrivate, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signingDER, err := x509.MarshalPKCS8PrivateKey(signingPrivate)
	require.NoError(t, err)
	signingPath := filepath.Join(t.TempDir(), "signing.key")
	require.NoError(t, os.WriteFile(signingPath, pem.EncodeToMemory(&pem.Block{
		Type: "PRIVATE KEY", Bytes: signingDER,
	}), 0o600))
	loadedSigning, err := readConnectorKey(signingPath, ed25519.PrivateKeySize)
	require.NoError(t, err)
	require.Equal(t, []byte(signingPrivate), loadedSigning)

	snapshotPrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	snapshotDER, err := x509.MarshalPKIXPublicKey(snapshotPrivate.PublicKey())
	require.NoError(t, err)
	snapshotPath := filepath.Join(t.TempDir(), "snapshot.pub")
	require.NoError(t, os.WriteFile(snapshotPath, pem.EncodeToMemory(&pem.Block{
		Type: "PUBLIC KEY", Bytes: snapshotDER,
	}), 0o600))
	loadedSnapshot, err := readConnectorKey(snapshotPath, 32)
	require.NoError(t, err)
	require.Equal(t, snapshotPrivate.PublicKey().Bytes(), loadedSnapshot)

	legacyPath := filepath.Join(t.TempDir(), "legacy.key")
	require.NoError(t, os.WriteFile(
		legacyPath,
		[]byte(base64.RawStdEncoding.EncodeToString(snapshotPrivate.PublicKey().Bytes())),
		0o600,
	))
	loadedLegacy, err := readConnectorKey(legacyPath, 32)
	require.NoError(t, err)
	require.Equal(t, snapshotPrivate.PublicKey().Bytes(), loadedLegacy)
}

func TestReadConnectorKeyRejectsWrongPEMCurve(t *testing.T) {
	t.Parallel()

	privateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	publicDER, err := x509.MarshalPKIXPublicKey(privateKey.PublicKey())
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "wrong-curve.pub")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{
		Type: "PUBLIC KEY", Bytes: publicDER,
	}), 0o600))

	_, err = readConnectorKey(path, 32)
	require.Error(t, err)
}
