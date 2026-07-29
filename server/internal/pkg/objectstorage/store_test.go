package objectstorage

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStoreCheckBucketUsesConfiguredTLSCAFile(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodHead, r.Method)
		require.Equal(t, "/stuhelper-test", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	caPath := writeServerCertificatePEM(t, server)
	t.Setenv("AWS_CA_BUNDLE", caPath)
	store, err := New(context.Background(), Config{
		Endpoint:        server.URL,
		Region:          "us-east-1",
		Bucket:          "stuhelper-test",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		UseSSL:          true,
		ForcePathStyle:  true,
		PresignTTL:      time.Minute,
		TLSCAFile:       caPath,
	})
	require.NoError(t, err)

	require.NoError(t, store.CheckBucket(context.Background()))
}

func TestStoreNewRejectsInvalidTLSCAFile(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), Config{
		Endpoint:        "https://object-storage:8334",
		Region:          "us-east-1",
		Bucket:          "stuhelper-test",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		UseSSL:          true,
		ForcePathStyle:  true,
		PresignTTL:      time.Minute,
		TLSCAFile:       t.TempDir() + "/missing.crt",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "load_ca_bundle")
}

func writeServerCertificatePEM(t *testing.T, server *httptest.Server) string {
	t.Helper()

	cert := server.Certificate()
	require.NotNil(t, cert)
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	require.NotEmpty(t, caPEM)

	caPath := t.TempDir() + "/ca.crt"
	require.NoError(t, os.WriteFile(caPath, caPEM, 0o644))
	return caPath
}
