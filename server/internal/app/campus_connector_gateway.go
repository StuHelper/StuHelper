package app

import (
	"bytes"
	"crypto/ecdh"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/modules/campusconnector"
	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
)

type campusConnectorRuntime struct {
	service   *campusconnector.Service
	server    *http.Server
	tlsConfig *tls.Config
	address   string
}

func (rt *Runtime) initCampusConnectorGateway() (*campusconnector.Service, error) {
	cfg := rt.cfg.CampusConnector
	if !cfg.Enabled {
		return nil, nil
	}
	tlsConfig, err := loadCampusConnectorServerTLS(cfg)
	if err != nil {
		return nil, err
	}
	snapshotPrivateKey, err := readSnapshotPrivateKeyFile(cfg.SnapshotDecryptionKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load campus connector snapshot decryption key: %w", err)
	}
	broker := campusconnector.NewBroker(32)
	service, err := campusconnector.NewService(
		campusconnector.NewRepository(rt.database),
		broker,
		rt.redisClient.GetClient(),
		nil,
		campusconnector.Config{
			ProtocolVersion:         cfg.ProtocolVersion,
			RequestSignatureMaxSkew: time.Duration(cfg.SignatureMaxSkewSeconds) * time.Second,
			RequestReplayTTL:        time.Duration(cfg.ReplayTTLSeconds) * time.Second,
			SnapshotKeyID:           cfg.SnapshotDecryptionKeyID,
			SnapshotPrivateKey:      snapshotPrivateKey,
			MaxSnapshotPlaintext:    cfg.MaxSnapshotPlaintextBytes,
			MaxInteractivePassword:  cfg.MaxInteractivePasswordBytes,
		},
	)
	for index := range snapshotPrivateKey {
		snapshotPrivateKey[index] = 0
	}
	if err != nil {
		return nil, fmt.Errorf("initialize campus connector service: %w", err)
	}
	gateway, err := campusconnector.NewGateway(service, campusconnector.GatewayConfig{
		PollWait:     time.Duration(cfg.PollWaitSeconds) * time.Second,
		MaxJSONBytes: int64(cfg.MaxSnapshotRequestBytes),
	})
	if err != nil {
		service.Close()
		return nil, fmt.Errorf("initialize campus connector gateway: %w", err)
	}
	rt.campusConnector = &campusConnectorRuntime{
		service: service,
		server: &http.Server{
			Handler:           gateway.Handler(),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       3 * time.Minute,
			WriteTimeout:      3 * time.Minute,
			IdleTimeout:       65 * time.Second,
			MaxHeaderBytes:    64 << 10,
			TLSConfig:         tlsConfig,
		},
		tlsConfig: tlsConfig,
		address:   cfg.ListenAddress,
	}
	rt.addCleanup(service.Close)
	return service, nil
}

func (runtime *campusConnectorRuntime) listen() (net.Listener, error) {
	if runtime == nil || runtime.server == nil || runtime.tlsConfig == nil {
		return nil, errors.New("campus connector runtime is not initialized")
	}
	listener, err := net.Listen("tcp", runtime.address)
	if err != nil {
		return nil, err
	}
	return tls.NewListener(listener, runtime.tlsConfig.Clone()), nil
}

func loadCampusConnectorServerTLS(cfg config.CampusConnectorGatewayConfig) (*tls.Config, error) {
	certificate, err := tls.LoadX509KeyPair(cfg.ServerCertificateFile, cfg.ServerPrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load campus connector gateway certificate: %w", err)
	}
	// #nosec G304 -- this is administrator-controlled startup configuration, never request input.
	clientCAPEM, err := os.ReadFile(cfg.ClientCAFile)
	if err != nil {
		return nil, fmt.Errorf("read campus connector client CA: %w", err)
	}
	defer func() {
		for index := range clientCAPEM {
			clientCAPEM[index] = 0
		}
	}()
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(clientCAPEM) {
		return nil, errors.New("campus connector client CA file contains no certificate")
	}
	return &tls.Config{
		MinVersion:             tls.VersionTLS13,
		Certificates:           []tls.Certificate{certificate},
		ClientCAs:              clientCAs,
		ClientAuth:             tls.RequireAndVerifyClientCert,
		NextProtos:             []string{"h2", "http/1.1"},
		SessionTicketsDisabled: true,
	}, nil
}

func readSnapshotPrivateKeyFile(path string) ([]byte, error) {
	// #nosec G304 -- key paths are administrator-controlled startup configuration, never request input.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		for index := range raw {
			raw[index] = 0
		}
	}()
	if block, _ := pem.Decode(raw); block != nil {
		parsed, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		key, ok := parsed.(*ecdh.PrivateKey)
		if parseErr != nil || !ok || key.Curve() != ecdh.X25519() || len(key.Bytes()) != 32 {
			return nil, errors.New("snapshot key must be an X25519 PKCS#8 private key")
		}
		material := append([]byte(nil), key.Bytes()...)
		if _, validateErr := ecdh.X25519().NewPrivateKey(material); validateErr != nil {
			for index := range material {
				material[index] = 0
			}
			return nil, errors.New("snapshot key must use X25519")
		}
		return material, nil
	}
	const expectedBytes = 32
	encoded := bytes.TrimSpace(raw)
	decoded := make([]byte, base64.RawStdEncoding.DecodedLen(len(encoded)))
	decodedLength, decodeErr := base64.RawStdEncoding.Decode(decoded, encoded)
	if decodeErr != nil {
		decoded = make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
		decodedLength, decodeErr = base64.StdEncoding.Decode(decoded, encoded)
	}
	if decodeErr != nil || decodedLength != expectedBytes {
		for index := range decoded {
			decoded[index] = 0
		}
		return nil, fmt.Errorf("key must be X25519 PKCS#8 PEM or base64-encoded %d-byte material", expectedBytes)
	}
	return decoded[:decodedLength], nil
}
