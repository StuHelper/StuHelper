package campusconnectorprotocol

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInteractiveDeliveryRoundTripKeepsPasswordBinary(t *testing.T) {
	password := []byte("request-scoped-password")
	metadata := InteractiveMetadata{
		RequestID: "fdbf13a4-817b-4d3c-b9e2-5e4c8060ac47",
		SchoolID:  1, SchoolCode: "0000000001", OperationKey: "buaa.school_account.authenticate",
		AdapterID: "buaa_ldap_bind", AdapterVersion: "1",
		StudentID: "20990001", DeadlineAt: time.Now().Add(time.Minute).UTC(),
	}
	var wire bytes.Buffer
	require.NoError(t, WriteInteractiveDelivery(&wire, metadata, password))
	require.NotContains(t, wire.String(), `"password"`)
	require.True(t, bytes.HasSuffix(wire.Bytes(), password))

	gotMetadata, gotPassword, err := ReadInteractiveDelivery(&wire, 256)
	require.NoError(t, err)
	require.Equal(t, metadata, gotMetadata)
	require.Equal(t, password, gotPassword)
}

func TestSignedRequestRejectsBodyMutation(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, "https://connector.example/v1/heartbeat?wait=1", nil)
	require.NoError(t, err)
	nonce := bytes.Repeat([]byte{7}, 24)
	body := []byte(`{"softwareVersion":"test"}`)
	require.NoError(t, SignRequest(req, "node-1", "key-1", privateKey, body, time.Unix(1_700_000_000, 0), nonce))
	require.True(t, VerifyRequestSignature(req, publicKey, body))
	require.False(t, VerifyRequestSignature(req, publicKey, append(body, byte('x'))))
}

func TestEncryptedSnapshotRoundTripAndTamperRejection(t *testing.T) {
	curve := ecdh.X25519()
	recipient, err := curve.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signingPublic, signingPrivate, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	plaintext := []byte(`{"records":[{"studentID":"20990001"}]}`)
	manifest := SnapshotManifest{
		SchemaVersion: 1, RequestID: "fdbf13a4-817b-4d3c-b9e2-5e4c8060ac47",
		NodeID: "node-1", SchoolCode: "1000000006",
		OperationKey: "buaa.roster.full", AdapterID: "buaa_oracle_roster",
		AdapterVersion: "1", MappingVersion: "buaa-v1", SourceVersion: "scn-1",
		SourceStartedAt: time.Now().Add(-time.Minute).UTC(), SourceCutoffAt: time.Now().UTC(),
		RowCount: 1, EncryptionKeyID: "central-x25519-1", SigningKeyID: "node-ed25519-1",
	}
	envelope, err := EncryptSnapshot(manifest, plaintext, recipient.PublicKey().Bytes(), signingPrivate)
	require.NoError(t, err)
	decoded, err := DecryptSnapshot(*envelope, recipient.Bytes(), signingPublic, 1024)
	require.NoError(t, err)
	require.Equal(t, plaintext, decoded)

	tampered := *envelope
	tampered.Manifest.SourceVersion = "scn-2"
	_, err = DecryptSnapshot(tampered, recipient.Bytes(), signingPublic, 1024)
	require.ErrorIs(t, err, ErrSnapshotSignature)
}

func TestOperationTargetFingerprintIsExactAndStable(t *testing.T) {
	first := OperationTargetFingerprint("ldaps", "ldap.internal", 636, "ldap.internal")
	second := OperationTargetFingerprint(" LDAPS ", "LDAP.INTERNAL", 636, "LDAP.INTERNAL")
	require.Equal(t, first, second)
	require.NotEqual(t, first, OperationTargetFingerprint("ldaps", "ldap.internal", 1636, "ldap.internal"))
}
