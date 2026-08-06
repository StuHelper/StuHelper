package campusconnector

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	registryUUIDPattern       = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	registrySchoolCodePattern = regexp.MustCompile(`^\d{10}$`)
	registryOperationPattern  = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)
	registryAdapterPattern    = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
)

type RegistryManifest struct {
	Node       RegistryNode
	Operations []RegistryOperation
	Reason     string
}

type RegistryNode struct {
	ID                       string
	DisplayName              string
	ProtocolVersion          string
	SoftwareVersion          string
	CertificateFingerprint   string
	CertificateNotAfter      time.Time
	SigningKeyID             string
	SigningPublicKey         []byte
	MaxConcurrency           int
	HeartbeatIntervalSeconds int
	ExpectedRevision         int64
}

type RegistryOperation struct {
	SchoolCode            string
	OperationKey          string
	OperationType         string
	AdapterID             string
	AdapterVersion        string
	UpstreamProtocol      string
	TargetHost            string
	TargetPort            int
	TargetTLSServerName   *string
	AllowlistedAttributes []string
	TimeoutMilliseconds   int
	MaxConcurrency        int
	RateLimitPerMinute    int
	ValidationStatus      string
	Enabled               bool
	ExpectedRevision      int64
}

func (manifest RegistryManifest) Validate(now time.Time) error {
	node := manifest.Node
	if !registryUUIDPattern.MatchString(node.ID) || strings.TrimSpace(node.DisplayName) == "" ||
		len(node.DisplayName) > 100 || node.ProtocolVersion != "1" ||
		strings.TrimSpace(node.SoftwareVersion) == "" || len(node.SoftwareVersion) > 128 ||
		!regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(node.CertificateFingerprint) ||
		!node.CertificateNotAfter.After(now) || strings.TrimSpace(node.SigningKeyID) == "" ||
		len(node.SigningPublicKey) != 32 || node.MaxConcurrency < 1 || node.MaxConcurrency > 128 ||
		node.HeartbeatIntervalSeconds < 5 || node.HeartbeatIntervalSeconds > 3600 ||
		node.ExpectedRevision < 0 || len(strings.TrimSpace(manifest.Reason)) < 4 || len(manifest.Reason) > 500 {
		return errors.New("campus connector registry node manifest is invalid")
	}
	if len(manifest.Operations) == 0 || len(manifest.Operations) > 256 {
		return errors.New("campus connector registry must contain a bounded operation set")
	}
	seen := make(map[string]struct{}, len(manifest.Operations))
	for _, operation := range manifest.Operations {
		identity := operation.SchoolCode + "\x00" + operation.OperationKey
		if _, duplicate := seen[identity]; duplicate {
			return errors.New("campus connector registry operation is duplicated")
		}
		seen[identity] = struct{}{}
		if err := operation.Validate(); err != nil {
			return fmt.Errorf("operation %q: %w", operation.OperationKey, err)
		}
	}
	return nil
}

func (operation RegistryOperation) Validate() error {
	if !registrySchoolCodePattern.MatchString(operation.SchoolCode) ||
		!registryOperationPattern.MatchString(operation.OperationKey) ||
		!registryAdapterPattern.MatchString(operation.AdapterID) ||
		strings.TrimSpace(operation.AdapterVersion) == "" || len(operation.AdapterVersion) > 64 ||
		strings.TrimSpace(operation.TargetHost) == "" || strings.ContainsAny(operation.TargetHost, "/?#@") ||
		operation.TargetPort < 1 || operation.TargetPort > 65535 ||
		operation.TimeoutMilliseconds < 100 || operation.TimeoutMilliseconds > 120000 ||
		operation.MaxConcurrency < 1 || operation.MaxConcurrency > 64 ||
		operation.RateLimitPerMinute < 1 || operation.RateLimitPerMinute > 10000 ||
		operation.ExpectedRevision < 0 || len(operation.AllowlistedAttributes) > 64 {
		return errors.New("operation identity, endpoint, or limits are invalid")
	}
	switch operation.OperationType {
	case "school_account_authenticate":
		switch operation.UpstreamProtocol {
		case "ldaps", "ldap_starttls":
			if err := validateRegistryTLSTarget(operation.TargetTLSServerName); err != nil {
				return err
			}
		case "ldap_plain_private_network":
			ip := net.ParseIP(strings.TrimSpace(operation.TargetHost))
			if operation.TargetTLSServerName != nil || operation.TargetPort != 389 ||
				ip == nil || !isRegistryRFC1918IPv4(ip) {
				return errors.New("plaintext LDAP registry operation requires one RFC1918 IPv4:389 target and no TLS name")
			}
		default:
			return errors.New("school account operation requires an approved LDAP protocol")
		}
	case "roster_snapshot_upload":
		switch operation.UpstreamProtocol {
		case "oracle_tls", "https":
			if err := validateRegistryTLSTarget(operation.TargetTLSServerName); err != nil {
				return err
			}
		case "oracle_ssh_tunnel":
			ip := net.ParseIP(strings.TrimSpace(operation.TargetHost))
			if operation.TargetTLSServerName != nil || operation.TargetPort < 1024 || ip == nil || !ip.IsLoopback() {
				return errors.New("oracle SSH tunnel registry operation requires one loopback high-port target and no TLS name")
			}
		default:
			return errors.New("roster operation requires an approved Oracle or HTTPS protocol")
		}
	default:
		return errors.New("operation type is not allowed")
	}
	if operation.ValidationStatus != "pending" && operation.ValidationStatus != "valid" && operation.ValidationStatus != "invalid" {
		return errors.New("validation status is invalid")
	}
	if operation.Enabled && operation.ValidationStatus != "valid" {
		return errors.New("only a validated operation can be enabled")
	}
	for _, attribute := range operation.AllowlistedAttributes {
		if !regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`).MatchString(attribute) {
			return errors.New("allowlisted attribute is invalid")
		}
	}
	return nil
}

func validateRegistryTLSTarget(serverName *string) error {
	if serverName == nil || strings.TrimSpace(*serverName) == "" ||
		strings.ContainsAny(*serverName, "/?#@") || net.ParseIP(strings.TrimSpace(*serverName)) != nil {
		return errors.New("encrypted registry operation requires a non-IP TLS certificate name")
	}
	return nil
}

func isRegistryRFC1918IPv4(ip net.IP) bool {
	ip = ip.To4()
	if ip == nil {
		return false
	}
	return ip[0] == 10 ||
		(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
		(ip[0] == 192 && ip[1] == 168)
}

// ApplyRegistryManifest atomically enrolls or rotates one node and its complete
// approved operation set. Missing operations are disabled, not silently kept
// active. Private keys and upstream credentials are not representable here.
func (r *Repository) ApplyRegistryManifest(ctx context.Context, manifest RegistryManifest, now time.Time) error {
	if err := manifest.Validate(now); err != nil {
		return err
	}
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		nodeRevision, eventType, err := applyRegistryNode(ctx, tx, manifest.Node, now)
		if err != nil {
			return err
		}
		seen := make([]string, 0, len(manifest.Operations))
		for _, operation := range manifest.Operations {
			var schoolID int64
			if err := tx.QueryRow(ctx, `SELECT id FROM schools WHERE code = $1`, operation.SchoolCode).Scan(&schoolID); err != nil {
				return fmt.Errorf("resolve operation school: %w", err)
			}
			if err := applyRegistryOperation(ctx, tx, manifest.Node.ID, schoolID, operation, now); err != nil {
				return err
			}
			seen = append(seen, operation.SchoolCode+"\x00"+operation.OperationKey)
		}
		rows, err := tx.Query(ctx, `
			SELECT school.code, operation.school_id, operation.operation_key
			FROM campus_connector_school_operations operation
			JOIN schools school ON school.id = operation.school_id
			WHERE operation.node_id = $1
			FOR UPDATE OF operation
		`, manifest.Node.ID)
		if err != nil {
			return err
		}
		var obsolete [][2]any
		for rows.Next() {
			var schoolCode, operationKey string
			var schoolID int64
			if err := rows.Scan(&schoolCode, &schoolID, &operationKey); err != nil {
				rows.Close()
				return err
			}
			identity := schoolCode + "\x00" + operationKey
			found := false
			for _, approved := range seen {
				if approved == identity {
					found = true
					break
				}
			}
			if !found {
				obsolete = append(obsolete, [2]any{schoolID, operationKey})
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		for _, item := range obsolete {
			if _, err := tx.Exec(ctx, `
				UPDATE campus_connector_school_operations
				SET enabled = false, validation_status = 'invalid',
				    health_status = 'unavailable', health_code = 'removed_from_manifest',
				    config_revision = config_revision + 1, updated_at = $4
				WHERE node_id = $1 AND school_id = $2 AND operation_key = $3
			`, manifest.Node.ID, item[0], item[1], now); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO campus_connector_node_events (
				node_id, event_type, event_code, reason, revision, occurred_at, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $6)
		`, manifest.Node.ID, eventType, "registry_manifest_applied",
			strings.TrimSpace(manifest.Reason), nodeRevision, now)
		return err
	})
}

func applyRegistryNode(ctx context.Context, tx pgx.Tx, node RegistryNode, now time.Time) (int64, string, error) {
	if node.ExpectedRevision == 0 {
		var revision int64
		err := tx.QueryRow(ctx, `
			INSERT INTO campus_connector_nodes (
				id, display_name, status, protocol_version, software_version,
				certificate_fingerprint, signing_key_id, signing_public_key,
				max_concurrency, heartbeat_interval_seconds, certificate_not_after,
				revision, created_at, updated_at
			) VALUES ($1, $2, 'registered', $3, $4, $5, $6, $7, $8, $9, $10, 1, $11, $11)
			RETURNING revision
		`, node.ID, node.DisplayName, node.ProtocolVersion, node.SoftwareVersion,
			node.CertificateFingerprint, node.SigningKeyID, node.SigningPublicKey,
			node.MaxConcurrency, node.HeartbeatIntervalSeconds, node.CertificateNotAfter, now,
		).Scan(&revision)
		return revision, "registered", err
	}
	var currentRevision int64
	var currentFingerprint string
	if err := tx.QueryRow(ctx, `
		SELECT revision, certificate_fingerprint
		FROM campus_connector_nodes WHERE id = $1 FOR UPDATE
	`, node.ID).Scan(&currentRevision, &currentFingerprint); err != nil {
		return 0, "", err
	}
	if currentRevision != node.ExpectedRevision {
		return 0, "", errors.New("campus connector node revision conflict")
	}
	eventType := "configuration_changed"
	if currentFingerprint != node.CertificateFingerprint {
		eventType = "certificate_rotated"
	}
	var revision int64
	err := tx.QueryRow(ctx, `
		UPDATE campus_connector_nodes
		SET display_name = $2, protocol_version = $3, software_version = $4,
		    certificate_fingerprint = $5, signing_key_id = $6, signing_public_key = $7,
		    max_concurrency = $8, heartbeat_interval_seconds = $9,
		    certificate_not_after = $10, revision = revision + 1, updated_at = $11
		WHERE id = $1 AND revision = $12 AND revoked_at IS NULL
		RETURNING revision
	`, node.ID, node.DisplayName, node.ProtocolVersion, node.SoftwareVersion,
		node.CertificateFingerprint, node.SigningKeyID, node.SigningPublicKey,
		node.MaxConcurrency, node.HeartbeatIntervalSeconds, node.CertificateNotAfter,
		now, node.ExpectedRevision).Scan(&revision)
	return revision, eventType, err
}

func applyRegistryOperation(
	ctx context.Context,
	tx pgx.Tx,
	nodeID string,
	schoolID int64,
	operation RegistryOperation,
	now time.Time,
) error {
	if operation.ExpectedRevision == 0 {
		_, err := tx.Exec(ctx, `
			INSERT INTO campus_connector_school_operations (
				node_id, school_id, operation_key, operation_type,
				adapter_id, adapter_version, upstream_protocol,
				target_host, target_port, target_tls_server_name,
				allowlisted_attributes, timeout_milliseconds, max_concurrency,
				rate_limit_per_minute, enabled, validation_status,
				health_status, health_code, config_revision, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16,
				'unknown', 'awaiting_heartbeat', 1, $17, $17
			)
		`, nodeID, schoolID, operation.OperationKey, operation.OperationType,
			operation.AdapterID, operation.AdapterVersion, operation.UpstreamProtocol,
			operation.TargetHost, operation.TargetPort, operation.TargetTLSServerName,
			operation.AllowlistedAttributes, operation.TimeoutMilliseconds,
			operation.MaxConcurrency, operation.RateLimitPerMinute, operation.Enabled,
			operation.ValidationStatus, now)
		return err
	}
	command, err := tx.Exec(ctx, `
		UPDATE campus_connector_school_operations
		SET operation_type = $4, adapter_id = $5, adapter_version = $6,
		    upstream_protocol = $7, target_host = $8, target_port = $9,
		    target_tls_server_name = $10, allowlisted_attributes = $11,
		    timeout_milliseconds = $12, max_concurrency = $13,
		    rate_limit_per_minute = $14, enabled = $15,
		    validation_status = $16, health_status = 'unknown',
		    health_code = 'awaiting_heartbeat', health_checked_at = NULL,
		    config_revision = config_revision + 1, updated_at = $17
		WHERE node_id = $1 AND school_id = $2 AND operation_key = $3
		  AND config_revision = $18
	`, nodeID, schoolID, operation.OperationKey, operation.OperationType,
		operation.AdapterID, operation.AdapterVersion, operation.UpstreamProtocol,
		operation.TargetHost, operation.TargetPort, operation.TargetTLSServerName,
		operation.AllowlistedAttributes, operation.TimeoutMilliseconds,
		operation.MaxConcurrency, operation.RateLimitPerMinute, operation.Enabled,
		operation.ValidationStatus, now, operation.ExpectedRevision)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("campus connector operation revision conflict")
	}
	return nil
}
