// Package campusconnectorprotocol defines the narrow, versioned wire contract
// shared by the StuHelper campus-connector gateway and campus-side node.
//
// The contract deliberately cannot represent a generic URL, SQL statement,
// shell command, TCP route, or proxy request. Interactive payloads carry only a
// pre-authorized school-account operation and a single request-scoped password;
// roster payloads are signed and end-to-end encrypted before transport.
package campusconnectorprotocol

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	ProtocolVersion        = "1"
	InteractiveContentType = "application/vnd.stuhelper.connector-interactive-v1"
	RosterSyncContentType  = "application/vnd.stuhelper.connector-roster-sync-v1+json"

	HeaderNodeID          = "X-StuHelper-Connector-Node"
	HeaderProtocolVersion = "X-StuHelper-Connector-Protocol"
	HeaderSigningKeyID    = "X-StuHelper-Connector-Key"
	HeaderTimestamp       = "X-StuHelper-Connector-Timestamp"
	HeaderNonce           = "X-StuHelper-Connector-Nonce"
	HeaderSignature       = "X-StuHelper-Connector-Signature"

	interactiveMagic = "SHCI"
	snapshotKDFInfo  = "stuhelper-campus-connector-snapshot:v1"
)

var (
	ErrInvalidInteractiveDelivery = errors.New("invalid campus connector interactive delivery")
	ErrInvalidSnapshotEnvelope    = errors.New("invalid campus connector snapshot envelope")
	ErrSnapshotSignature          = errors.New("campus connector snapshot signature is invalid")
	ErrSnapshotDecryption         = errors.New("campus connector snapshot decryption failed")
)

type InteractiveMetadata struct {
	RequestID      string    `json:"requestID"`
	SchoolID       int64     `json:"schoolID"`
	SchoolCode     string    `json:"schoolCode"`
	OperationKey   string    `json:"operationKey"`
	AdapterID      string    `json:"adapterID"`
	AdapterVersion string    `json:"adapterVersion"`
	StudentID      string    `json:"studentID"`
	DeadlineAt     time.Time `json:"deadlineAt"`
}

type InteractiveResult struct {
	RequestID      string `json:"requestID"`
	ResultCode     string `json:"resultCode"`
	AccountSubject string `json:"accountSubject,omitempty"`
	StudentID      string `json:"studentID,omitempty"`
}

// RosterSyncCommand is deliberately narrower than a generic remote job.  It
// only selects an operation that already exists in the signed connector node
// configuration and central allowlist; it cannot carry SQL, URLs, routes,
// credentials, or executable arguments.
type RosterSyncCommand struct {
	RequestID      string    `json:"requestID"`
	SchoolCode     string    `json:"schoolCode"`
	OperationKey   string    `json:"operationKey"`
	AdapterID      string    `json:"adapterID"`
	AdapterVersion string    `json:"adapterVersion"`
	DeadlineAt     time.Time `json:"deadlineAt"`
}

type RosterSyncResult struct {
	RequestID  string `json:"requestID"`
	ResultCode string `json:"resultCode"`
}

type OperationHealth struct {
	OperationKey      string `json:"operationKey"`
	SchoolCode        string `json:"schoolCode"`
	OperationType     string `json:"operationType"`
	AdapterID         string `json:"adapterID"`
	AdapterVersion    string `json:"adapterVersion"`
	UpstreamProtocol  string `json:"upstreamProtocol"`
	TargetFingerprint string `json:"targetFingerprint"`
	HealthCode        string `json:"healthCode"`
}

type Heartbeat struct {
	SoftwareVersion string            `json:"softwareVersion"`
	ProtocolVersion string            `json:"protocolVersion"`
	Operations      []OperationHealth `json:"operations"`
}

type SnapshotManifest struct {
	SchemaVersion   int                     `json:"schemaVersion"`
	RequestID       string                  `json:"requestID"`
	NodeID          string                  `json:"nodeID"`
	SchoolCode      string                  `json:"schoolCode"`
	OperationKey    string                  `json:"operationKey"`
	AdapterID       string                  `json:"adapterID"`
	AdapterVersion  string                  `json:"adapterVersion"`
	MappingVersion  string                  `json:"mappingVersion"`
	SourceVersion   string                  `json:"sourceVersion"`
	SourceStartedAt time.Time               `json:"sourceStartedAt"`
	SourceCutoffAt  time.Time               `json:"sourceCutoffAt"`
	RowCount        int64                   `json:"rowCount"`
	PlaintextSHA256 string                  `json:"plaintextSHA256"`
	EncryptionKeyID string                  `json:"encryptionKeyID"`
	SigningKeyID    string                  `json:"signingKeyID"`
	QualitySummary  *SnapshotQualitySummary `json:"qualitySummary,omitempty"`
}

// SnapshotQualitySummary is signed as part of the manifest. It contains only
// aggregate mapping evidence; source identifiers and raw values are never
// included in a connector envelope.
type SnapshotQualitySummary struct {
	RowsRead              int64 `json:"rowsRead"`
	RecordsEmitted        int64 `json:"recordsEmitted"`
	MissingDocumentNumber int64 `json:"missingDocumentNumber"`
	InvalidDocumentNumber int64 `json:"invalidDocumentNumber"`
	MissingPhone          int64 `json:"missingPhone"`
	InvalidPhone          int64 `json:"invalidPhone"`
	MissingEnrollmentYear int64 `json:"missingEnrollmentYear"`
	InvalidEnrollmentYear int64 `json:"invalidEnrollmentYear"`
}

type RosterSnapshotPayload struct {
	Records []RosterRecord `json:"records"`
}

type RosterRecord struct {
	StudentID          string     `json:"studentID"`
	Name               string     `json:"name"`
	DocumentType       string     `json:"documentType,omitempty"`
	DocumentNumber     string     `json:"documentNumber,omitempty"`
	Phone              string     `json:"phone,omitempty"`
	StudentStatus      string     `json:"studentStatus,omitempty"`
	OnCampusStatus     string     `json:"onCampusStatus,omitempty"`
	RegistrationStatus string     `json:"registrationStatus,omitempty"`
	EducationLevel     string     `json:"educationLevel,omitempty"`
	StudentCategory    string     `json:"studentCategory,omitempty"`
	EnrollmentYear     *int       `json:"enrollmentYear,omitempty"`
	ValidFrom          *time.Time `json:"validFrom,omitempty"`
	ValidUntil         *time.Time `json:"validUntil,omitempty"`
	CurrentMarker      *bool      `json:"currentMarker,omitempty"`
	EligibilityCode    string     `json:"eligibilityCode"`
	SourceUpdatedAt    *time.Time `json:"sourceUpdatedAt,omitempty"`
}

type EncryptedSnapshot struct {
	Manifest           SnapshotManifest `json:"manifest"`
	EphemeralPublicKey string           `json:"ephemeralPublicKey"`
	Nonce              string           `json:"nonce"`
	Ciphertext         string           `json:"ciphertext"`
	Signature          string           `json:"signature"`
}

// WriteInteractiveDelivery writes metadata followed by the raw password. The
// password is never converted to a Go string or JSON/base64 value, keeping the
// number of immutable in-memory copies bounded. Callers still own and must wipe
// password after this function returns.
func WriteInteractiveDelivery(w io.Writer, metadata InteractiveMetadata, password []byte) error {
	if len(password) == 0 || len(password) > 1024 {
		return ErrInvalidInteractiveDelivery
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal interactive metadata: %w", err)
	}
	defer wipe(metadataJSON)
	if len(metadataJSON) == 0 || len(metadataJSON) > 16*1024 {
		return ErrInvalidInteractiveDelivery
	}
	metadataLength, ok := checkedUint32Length(metadataJSON)
	if !ok {
		return ErrInvalidInteractiveDelivery
	}
	passwordLength, ok := checkedUint32Length(password)
	if !ok {
		return ErrInvalidInteractiveDelivery
	}
	header := make([]byte, 12)
	copy(header[:4], interactiveMagic)
	binary.BigEndian.PutUint32(header[4:8], metadataLength)
	binary.BigEndian.PutUint32(header[8:12], passwordLength)
	if _, err := w.Write(header); err != nil {
		return err
	}
	if _, err := w.Write(metadataJSON); err != nil {
		return err
	}
	_, err = w.Write(password)
	return err
}

// ReadInteractiveDelivery reads one bounded delivery. The returned password
// aliases a dedicated mutable byte slice and must be wiped by the caller.
func ReadInteractiveDelivery(r io.Reader, maxPasswordBytes int) (InteractiveMetadata, []byte, error) {
	if maxPasswordBytes <= 0 || maxPasswordBytes > 4096 {
		maxPasswordBytes = 1024
	}
	header := make([]byte, 12)
	if _, err := io.ReadFull(r, header); err != nil {
		return InteractiveMetadata{}, nil, ErrInvalidInteractiveDelivery
	}
	if string(header[:4]) != interactiveMagic {
		return InteractiveMetadata{}, nil, ErrInvalidInteractiveDelivery
	}
	metadataLength := int(binary.BigEndian.Uint32(header[4:8]))
	passwordLength := int(binary.BigEndian.Uint32(header[8:12]))
	if metadataLength <= 0 || metadataLength > 16*1024 || passwordLength <= 0 || passwordLength > maxPasswordBytes {
		return InteractiveMetadata{}, nil, ErrInvalidInteractiveDelivery
	}
	metadataJSON := make([]byte, metadataLength)
	if _, err := io.ReadFull(r, metadataJSON); err != nil {
		wipe(metadataJSON)
		return InteractiveMetadata{}, nil, ErrInvalidInteractiveDelivery
	}
	defer wipe(metadataJSON)
	var metadata InteractiveMetadata
	decoder := json.NewDecoder(bytes.NewReader(metadataJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil || metadata.RequestID == "" ||
		metadata.SchoolID <= 0 || metadata.SchoolCode == "" || metadata.OperationKey == "" ||
		metadata.AdapterID == "" || metadata.AdapterVersion == "" || metadata.StudentID == "" ||
		metadata.DeadlineAt.IsZero() {
		return InteractiveMetadata{}, nil, ErrInvalidInteractiveDelivery
	}
	password := make([]byte, passwordLength)
	if _, err := io.ReadFull(r, password); err != nil {
		wipe(password)
		return InteractiveMetadata{}, nil, ErrInvalidInteractiveDelivery
	}
	var trailing [1]byte
	n, err := r.Read(trailing[:])
	if n != 0 || (err != nil && !errors.Is(err, io.EOF)) {
		wipe(password)
		return InteractiveMetadata{}, nil, ErrInvalidInteractiveDelivery
	}
	return metadata, password, nil
}

func RequestBodyDigest(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func RequestSignatureInput(method, requestURI, timestamp, nonce, bodyDigest string) []byte {
	return []byte(strings.Join([]string{
		ProtocolVersion,
		strings.ToUpper(strings.TrimSpace(method)),
		requestURI,
		timestamp,
		nonce,
		bodyDigest,
	}, "\n"))
}

func SignRequest(req *http.Request, nodeID, keyID string, privateKey ed25519.PrivateKey, body []byte, now time.Time, nonce []byte) error {
	if len(privateKey) != ed25519.PrivateKeySize || strings.TrimSpace(nodeID) == "" || strings.TrimSpace(keyID) == "" {
		return errors.New("invalid connector request signing configuration")
	}
	if len(nonce) < 16 {
		return errors.New("connector request nonce is too short")
	}
	timestamp := strconv.FormatInt(now.UTC().Unix(), 10)
	nonceValue := base64.RawURLEncoding.EncodeToString(nonce)
	digest := RequestBodyDigest(body)
	signature := ed25519.Sign(privateKey, RequestSignatureInput(req.Method, req.URL.RequestURI(), timestamp, nonceValue, digest))
	req.Header.Set(HeaderNodeID, nodeID)
	req.Header.Set(HeaderProtocolVersion, ProtocolVersion)
	req.Header.Set(HeaderSigningKeyID, keyID)
	req.Header.Set(HeaderTimestamp, timestamp)
	req.Header.Set(HeaderNonce, nonceValue)
	req.Header.Set(HeaderSignature, base64.RawURLEncoding.EncodeToString(signature))
	return nil
}

func NewRequestNonce() ([]byte, error) {
	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return nonce, nil
}

func VerifyRequestSignature(req *http.Request, publicKey ed25519.PublicKey, body []byte) bool {
	if len(publicKey) != ed25519.PublicKeySize || req.Header.Get(HeaderProtocolVersion) != ProtocolVersion {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(req.Header.Get(HeaderSignature))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return false
	}
	input := RequestSignatureInput(
		req.Method,
		req.URL.RequestURI(),
		req.Header.Get(HeaderTimestamp),
		req.Header.Get(HeaderNonce),
		RequestBodyDigest(body),
	)
	return ed25519.Verify(publicKey, input, signature)
}

func EncryptSnapshot(
	manifest SnapshotManifest,
	plaintext []byte,
	recipientPublicKey []byte,
	signingPrivateKey ed25519.PrivateKey,
) (*EncryptedSnapshot, error) {
	if len(plaintext) == 0 || len(recipientPublicKey) != 32 || len(signingPrivateKey) != ed25519.PrivateKeySize {
		return nil, ErrInvalidSnapshotEnvelope
	}
	plaintextDigest := sha256.Sum256(plaintext)
	manifest.PlaintextSHA256 = hex.EncodeToString(plaintextDigest[:])
	curve := ecdh.X25519()
	recipient, err := curve.NewPublicKey(recipientPublicKey)
	if err != nil {
		return nil, ErrInvalidSnapshotEnvelope
	}
	ephemeral, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate snapshot ephemeral key: %w", err)
	}
	sharedSecret, err := ephemeral.ECDH(recipient)
	if err != nil {
		return nil, ErrInvalidSnapshotEnvelope
	}
	defer wipe(sharedSecret)
	key, err := hkdf.Key(sha256.New, sharedSecret, nil, snapshotKDFInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("derive snapshot encryption key: %w", err)
	}
	defer wipe(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	defer wipe(manifestJSON)
	ciphertext := aead.Seal(nil, nonce, plaintext, manifestJSON)
	envelope := &EncryptedSnapshot{
		Manifest:           manifest,
		EphemeralPublicKey: base64.RawStdEncoding.EncodeToString(ephemeral.PublicKey().Bytes()),
		Nonce:              base64.RawStdEncoding.EncodeToString(nonce),
		Ciphertext:         base64.RawStdEncoding.EncodeToString(ciphertext),
	}
	signatureInput, err := snapshotSignatureInput(*envelope)
	if err != nil {
		return nil, err
	}
	defer wipe(signatureInput)
	envelope.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(signingPrivateKey, signatureInput))
	return envelope, nil
}

func DecryptSnapshot(
	envelope EncryptedSnapshot,
	recipientPrivateKey []byte,
	signingPublicKey ed25519.PublicKey,
	maxPlaintextBytes int,
) ([]byte, error) {
	if len(recipientPrivateKey) != 32 || len(signingPublicKey) != ed25519.PublicKeySize ||
		envelope.Manifest.SchemaVersion <= 0 || envelope.Manifest.PlaintextSHA256 == "" {
		return nil, ErrInvalidSnapshotEnvelope
	}
	signature, err := base64.RawStdEncoding.DecodeString(envelope.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return nil, ErrSnapshotSignature
	}
	signatureInput, err := snapshotSignatureInput(envelope)
	if err != nil {
		return nil, err
	}
	defer wipe(signatureInput)
	if !ed25519.Verify(signingPublicKey, signatureInput, signature) {
		return nil, ErrSnapshotSignature
	}
	ephemeralBytes, err := base64.RawStdEncoding.DecodeString(envelope.EphemeralPublicKey)
	if err != nil || len(ephemeralBytes) != 32 {
		return nil, ErrInvalidSnapshotEnvelope
	}
	nonce, err := base64.RawStdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, ErrInvalidSnapshotEnvelope
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil || len(ciphertext) == 0 {
		return nil, ErrInvalidSnapshotEnvelope
	}
	defer wipe(ciphertext)
	if maxPlaintextBytes > 0 && len(ciphertext) > maxPlaintextBytes+64 {
		return nil, ErrInvalidSnapshotEnvelope
	}
	curve := ecdh.X25519()
	privateKey, err := curve.NewPrivateKey(recipientPrivateKey)
	if err != nil {
		return nil, ErrInvalidSnapshotEnvelope
	}
	ephemeral, err := curve.NewPublicKey(ephemeralBytes)
	if err != nil {
		return nil, ErrInvalidSnapshotEnvelope
	}
	sharedSecret, err := privateKey.ECDH(ephemeral)
	if err != nil {
		return nil, ErrSnapshotDecryption
	}
	defer wipe(sharedSecret)
	key, err := hkdf.Key(sha256.New, sharedSecret, nil, snapshotKDFInfo, 32)
	if err != nil {
		return nil, ErrSnapshotDecryption
	}
	defer wipe(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrSnapshotDecryption
	}
	aead, err := cipher.NewGCM(block)
	if err != nil || len(nonce) != aead.NonceSize() {
		return nil, ErrSnapshotDecryption
	}
	manifestJSON, err := json.Marshal(envelope.Manifest)
	if err != nil {
		return nil, ErrInvalidSnapshotEnvelope
	}
	defer wipe(manifestJSON)
	plaintext, err := aead.Open(nil, nonce, ciphertext, manifestJSON)
	if err != nil {
		return nil, ErrSnapshotDecryption
	}
	if maxPlaintextBytes > 0 && len(plaintext) > maxPlaintextBytes {
		wipe(plaintext)
		return nil, ErrInvalidSnapshotEnvelope
	}
	digest := sha256.Sum256(plaintext)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), envelope.Manifest.PlaintextSHA256) {
		wipe(plaintext)
		return nil, ErrSnapshotDecryption
	}
	return plaintext, nil
}

func snapshotSignatureInput(envelope EncryptedSnapshot) ([]byte, error) {
	manifestJSON, err := json.Marshal(envelope.Manifest)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(envelope.Ciphertext))
	return bytes.Join([][]byte{
		[]byte("stuhelper-campus-connector-snapshot-signature:v1"),
		manifestJSON,
		[]byte(envelope.EphemeralPublicKey),
		[]byte(envelope.Nonce),
		[]byte(hex.EncodeToString(digest[:])),
	}, []byte{0}), nil
}

func OperationTargetFingerprint(protocol, host string, port int, tlsServerName string) string {
	value := strings.Join([]string{
		strings.ToLower(strings.TrimSpace(protocol)),
		strings.ToLower(strings.TrimSpace(host)),
		strconv.Itoa(port),
		strings.ToLower(strings.TrimSpace(tlsServerName)),
	}, "\x00")
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func wipe(value []byte) {
	for i := range value {
		value[i] = 0
	}
}

func checkedUint32Length(value []byte) (uint32, bool) {
	length := len(value)
	if uint64(length) > math.MaxUint32 {
		return 0, false
	}
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], uint64(length))
	return binary.BigEndian.Uint32(encoded[4:]), true
}
