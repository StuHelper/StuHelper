package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	connectorprotocol "github.com/StuHelper/StuHelper/server/internal/pkg/campusconnectorprotocol"
)

var (
	operationKeyPattern     = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)
	adapterIDPattern        = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	oracleIdentifierPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,127}$`)
	ldapAttributePattern    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]{0,63}$`)
	schoolCodePattern       = regexp.MustCompile(`^\d{10}$`)
)

type Config struct {
	NodeID                      string            `json:"nodeID"`
	SoftwareVersion             string            `json:"softwareVersion"`
	CentralBaseURL              string            `json:"centralBaseURL"`
	ClientCertificateFile       string            `json:"clientCertificateFile"`
	ClientPrivateKeyFile        string            `json:"clientPrivateKeyFile"`
	CentralCAFile               string            `json:"centralCAFile"`
	SigningKeyID                string            `json:"signingKeyID"`
	SigningPrivateKeyFile       string            `json:"signingPrivateKeyFile"`
	SnapshotEncryptionKeyID     string            `json:"snapshotEncryptionKeyID"`
	SnapshotPublicKeyFile       string            `json:"snapshotPublicKeyFile"`
	ProtocolVersion             string            `json:"protocolVersion"`
	HeartbeatIntervalSeconds    int               `json:"heartbeatIntervalSeconds"`
	PollWorkers                 int               `json:"pollWorkers"`
	MaxInteractivePasswordBytes int               `json:"maxInteractivePasswordBytes"`
	Operations                  []OperationConfig `json:"operations"`
}

type OperationConfig struct {
	Key                 string                       `json:"key"`
	SchoolCode          string                       `json:"schoolCode"`
	Type                string                       `json:"type"`
	AdapterID           string                       `json:"adapterID"`
	AdapterVersion      string                       `json:"adapterVersion"`
	UpstreamProtocol    string                       `json:"upstreamProtocol"`
	TargetHost          string                       `json:"targetHost"`
	TargetPort          int                          `json:"targetPort"`
	TLSServerName       string                       `json:"tlsServerName"`
	TimeoutMilliseconds int                          `json:"timeoutMilliseconds"`
	LDAP                *LDAPOperationConfig         `json:"ldap,omitempty"`
	OracleRoster        *OracleRosterOperationConfig `json:"oracleRoster,omitempty"`
}

type LDAPOperationConfig struct {
	CAFile                       string   `json:"caFile"`
	AllowPlaintextPrivateNetwork bool     `json:"allowPlaintextPrivateNetwork,omitempty"`
	UserDNTemplate               string   `json:"userDNTemplate"`
	SearchBaseDN                 string   `json:"searchBaseDN"`
	SystemBindDN                 string   `json:"systemBindDN"`
	SystemBindPasswordFile       string   `json:"systemBindPasswordFile"`
	UIDAttribute                 string   `json:"uidAttribute"`
	SubjectAttribute             string   `json:"subjectAttribute"`
	AccountLockedAttribute       string   `json:"accountLockedAttribute"`
	AccountDisabledAttribute     string   `json:"accountDisabledAttribute"`
	StudentMarkerAttribute       string   `json:"studentMarkerAttribute,omitempty"`
	StudentMarkerValues          []string `json:"studentMarkerValues,omitempty"`
}

type OracleRosterOperationConfig struct {
	ServiceName                 string              `json:"serviceName"`
	UsernameFile                string              `json:"usernameFile"`
	PasswordFile                string              `json:"passwordFile"`
	CAFile                      string              `json:"caFile"`
	AllowPlaintextSSHTunnel     bool                `json:"allowPlaintextSSHTunnel,omitempty"`
	AllowedDialTargets          []string            `json:"allowedDialTargets"`
	ExpectedUsername            string              `json:"expectedUsername"`
	Schema                      string              `json:"schema"`
	Table                       string              `json:"table"`
	ActiveFilterColumn          string              `json:"activeFilterColumn"`
	ActiveFilterValue           string              `json:"activeFilterValue"`
	ActiveEligibilityCode       string              `json:"activeEligibilityCode"`
	Columns                     OracleRosterColumns `json:"columns"`
	FullSnapshotIntervalMinutes int                 `json:"fullSnapshotIntervalMinutes"`
	MappingVersion              string              `json:"mappingVersion"`
	DefaultDocumentType         string              `json:"defaultDocumentType,omitempty"`
	MaximumRows                 int                 `json:"maximumRows"`
}

type OracleRosterColumns struct {
	StudentID          string `json:"studentID"`
	Name               string `json:"name"`
	DocumentType       string `json:"documentType"`
	DocumentNumber     string `json:"documentNumber"`
	Phone              string `json:"phone"`
	StudentStatus      string `json:"studentStatus"`
	OnCampusStatus     string `json:"onCampusStatus"`
	RegistrationStatus string `json:"registrationStatus"`
	EducationLevel     string `json:"educationLevel"`
	StudentCategory    string `json:"studentCategory"`
	EnrollmentYear     string `json:"enrollmentYear"`
	ValidFrom          string `json:"validFrom"`
	ValidUntil         string `json:"validUntil"`
	CurrentMarker      string `json:"currentMarker"`
	EligibilityCode    string `json:"eligibilityCode"`
	SourceUpdatedAt    string `json:"sourceUpdatedAt"`
}

func LoadConfig(path string) (Config, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Config{}, errors.New("campus connector node config file is required")
	}
	// #nosec G304 -- the path is a local process startup argument, never request input.
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read campus connector node config: %w", err)
	}
	defer wipe(raw)
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode campus connector node config: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Config{}, errors.New("campus connector node config contains trailing JSON")
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) Validate() error {
	if _, err := uuid.Parse(cfg.NodeID); err != nil {
		return errors.New("nodeID must be a UUID")
	}
	if strings.TrimSpace(cfg.SoftwareVersion) == "" || len(cfg.SoftwareVersion) > 128 {
		return errors.New("softwareVersion is required and must be at most 128 characters")
	}
	centralURL, err := url.Parse(cfg.CentralBaseURL)
	if err != nil || centralURL.Scheme != "https" || centralURL.Host == "" ||
		centralURL.User != nil || centralURL.RawQuery != "" || centralURL.Fragment != "" ||
		(centralURL.Path != "" && centralURL.Path != "/") {
		return errors.New("centralBaseURL must be an HTTPS origin without credentials, path, query, or fragment")
	}
	for name, value := range map[string]string{
		"clientCertificateFile":   cfg.ClientCertificateFile,
		"clientPrivateKeyFile":    cfg.ClientPrivateKeyFile,
		"centralCAFile":           cfg.CentralCAFile,
		"signingKeyID":            cfg.SigningKeyID,
		"signingPrivateKeyFile":   cfg.SigningPrivateKeyFile,
		"snapshotEncryptionKeyID": cfg.SnapshotEncryptionKeyID,
		"snapshotPublicKeyFile":   cfg.SnapshotPublicKeyFile,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if cfg.ProtocolVersion != connectorprotocol.ProtocolVersion {
		return errors.New("unsupported protocolVersion")
	}
	if cfg.HeartbeatIntervalSeconds < 5 || cfg.HeartbeatIntervalSeconds > 3600 {
		return errors.New("heartbeatIntervalSeconds must be between 5 and 3600")
	}
	if cfg.PollWorkers < 1 || cfg.PollWorkers > 128 {
		return errors.New("pollWorkers must be between 1 and 128")
	}
	if cfg.MaxInteractivePasswordBytes < 64 || cfg.MaxInteractivePasswordBytes > 4096 {
		return errors.New("maxInteractivePasswordBytes must be between 64 and 4096")
	}
	if len(cfg.Operations) == 0 || len(cfg.Operations) > 256 {
		return errors.New("operations must contain between 1 and 256 entries")
	}
	seen := make(map[string]struct{}, len(cfg.Operations))
	for index := range cfg.Operations {
		operation := cfg.Operations[index]
		if _, duplicate := seen[operation.Key]; duplicate {
			return fmt.Errorf("operation %q is duplicated", operation.Key)
		}
		seen[operation.Key] = struct{}{}
		if err := operation.Validate(); err != nil {
			return fmt.Errorf("operation %q: %w", operation.Key, err)
		}
	}
	return nil
}

func (cfg OperationConfig) Validate() error {
	if !operationKeyPattern.MatchString(cfg.Key) || !schoolCodePattern.MatchString(cfg.SchoolCode) ||
		!adapterIDPattern.MatchString(cfg.AdapterID) ||
		strings.TrimSpace(cfg.AdapterVersion) == "" || len(cfg.AdapterVersion) > 64 {
		return errors.New("operation key or adapter identity is invalid")
	}
	if cfg.TargetPort < 1 || cfg.TargetPort > 65535 || strings.TrimSpace(cfg.TargetHost) == "" ||
		strings.ContainsAny(cfg.TargetHost, "/?#@") {
		return errors.New("targetHost and targetPort must identify one exact endpoint")
	}
	if strings.ContainsAny(cfg.TLSServerName, "/?#@") {
		return errors.New("tlsServerName contains invalid URL characters")
	}
	if cfg.TimeoutMilliseconds < 100 || cfg.TimeoutMilliseconds > 120000 {
		return errors.New("timeoutMilliseconds must be between 100 and 120000")
	}
	switch cfg.Type {
	case "school_account_authenticate":
		if !slices.Contains([]string{"ldaps", "ldap_starttls", "ldap_plain_private_network"}, cfg.UpstreamProtocol) ||
			cfg.LDAP == nil || cfg.OracleRoster != nil {
			return errors.New("school account operation requires exactly one approved LDAP configuration")
		}
		switch cfg.UpstreamProtocol {
		case "ldap_plain_private_network":
			if !cfg.LDAP.AllowPlaintextPrivateNetwork || cfg.TargetPort != 389 ||
				net.ParseIP(strings.TrimSpace(cfg.TargetHost)) == nil ||
				!isRFC1918IPv4(net.ParseIP(strings.TrimSpace(cfg.TargetHost))) ||
				strings.TrimSpace(cfg.TLSServerName) != "" || strings.TrimSpace(cfg.LDAP.CAFile) != "" {
				return errors.New("plaintext LDAP requires an explicit RFC1918 IPv4 risk flag, port 389, and no TLS name")
			}
		case "ldaps", "ldap_starttls":
			if net.ParseIP(strings.TrimSpace(cfg.TLSServerName)) != nil || strings.TrimSpace(cfg.TLSServerName) == "" {
				return errors.New("encrypted LDAP requires a non-IP TLS certificate name")
			}
			if strings.TrimSpace(cfg.LDAP.CAFile) == "" {
				return errors.New("encrypted LDAP requires a CA file")
			}
		}
		return cfg.LDAP.Validate()
	case "roster_snapshot_upload":
		if !slices.Contains([]string{"oracle_tls", "oracle_ssh_tunnel"}, cfg.UpstreamProtocol) ||
			cfg.OracleRoster == nil || cfg.LDAP != nil {
			return errors.New("roster operation requires exactly one approved Oracle configuration")
		}
		switch cfg.UpstreamProtocol {
		case "oracle_tls":
			if net.ParseIP(strings.TrimSpace(cfg.TLSServerName)) != nil || strings.TrimSpace(cfg.TLSServerName) == "" {
				return errors.New("oracle TLS roster operation requires a non-IP certificate name")
			}
			if strings.TrimSpace(cfg.OracleRoster.CAFile) == "" || cfg.OracleRoster.AllowPlaintextSSHTunnel {
				return errors.New("oracle TLS roster operation requires a CA file and forbids the SSH tunnel risk flag")
			}
		case "oracle_ssh_tunnel":
			hostIP := net.ParseIP(strings.TrimSpace(cfg.TargetHost))
			if !cfg.OracleRoster.AllowPlaintextSSHTunnel || hostIP == nil || !hostIP.IsLoopback() ||
				cfg.TargetPort < 1024 || strings.TrimSpace(cfg.TLSServerName) != "" ||
				strings.TrimSpace(cfg.OracleRoster.CAFile) != "" {
				return errors.New("oracle SSH tunnel requires an explicit risk flag, a loopback high port, and no Oracle TLS fields")
			}
		}
		if err := cfg.OracleRoster.Validate(); err != nil {
			return err
		}
		return cfg.OracleRoster.ValidateDialTargets(cfg.TargetHost, cfg.TargetPort)
	default:
		return errors.New("operation type is not allowed")
	}
}

func (cfg LDAPOperationConfig) Validate() error {
	if strings.Count(cfg.UserDNTemplate, "{student_id}") != 1 ||
		strings.Contains(strings.ReplaceAll(cfg.UserDNTemplate, "{student_id}", ""), "{") ||
		strings.TrimSpace(cfg.SearchBaseDN) == "" || strings.TrimSpace(cfg.SystemBindDN) == "" ||
		!validSecretFileReference(cfg.SystemBindPasswordFile) {
		return errors.New("LDAP DN templates or system secret reference are invalid")
	}
	attributes := []string{cfg.UIDAttribute, cfg.SubjectAttribute,
		cfg.AccountLockedAttribute, cfg.AccountDisabledAttribute}
	if cfg.StudentMarkerAttribute != "" {
		attributes = append(attributes, cfg.StudentMarkerAttribute)
		if len(cfg.StudentMarkerValues) == 0 || len(cfg.StudentMarkerValues) > 32 {
			return errors.New("student marker values are required when the marker attribute is configured")
		}
	}
	for _, attribute := range attributes {
		if !ldapAttributePattern.MatchString(attribute) {
			return errors.New("LDAP attribute name is invalid")
		}
	}
	return nil
}

func isRFC1918IPv4(ip net.IP) bool {
	ip = ip.To4()
	if ip == nil {
		return false
	}
	return ip[0] == 10 ||
		(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
		(ip[0] == 192 && ip[1] == 168)
}

func (cfg OracleRosterOperationConfig) Validate() error {
	if strings.TrimSpace(cfg.ServiceName) == "" || !validSecretFileReference(cfg.UsernameFile) ||
		!validSecretFileReference(cfg.PasswordFile) || cfg.UsernameFile == cfg.PasswordFile ||
		len(cfg.AllowedDialTargets) == 0 || len(cfg.AllowedDialTargets) > 64 ||
		!oracleIdentifierPattern.MatchString(cfg.ExpectedUsername) ||
		!oracleIdentifierPattern.MatchString(cfg.Schema) || !oracleIdentifierPattern.MatchString(cfg.Table) ||
		!oracleIdentifierPattern.MatchString(cfg.ActiveFilterColumn) ||
		strings.TrimSpace(cfg.ActiveFilterValue) == "" || len(cfg.ActiveFilterValue) > 128 ||
		strings.TrimSpace(cfg.ActiveEligibilityCode) == "" || len(cfg.ActiveEligibilityCode) > 128 ||
		strings.TrimSpace(cfg.MappingVersion) == "" || len(cfg.MappingVersion) > 64 ||
		cfg.FullSnapshotIntervalMinutes < 5 || cfg.FullSnapshotIntervalMinutes > 10080 ||
		cfg.MaximumRows < 1 || cfg.MaximumRows > 2000000 || len(cfg.DefaultDocumentType) > 64 {
		return errors.New("oracle connection, secret reference, mapping, or schedule is invalid")
	}
	required := []string{
		cfg.Columns.StudentID, cfg.Columns.Name,
		cfg.Columns.CurrentMarker, cfg.Columns.EligibilityCode,
	}
	for _, column := range required {
		if !oracleIdentifierPattern.MatchString(column) {
			return errors.New("required oracle roster column is invalid")
		}
	}
	optional := []string{
		cfg.Columns.DocumentType, cfg.Columns.DocumentNumber, cfg.Columns.Phone, cfg.Columns.StudentStatus,
		cfg.Columns.OnCampusStatus, cfg.Columns.RegistrationStatus,
		cfg.Columns.EducationLevel, cfg.Columns.StudentCategory, cfg.Columns.EnrollmentYear, cfg.Columns.ValidFrom,
		cfg.Columns.ValidUntil, cfg.Columns.SourceUpdatedAt,
	}
	for _, column := range optional {
		if column != "" && !oracleIdentifierPattern.MatchString(column) {
			return errors.New("optional Oracle roster column is invalid")
		}
	}
	filterApproved := false
	for _, column := range []string{
		cfg.Columns.OnCampusStatus,
		cfg.Columns.CurrentMarker,
		cfg.Columns.EligibilityCode,
	} {
		if column != "" && strings.EqualFold(column, cfg.ActiveFilterColumn) {
			filterApproved = true
			break
		}
	}
	if !filterApproved {
		return errors.New("oracle active filter must use an approved status column")
	}
	return nil
}

// validSecretFileReference limits upstream credentials to one exact file under
// the connector's read-only secret mount. This prevents the non-secret node
// configuration from becoming an arbitrary local-file reader.
func validSecretFileReference(path string) bool {
	trimmed := strings.TrimSpace(path)
	if path == "" || path != trimmed || path != filepath.Clean(path) ||
		!filepath.IsAbs(path) || strings.ContainsRune(path, '\x00') {
		return false
	}
	relative, err := filepath.Rel(secretFileRoot, path)
	return err == nil && relative != "." && relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func (cfg OracleRosterOperationConfig) ValidateDialTargets(initialHost string, initialPort int) error {
	approved := make(map[string]struct{}, len(cfg.AllowedDialTargets))
	for _, raw := range cfg.AllowedDialTargets {
		target, err := normalizeOracleDialTarget(raw, cfg.AllowPlaintextSSHTunnel)
		if err != nil {
			return err
		}
		approved[target] = struct{}{}
	}
	initial, err := normalizeOracleDialTarget(
		net.JoinHostPort(initialHost, strconv.Itoa(initialPort)), cfg.AllowPlaintextSSHTunnel,
	)
	if err != nil {
		return err
	}
	if _, ok := approved[initial]; !ok {
		return errors.New("allowedDialTargets must include the configured initial Oracle endpoint")
	}
	if cfg.AllowPlaintextSSHTunnel && (len(approved) != 1 || len(cfg.AllowedDialTargets) != 1) {
		return errors.New("oracle SSH tunnel must allow exactly one loopback endpoint and no listener redirects")
	}
	return nil
}

func normalizeOracleDialTarget(raw string, allowLoopback bool) (string, error) {
	host, rawPort, err := net.SplitHostPort(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("oracle dial target must be an exact host:port endpoint")
	}
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 || host == "" || len(host) > 253 ||
		host == "localhost" || strings.ContainsAny(host, "/?#@%\\") ||
		strings.ContainsFunc(host, unicode.IsSpace) {
		return "", errors.New("oracle dial target is invalid")
	}
	if parsed := net.ParseIP(host); parsed != nil &&
		((parsed.IsLoopback() && !allowLoopback) || parsed.IsUnspecified() || parsed.IsMulticast()) {
		return "", errors.New("oracle dial target cannot be loopback, unspecified, or multicast")
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

func (cfg Config) HeartbeatInterval() time.Duration {
	return time.Duration(cfg.HeartbeatIntervalSeconds) * time.Second
}

func (cfg Config) Operation(key string) (OperationConfig, bool) {
	for _, operation := range cfg.Operations {
		if operation.Key == key {
			return operation, true
		}
	}
	return OperationConfig{}, false
}

func wipe(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
