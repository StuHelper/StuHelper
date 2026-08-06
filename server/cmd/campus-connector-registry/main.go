// Command campus-connector-registry validates and atomically applies one
// operator-reviewed campus connector node manifest. It never accepts private
// keys, upstream credentials, arbitrary routes, SQL, or shell operations.
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/modules/campusconnector"
	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
)

type manifestFile struct {
	Node struct {
		ID                       string `json:"id"`
		DisplayName              string `json:"displayName"`
		ProtocolVersion          string `json:"protocolVersion"`
		SoftwareVersion          string `json:"softwareVersion"`
		ClientCertificateFile    string `json:"clientCertificateFile"`
		SigningKeyID             string `json:"signingKeyID"`
		SigningPublicKeyFile     string `json:"signingPublicKeyFile"`
		MaxConcurrency           int    `json:"maxConcurrency"`
		HeartbeatIntervalSeconds int    `json:"heartbeatIntervalSeconds"`
		ExpectedRevision         int64  `json:"expectedRevision"`
	} `json:"node"`
	Operations []struct {
		SchoolCode            string   `json:"schoolCode"`
		OperationKey          string   `json:"operationKey"`
		OperationType         string   `json:"operationType"`
		AdapterID             string   `json:"adapterID"`
		AdapterVersion        string   `json:"adapterVersion"`
		UpstreamProtocol      string   `json:"upstreamProtocol"`
		TargetHost            string   `json:"targetHost"`
		TargetPort            int      `json:"targetPort"`
		TargetTLSServerName   *string  `json:"targetTLSServerName"`
		AllowlistedAttributes []string `json:"allowlistedAttributes"`
		TimeoutMilliseconds   int      `json:"timeoutMilliseconds"`
		MaxConcurrency        int      `json:"maxConcurrency"`
		RateLimitPerMinute    int      `json:"rateLimitPerMinute"`
		ValidationStatus      string   `json:"validationStatus"`
		Enabled               bool     `json:"enabled"`
		ExpectedRevision      int64    `json:"expectedRevision"`
	} `json:"operations"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "campus-connector-registry: operation failed")
		os.Exit(1)
	}
}

func run() error {
	manifestPath := flag.String("manifest", "", "path to the operator-reviewed public registry manifest")
	reason := flag.String("reason", "", "audited reason, 4-500 characters")
	apply := flag.Bool("apply", false, "apply after validation; the default is a read-only dry run")
	flag.Parse()
	manifest, err := loadManifest(*manifestPath, *reason)
	if err != nil {
		return err
	}
	if !*apply {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"apply": false, "nodeID": manifest.Node.ID,
			"operationCount": len(manifest.Operations), "valid": true,
		})
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	dbConfig := config.DatabaseConfig{
		URL:      databaseURL,
		MaxConns: 4, MinConns: 0,
		MaxConnLifetime: envInt("DB_MAX_CONN_LIFETIME", 30),
		MaxConnIdleTime: envInt("DB_MAX_CONN_IDLE_TIME", 5),
		QueryTimeout:    envInt("DB_QUERY_TIMEOUT", 15),
		SSLMode:         envString("DB_SSL_MODE", "verify-full"),
		SSLRootCert:     os.Getenv("DB_SSL_ROOT_CERT"),
		SSLCert:         os.Getenv("DB_SSL_CERT"),
		SSLKey:          os.Getenv("DB_SSL_KEY"),
	}
	pool, err := db.NewPGPool(dbConfig)
	if err != nil {
		return err
	}
	database := db.NewDB(pool, time.Duration(dbConfig.QueryTimeout)*time.Second)
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := campusconnector.NewRepository(database).ApplyRegistryManifest(ctx, manifest, time.Now().UTC()); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"apply": true, "nodeID": manifest.Node.ID,
		"operationCount": len(manifest.Operations),
	})
}

func loadManifest(path string, reason string) (campusconnector.RegistryManifest, error) {
	raw, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return campusconnector.RegistryManifest{}, err
	}
	defer wipe(raw)
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var input manifestFile
	if err := decoder.Decode(&input); err != nil {
		return campusconnector.RegistryManifest{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return campusconnector.RegistryManifest{}, errors.New("registry manifest contains trailing JSON")
	}
	certificate, err := readLeafCertificate(input.Node.ClientCertificateFile)
	if err != nil {
		return campusconnector.RegistryManifest{}, err
	}
	fingerprint := sha256.Sum256(certificate.Raw)
	publicKey, err := readPublicKey(input.Node.SigningPublicKeyFile)
	if err != nil {
		return campusconnector.RegistryManifest{}, err
	}
	manifest := campusconnector.RegistryManifest{
		Node: campusconnector.RegistryNode{
			ID: input.Node.ID, DisplayName: input.Node.DisplayName,
			ProtocolVersion:        input.Node.ProtocolVersion,
			SoftwareVersion:        input.Node.SoftwareVersion,
			CertificateFingerprint: hex.EncodeToString(fingerprint[:]),
			CertificateNotAfter:    certificate.NotAfter.UTC(),
			SigningKeyID:           input.Node.SigningKeyID, SigningPublicKey: publicKey,
			MaxConcurrency:           input.Node.MaxConcurrency,
			HeartbeatIntervalSeconds: input.Node.HeartbeatIntervalSeconds,
			ExpectedRevision:         input.Node.ExpectedRevision,
		},
		Operations: make([]campusconnector.RegistryOperation, len(input.Operations)),
		Reason:     strings.TrimSpace(reason),
	}
	for index := range input.Operations {
		operation := input.Operations[index]
		manifest.Operations[index] = campusconnector.RegistryOperation{
			SchoolCode: operation.SchoolCode, OperationKey: operation.OperationKey,
			OperationType: operation.OperationType, AdapterID: operation.AdapterID,
			AdapterVersion: operation.AdapterVersion, UpstreamProtocol: operation.UpstreamProtocol,
			TargetHost: operation.TargetHost, TargetPort: operation.TargetPort,
			TargetTLSServerName:   operation.TargetTLSServerName,
			AllowlistedAttributes: operation.AllowlistedAttributes,
			TimeoutMilliseconds:   operation.TimeoutMilliseconds,
			MaxConcurrency:        operation.MaxConcurrency,
			RateLimitPerMinute:    operation.RateLimitPerMinute,
			ValidationStatus:      operation.ValidationStatus, Enabled: operation.Enabled,
			ExpectedRevision: operation.ExpectedRevision,
		}
	}
	if err := manifest.Validate(time.Now().UTC()); err != nil {
		wipe(publicKey)
		return campusconnector.RegistryManifest{}, err
	}
	return manifest, nil
}

func readLeafCertificate(path string) (*x509.Certificate, error) {
	raw, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("client certificate file contains no certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}

func readPublicKey(path string) ([]byte, error) {
	raw, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	defer wipe(raw)
	if block, _ := pem.Decode(raw); block != nil {
		parsed, parseErr := x509.ParsePKIXPublicKey(block.Bytes)
		key, ok := parsed.(ed25519.PublicKey)
		if parseErr != nil || !ok || len(key) != ed25519.PublicKeySize {
			return nil, errors.New("signing public key must be an Ed25519 PKIX public key")
		}
		return append([]byte(nil), key...), nil
	}
	encoded := bytes.TrimSpace(raw)
	decoded := make([]byte, base64.RawStdEncoding.DecodedLen(len(encoded)))
	n, decodeErr := base64.RawStdEncoding.Decode(decoded, encoded)
	if decodeErr != nil {
		decoded = make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
		n, decodeErr = base64.StdEncoding.Decode(decoded, encoded)
	}
	if decodeErr != nil || n != ed25519.PublicKeySize {
		wipe(decoded)
		return nil, errors.New("signing public key file is invalid")
	}
	return decoded[:n], nil
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func wipe(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
