package campusconnector

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRegistryManifestIsNarrowAndRevisionFenced(t *testing.T) {
	now := time.Now().UTC()
	manifest := RegistryManifest{
		Node: RegistryNode{
			ID:          "00000000-0000-4000-8000-000000000001",
			DisplayName: "approved node", ProtocolVersion: "1", SoftwareVersion: "test",
			CertificateFingerprint: string(bytes.Repeat([]byte{'a'}, 64)),
			CertificateNotAfter:    now.Add(24 * time.Hour), SigningKeyID: "key-1",
			SigningPublicKey: bytes.Repeat([]byte{1}, 32), MaxConcurrency: 4,
			HeartbeatIntervalSeconds: 30, ExpectedRevision: 0,
		},
		Operations: []RegistryOperation{{
			SchoolCode: "0000000001", OperationKey: "school.account.authenticate",
			OperationType: "school_account_authenticate", AdapterID: "school_ldap_bind",
			AdapterVersion: "1", UpstreamProtocol: "ldaps",
			TargetHost: "ldap.internal.example", TargetPort: 636,
			TargetTLSServerName: stringPointer("ldap.internal.example"),
			TimeoutMilliseconds: 5000, MaxConcurrency: 2, RateLimitPerMinute: 30,
			ValidationStatus: "pending", ExpectedRevision: 0,
		}},
		Reason: "initial reviewed enrollment",
	}
	require.NoError(t, manifest.Validate(now))

	manifest.Operations[0].OperationType = "tcp_proxy"
	require.Error(t, manifest.Validate(now))
	manifest.Operations[0].OperationType = "school_account_authenticate"
	manifest.Operations[0].TargetHost = "ldap.internal.example/path"
	require.Error(t, manifest.Validate(now))
	manifest.Operations[0].TargetHost = "ldap.internal.example"
	manifest.Operations[0].Enabled = true
	require.Error(t, manifest.Validate(now), "pending operations cannot be enabled")
}

func TestRegistryManifestAcceptsOnlyBoundedPlaintextExceptions(t *testing.T) {
	now := time.Now().UTC()
	manifest := validRegistryManifest(now)
	operation := &manifest.Operations[0]
	operation.UpstreamProtocol = "ldap_plain_private_network"
	operation.TargetHost = "10.20.30.40"
	operation.TargetPort = 389
	operation.TargetTLSServerName = nil
	require.NoError(t, manifest.Validate(now))

	operation.TargetHost = "ldap.internal.example"
	require.Error(t, manifest.Validate(now))

	manifest = validRegistryManifest(now)
	operation = &manifest.Operations[0]
	operation.OperationType = "roster_snapshot_upload"
	operation.OperationKey = "school.roster.full"
	operation.AdapterID = "school_oracle_roster"
	operation.UpstreamProtocol = "oracle_ssh_tunnel"
	operation.TargetHost = "127.0.0.1"
	operation.TargetPort = 61521
	operation.TargetTLSServerName = nil
	require.NoError(t, manifest.Validate(now))

	operation.TargetHost = "10.20.30.40"
	require.Error(t, manifest.Validate(now))
}

func TestRegistryManifestRequiresTLSNameForEncryptedProtocols(t *testing.T) {
	now := time.Now().UTC()
	manifest := validRegistryManifest(now)
	manifest.Operations[0].TargetTLSServerName = nil
	require.Error(t, manifest.Validate(now))

	manifest = validRegistryManifest(now)
	operation := &manifest.Operations[0]
	operation.OperationType = "roster_snapshot_upload"
	operation.OperationKey = "school.roster.full"
	operation.AdapterID = "school_oracle_roster"
	operation.UpstreamProtocol = "oracle_tls"
	operation.TargetHost = "oracle.internal.example"
	operation.TargetPort = 2484
	operation.TargetTLSServerName = nil
	require.Error(t, manifest.Validate(now))
}

func validRegistryManifest(now time.Time) RegistryManifest {
	return RegistryManifest{
		Node: RegistryNode{
			ID: "00000000-0000-4000-8000-000000000001", DisplayName: "approved node",
			ProtocolVersion: "1", SoftwareVersion: "test",
			CertificateFingerprint: string(bytes.Repeat([]byte{'a'}, 64)),
			CertificateNotAfter:    now.Add(24 * time.Hour), SigningKeyID: "key-1",
			SigningPublicKey: bytes.Repeat([]byte{1}, 32), MaxConcurrency: 4,
			HeartbeatIntervalSeconds: 30,
		},
		Operations: []RegistryOperation{{
			SchoolCode: "0000000001", OperationKey: "school.account.authenticate",
			OperationType: "school_account_authenticate", AdapterID: "school_ldap_bind",
			AdapterVersion: "1", UpstreamProtocol: "ldaps",
			TargetHost: "ldap.internal.example", TargetPort: 636,
			TargetTLSServerName: stringPointer("ldap.internal.example"),
			TimeoutMilliseconds: 5000, MaxConcurrency: 2, RateLimitPerMinute: 30,
			ValidationStatus: "pending",
		}},
		Reason: "initial reviewed enrollment",
	}
}

func stringPointer(value string) *string { return &value }
