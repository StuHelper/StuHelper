package node

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	connectorprotocol "github.com/StuHelper/StuHelper/server/internal/pkg/campusconnectorprotocol"
)

type Client struct {
	cfg               Config
	http              *http.Client
	signingPrivateKey ed25519.PrivateKey
	snapshotPublicKey []byte
}

type PollDelivery struct {
	Interactive *connectorprotocol.InteractiveMetadata
	Password    []byte
	RosterSync  *connectorprotocol.RosterSyncCommand
}

func NewClient(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	certificate, err := tls.LoadX509KeyPair(cfg.ClientCertificateFile, cfg.ClientPrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load connector client certificate: %w", err)
	}
	// #nosec G304 -- this path is validated startup configuration and is never derived from a request.
	caPEM, err := os.ReadFile(cfg.CentralCAFile)
	if err != nil {
		return nil, fmt.Errorf("read connector central CA: %w", err)
	}
	centralCAs := x509.NewCertPool()
	if !centralCAs.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("connector central CA file contains no certificate")
	}
	parsedURL, err := url.Parse(cfg.CentralBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse connector central URL: %w", err)
	}
	signingKeyBytes, err := readConnectorKey(cfg.SigningPrivateKeyFile, ed25519.PrivateKeySize)
	if err != nil {
		return nil, fmt.Errorf("load connector signing private key: %w", err)
	}
	snapshotPublicKey, err := readConnectorKey(cfg.SnapshotPublicKeyFile, 32)
	if err != nil {
		wipe(signingKeyBytes)
		return nil, fmt.Errorf("load snapshot recipient public key: %w", err)
	}
	transport := &http.Transport{
		Proxy: nil,
		TLSClientConfig: &tls.Config{
			MinVersion:   tls.VersionTLS13,
			ServerName:   parsedURL.Hostname(),
			RootCAs:      centralCAs,
			Certificates: []tls.Certificate{certificate},
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          cfg.PollWorkers + 4,
		MaxIdleConnsPerHost:   cfg.PollWorkers + 4,
		IdleConnTimeout:       70 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 3*time.Minute + 15*time.Second,
	}
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Transport: transport,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return errors.New("campus connector redirects are forbidden")
			},
		},
		signingPrivateKey: ed25519.PrivateKey(signingKeyBytes),
		snapshotPublicKey: snapshotPublicKey,
	}, nil
}

func (c *Client) Close() {
	if transport, ok := c.http.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	wipe(c.signingPrivateKey)
	wipe(c.snapshotPublicKey)
	c.signingPrivateKey = nil
	c.snapshotPublicKey = nil
}

func (c *Client) PostHeartbeat(ctx context.Context, heartbeat connectorprotocol.Heartbeat) error {
	body, err := json.Marshal(heartbeat)
	if err != nil {
		return err
	}
	defer wipe(body)
	response, err := c.doSigned(ctx, http.MethodPost, "/v1/heartbeat", body, "application/json")
	if err != nil {
		return err
	}
	statusCode := response.StatusCode
	closeErr := drainAndCloseResponse(response)
	if statusCode != http.StatusNoContent {
		return errors.Join(fmt.Errorf("heartbeat rejected with status %d", statusCode), closeErr)
	}
	return closeErr
}

func (c *Client) Poll(ctx context.Context) (*PollDelivery, error) {
	response, err := c.doSigned(ctx, http.MethodPost, "/v1/poll", nil, "")
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusNoContent {
		return nil, drainAndCloseResponse(response)
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.Join(
			fmt.Errorf("poll rejected with status %d", response.StatusCode),
			drainAndCloseResponse(response),
		)
	}
	contentType := response.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, connectorprotocol.RosterSyncContentType) {
		body, readErr := io.ReadAll(io.LimitReader(response.Body, (16<<10)+1))
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil || len(body) > 16<<10 {
			wipe(body)
			return nil, errors.Join(errors.New("invalid roster sync command body"), readErr, closeErr)
		}
		defer wipe(body)
		var command connectorprotocol.RosterSyncCommand
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&command); err != nil ||
			!errors.Is(decoder.Decode(&struct{}{}), io.EOF) ||
			strings.TrimSpace(command.RequestID) == "" ||
			strings.TrimSpace(command.SchoolCode) == "" ||
			strings.TrimSpace(command.OperationKey) == "" ||
			strings.TrimSpace(command.AdapterID) == "" ||
			strings.TrimSpace(command.AdapterVersion) == "" || command.DeadlineAt.IsZero() {
			return nil, errors.New("invalid roster sync command")
		}
		return &PollDelivery{RosterSync: &command}, nil
	}
	if !strings.HasPrefix(contentType, connectorprotocol.InteractiveContentType) {
		return nil, errors.Join(
			errors.New("poll returned an unsupported delivery type"),
			drainAndCloseResponse(response),
		)
	}
	metadata, password, err := connectorprotocol.ReadInteractiveDelivery(
		io.LimitReader(response.Body, int64(c.cfg.MaxInteractivePasswordBytes+(20<<10))),
		c.cfg.MaxInteractivePasswordBytes,
	)
	closeErr := response.Body.Close()
	if err != nil {
		wipe(password)
		return nil, errors.Join(err, closeErr)
	}
	if closeErr != nil {
		wipe(password)
		return nil, fmt.Errorf("close connector poll response: %w", closeErr)
	}
	return &PollDelivery{Interactive: &metadata, Password: password}, nil
}

func (c *Client) PostResult(ctx context.Context, result connectorprotocol.InteractiveResult) error {
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	defer wipe(body)
	response, err := c.doSigned(ctx, http.MethodPost, "/v1/results/"+url.PathEscape(result.RequestID), body, "application/json")
	if err != nil {
		return err
	}
	statusCode := response.StatusCode
	closeErr := drainAndCloseResponse(response)
	if statusCode != http.StatusNoContent && statusCode != http.StatusGone {
		return errors.Join(fmt.Errorf("connector result rejected with status %d", statusCode), closeErr)
	}
	return closeErr
}

func (c *Client) PostRosterResult(ctx context.Context, result connectorprotocol.RosterSyncResult) error {
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	defer wipe(body)
	response, err := c.doSigned(
		ctx,
		http.MethodPost,
		"/v1/roster-results/"+url.PathEscape(result.RequestID),
		body,
		"application/json",
	)
	if err != nil {
		return err
	}
	statusCode := response.StatusCode
	closeErr := drainAndCloseResponse(response)
	if statusCode != http.StatusNoContent && statusCode != http.StatusGone {
		return errors.Join(fmt.Errorf("roster result rejected with status %d", statusCode), closeErr)
	}
	return closeErr
}

func (c *Client) UploadSnapshot(ctx context.Context, envelope connectorprotocol.EncryptedSnapshot) error {
	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	defer wipe(body)
	response, err := c.doSigned(ctx, http.MethodPost, "/v1/snapshots", body, "application/json")
	if err != nil {
		return err
	}
	statusCode := response.StatusCode
	closeErr := drainAndCloseResponse(response)
	if statusCode != http.StatusCreated {
		return errors.Join(fmt.Errorf("snapshot upload rejected with status %d", statusCode), closeErr)
	}
	return closeErr
}

func (c *Client) EncryptSnapshot(
	manifest connectorprotocol.SnapshotManifest,
	plaintext []byte,
) (*connectorprotocol.EncryptedSnapshot, error) {
	return connectorprotocol.EncryptSnapshot(
		manifest, plaintext, c.snapshotPublicKey, c.signingPrivateKey,
	)
}

func (c *Client) doSigned(
	ctx context.Context,
	method string,
	path string,
	body []byte,
	contentType string,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(
		ctx, method, strings.TrimRight(c.cfg.CentralBaseURL, "/")+path, bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cache-Control", "no-store")
	nonce, err := connectorprotocol.NewRequestNonce()
	if err != nil {
		return nil, err
	}
	defer wipe(nonce)
	if err := connectorprotocol.SignRequest(
		request, c.cfg.NodeID, c.cfg.SigningKeyID,
		c.signingPrivateKey, body, time.Now().UTC(), nonce,
	); err != nil {
		return nil, err
	}
	return c.http.Do(request)
}

// readConnectorKey accepts the legacy raw Base64 representation and the
// standard PKCS#8/PKIX PEM files produced by the production PKI generator.
// expectedLength disambiguates the only two key roles used by the node:
// Ed25519 private signing keys (64 bytes) and X25519 public recipient keys
// (32 bytes).
func readConnectorKey(path string, expectedLength int) ([]byte, error) {
	// #nosec G304 -- key paths are validated startup configuration and are never request-controlled.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer wipe(raw)
	if block, _ := pem.Decode(raw); block != nil {
		var material []byte
		switch expectedLength {
		case ed25519.PrivateKeySize:
			parsed, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
			key, ok := parsed.(ed25519.PrivateKey)
			if parseErr != nil || !ok {
				return nil, errors.New("connector signing key must be an Ed25519 PKCS#8 private key")
			}
			material = key
		case 32:
			parsed, parseErr := x509.ParsePKIXPublicKey(block.Bytes)
			key, ok := parsed.(*ecdh.PublicKey)
			if parseErr != nil || !ok || key.Curve() != ecdh.X25519() {
				return nil, errors.New("snapshot recipient key must be an X25519 PKIX public key")
			}
			material = key.Bytes()
		default:
			return nil, errors.New("unsupported connector key role")
		}
		if len(material) != expectedLength {
			return nil, fmt.Errorf("connector key material must contain %d bytes", expectedLength)
		}
		return append([]byte(nil), material...), nil
	}
	encoded := bytes.TrimSpace(raw)
	decoded := make([]byte, base64.RawStdEncoding.DecodedLen(len(encoded)))
	decodedLength, decodeErr := base64.RawStdEncoding.Decode(decoded, encoded)
	if decodeErr != nil {
		decoded = make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
		decodedLength, decodeErr = base64.StdEncoding.Decode(decoded, encoded)
	}
	if decodeErr != nil || decodedLength != expectedLength {
		wipe(decoded)
		return nil, fmt.Errorf("key must be supported PEM or base64-encoded %d-byte material", expectedLength)
	}
	return decoded[:decodedLength], nil
}

func drainAndCloseResponse(response *http.Response) error {
	if response == nil || response.Body == nil {
		return nil
	}
	_, copyErr := io.Copy(io.Discard, io.LimitReader(response.Body, 16<<10))
	closeErr := response.Body.Close()
	if copyErr != nil {
		copyErr = fmt.Errorf("drain connector response: %w", copyErr)
	}
	if closeErr != nil {
		closeErr = fmt.Errorf("close connector response: %w", closeErr)
	}
	return errors.Join(copyErr, closeErr)
}
