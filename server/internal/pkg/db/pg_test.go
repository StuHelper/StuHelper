package db

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
)

func TestConfigurePGTLSVerifyFullRemovesPlaintextPreferFallback(t *testing.T) {
	poolCfg := parsePGPoolConfig(t,
		"postgres://user:pass@db.example.test:5432/stuhelper?sslmode=prefer",
	)
	require.NotNil(t, poolCfg.ConnConfig.TLSConfig)
	require.Len(t, poolCfg.ConnConfig.Fallbacks, 1)
	assert.Nil(t, poolCfg.ConnConfig.Fallbacks[0].TLSConfig)

	err := configurePGTLS(poolCfg, config.DatabaseConfig{SSLMode: "verify-full"})

	require.NoError(t, err)
	require.NotNil(t, poolCfg.ConnConfig.TLSConfig)
	assert.Equal(t, "db.example.test", poolCfg.ConnConfig.TLSConfig.ServerName)
	assert.Empty(t, poolCfg.ConnConfig.Fallbacks,
		"TLS and plaintext attempts for the same endpoint must collapse to one verified attempt")
}

func TestConfigurePGTLSVerifyFullSecuresEveryMultiHostFallback(t *testing.T) {
	poolCfg := parsePGPoolConfig(t,
		"postgres://user:pass@db-a.example.test:5432,db-b.example.test:5433/stuhelper?sslmode=prefer",
	)

	err := configurePGTLS(poolCfg, config.DatabaseConfig{SSLMode: "verify-full"})

	require.NoError(t, err)
	assert.Equal(t, "db-a.example.test", poolCfg.ConnConfig.TLSConfig.ServerName)
	require.Len(t, poolCfg.ConnConfig.Fallbacks, 1)
	fallback := poolCfg.ConnConfig.Fallbacks[0]
	assert.Equal(t, "db-b.example.test", fallback.Host)
	assert.Equal(t, uint16(5433), fallback.Port)
	require.NotNil(t, fallback.TLSConfig)
	assert.Equal(t, "db-b.example.test", fallback.TLSConfig.ServerName)
	assert.False(t, fallback.TLSConfig.InsecureSkipVerify)
}

func TestConfigurePGTLSDisableClearsDSNTLSAndDuplicateFallback(t *testing.T) {
	poolCfg := parsePGPoolConfig(t,
		"postgres://user:pass@db.example.test:5432/stuhelper?sslmode=prefer",
	)

	err := configurePGTLS(poolCfg, config.DatabaseConfig{SSLMode: "disable"})

	require.NoError(t, err)
	assert.Nil(t, poolCfg.ConnConfig.TLSConfig)
	assert.Empty(t, poolCfg.ConnConfig.Fallbacks)
}

func TestConfigurePGTLSVerifyCAUsesExplicitChainVerification(t *testing.T) {
	poolCfg := parsePGPoolConfig(t,
		"postgres://user:pass@db.example.test:5432/stuhelper?sslmode=disable",
	)

	err := configurePGTLS(poolCfg, config.DatabaseConfig{SSLMode: "verify-ca"})

	require.NoError(t, err)
	tlsConfig := poolCfg.ConnConfig.TLSConfig
	require.NotNil(t, tlsConfig)
	assert.True(t, tlsConfig.InsecureSkipVerify,
		"verify-ca intentionally skips only the standard hostname check")
	require.NotNil(t, tlsConfig.VerifyConnection)
	assert.Equal(t, "db.example.test", tlsConfig.ServerName)
	assert.Error(t, tlsConfig.VerifyConnection(tls.ConnectionState{}))
}

func TestPGCertificateChainVerifierAcceptsTrustedWrongHostnameAndRejectsUntrusted(t *testing.T) {
	ca, caKey := newPGTestCertificateAuthority(t, "trusted CA")
	leaf := newPGTestServerCertificate(t, ca, caKey, "different.example.test")
	roots := x509.NewCertPool()
	roots.AddCert(ca)
	verify := pgCertificateChainVerifier(roots)

	require.NoError(t, verify(tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{leaf, ca},
	}), "verify-ca validates trust without enforcing the DNS name")

	untrustedCA, untrustedKey := newPGTestCertificateAuthority(t, "untrusted CA")
	untrustedLeaf := newPGTestServerCertificate(t, untrustedCA, untrustedKey, "db.example.test")
	require.Error(t, verify(tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{untrustedLeaf, untrustedCA},
	}))
}

func TestConfigurePGTLSRejectsPartialClientCertificateAndVerifiedUnixSocket(t *testing.T) {
	t.Run("partial client certificate", func(t *testing.T) {
		poolCfg := parsePGPoolConfig(t,
			"postgres://user:pass@db.example.test:5432/stuhelper?sslmode=disable",
		)

		err := configurePGTLS(poolCfg, config.DatabaseConfig{
			SSLMode: "verify-full",
			SSLCert: "/run/secrets/postgres-client.crt",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "both DB_SSL_CERT and DB_SSL_KEY")
	})

	t.Run("verified unix socket", func(t *testing.T) {
		poolCfg := parsePGPoolConfig(t,
			"host=/var/run/postgresql port=5432 user=stuhelper dbname=stuhelper sslmode=disable",
		)

		err := configurePGTLS(poolCfg, config.DatabaseConfig{SSLMode: "verify-full"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot use Unix socket")
	})
}

func TestConfigurePGTLSPreservesDirectNegotiationALPN(t *testing.T) {
	poolCfg := parsePGPoolConfig(t,
		"postgres://user:pass@db.example.test:5432/stuhelper?sslmode=require&sslnegotiation=direct",
	)

	err := configurePGTLS(poolCfg, config.DatabaseConfig{SSLMode: "verify-full"})

	require.NoError(t, err)
	assert.Equal(t, []string{"postgresql"}, poolCfg.ConnConfig.TLSConfig.NextProtos)
}

func parsePGPoolConfig(t *testing.T, dsn string) *pgxpool.Config {
	t.Helper()
	poolCfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	return poolCfg
}

func newPGTestCertificateAuthority(t *testing.T, commonName string) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return certificate, key
}

func newPGTestServerCertificate(
	t *testing.T,
	ca *x509.Certificate,
	caKey *ecdsa.PrivateKey,
	dnsName string,
) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: dnsName},
		DNSNames:     []string{dnsName},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return certificate
}
