package node

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigRejectsGenericNetworkAndInlineSecretShapes(t *testing.T) {
	base := validTestConfig()
	require.NoError(t, base.Validate())

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name:   "central redirect path",
			mutate: func(cfg *Config) { cfg.CentralBaseURL = "https://central.example/proxy" },
		},
		{
			name:   "plaintext ldap",
			mutate: func(cfg *Config) { cfg.Operations[0].UpstreamProtocol = "ldap" },
		},
		{
			name:   "arbitrary target URL",
			mutate: func(cfg *Config) { cfg.Operations[0].TargetHost = "https://target.example/path" },
		},
		{
			name: "secret outside fixed mount",
			mutate: func(cfg *Config) {
				cfg.Operations[0].LDAP.SystemBindPasswordFile = "/tmp/literal-password"
			},
		},
		{
			name: "secret path traversal",
			mutate: func(cfg *Config) {
				cfg.Operations[0].LDAP.SystemBindPasswordFile = "/run/secrets/../connector-password"
			},
		},
		{
			name: "unreviewed DN token",
			mutate: func(cfg *Config) {
				cfg.Operations[0].LDAP.UserDNTemplate = "uid={student_id},ou={browser_value},dc=example"
			},
		},
		{
			name:   "unknown operation type",
			mutate: func(cfg *Config) { cfg.Operations[0].Type = "tcp_proxy" },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validTestConfig()
			test.mutate(&cfg)
			require.Error(t, cfg.Validate())
		})
	}
}

func TestLoadConfigRejectsRetiredEnvironmentSecretFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node.json")
	raw := `{
  "nodeID":"00000000-0000-4000-8000-000000000001",
  "softwareVersion":"test",
  "centralBaseURL":"https://central.example:9444",
  "clientCertificateFile":"/run/secrets/client.crt",
  "clientPrivateKeyFile":"/run/secrets/client.key",
  "centralCAFile":"/run/secrets/ca.crt",
  "signingKeyID":"signing-key-1",
  "signingPrivateKeyFile":"/run/secrets/signing.key",
  "snapshotEncryptionKeyID":"snapshot-key-1",
  "snapshotPublicKeyFile":"/run/secrets/snapshot.pub",
  "protocolVersion":"1",
  "heartbeatIntervalSeconds":30,
  "pollWorkers":2,
  "maxInteractivePasswordBytes":256,
  "operations":[{
    "key":"school.account.authenticate",
    "schoolCode":"0000000001",
    "type":"school_account_authenticate",
    "adapterID":"school_ldap_bind",
    "adapterVersion":"1",
    "upstreamProtocol":"ldaps",
    "targetHost":"ldap.internal.example",
    "targetPort":636,
    "tlsServerName":"ldap.internal.example",
    "timeoutMilliseconds":5000,
    "ldap":{
      "caFile":"/run/secrets/ldap-ca.crt",
      "userDNTemplate":"uid={student_id},ou=people,dc=example",
      "searchBaseDN":"ou=people,dc=example",
      "systemBindDN":"uid=reader,ou=system,dc=example",
      "systemBindPasswordEnv":"SCHOOL_LDAP_READER_PASSWORD",
      "uidAttribute":"uid",
      "subjectAttribute":"uid",
      "accountLockedAttribute":"accountLocked",
      "accountDisabledAttribute":"accountDisabled"
    }
  }]
}`
	require.NoError(t, os.WriteFile(path, []byte(raw), 0o600))
	_, err := LoadConfig(path)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "unknown field")
}

func TestOracleRosterConfigCannotRepresentSQL(t *testing.T) {
	cfg := validTestConfig()
	oracle := validOracleOperation()
	cfg.Operations = append(cfg.Operations, oracle)
	require.NoError(t, cfg.Validate())

	cfg.Operations[1].OracleRoster.Table = "T_XS_JBXX; DELETE FROM USERS"
	require.Error(t, cfg.Validate())
}

func TestOracleSSHTunnelIsExplicitAndRestrictedToOneLoopbackEndpoint(t *testing.T) {
	valid := validTestConfig()
	operation := validOracleSSHTunnelOperation()
	valid.Operations = append(valid.Operations, operation)
	require.NoError(t, valid.Validate())

	for _, mutate := range []func(*OperationConfig){
		func(operation *OperationConfig) { operation.TargetHost = "10.20.30.40" },
		func(operation *OperationConfig) { operation.TargetHost = "localhost" },
		func(operation *OperationConfig) { operation.TargetPort = 443 },
		func(operation *OperationConfig) { operation.TLSServerName = "oracle.internal.example" },
		func(operation *OperationConfig) { operation.OracleRoster.CAFile = "/run/secrets/oracle-ca.crt" },
		func(operation *OperationConfig) { operation.OracleRoster.AllowPlaintextSSHTunnel = false },
		func(operation *OperationConfig) {
			operation.OracleRoster.AllowedDialTargets = append(
				operation.OracleRoster.AllowedDialTargets, "127.0.0.1:61522",
			)
		},
	} {
		cfg := validTestConfig()
		cfg.Operations = append(cfg.Operations, validOracleSSHTunnelOperation())
		mutate(&cfg.Operations[1])
		require.Error(t, cfg.Validate())
	}
}

func TestPlaintextLDAPIsExplicitAndRestrictedToFixedRFC1918Endpoint(t *testing.T) {
	valid := validTestConfig()
	valid.Operations[0].UpstreamProtocol = "ldap_plain_private_network"
	valid.Operations[0].TargetHost = "10.20.30.40"
	valid.Operations[0].TargetPort = 389
	valid.Operations[0].TLSServerName = ""
	valid.Operations[0].LDAP.CAFile = ""
	valid.Operations[0].LDAP.AllowPlaintextPrivateNetwork = true
	require.NoError(t, valid.Validate())

	for _, mutate := range []func(*Config){
		func(cfg *Config) { cfg.Operations[0].TargetHost = "ldap.internal.example" },
		func(cfg *Config) { cfg.Operations[0].TargetHost = "203.0.113.10" },
		func(cfg *Config) { cfg.Operations[0].TargetPort = 1389 },
		func(cfg *Config) { cfg.Operations[0].LDAP.AllowPlaintextPrivateNetwork = false },
		func(cfg *Config) { cfg.Operations[0].LDAP.CAFile = "/run/secrets/unused-ca.crt" },
	} {
		cfg := validTestConfig()
		cfg.Operations[0].UpstreamProtocol = "ldap_plain_private_network"
		cfg.Operations[0].TargetHost = "10.20.30.40"
		cfg.Operations[0].TargetPort = 389
		cfg.Operations[0].TLSServerName = ""
		cfg.Operations[0].LDAP.CAFile = ""
		cfg.Operations[0].LDAP.AllowPlaintextPrivateNetwork = true
		mutate(&cfg)
		require.Error(t, cfg.Validate())
	}
}

func validTestConfig() Config {
	return Config{
		NodeID:          "00000000-0000-4000-8000-000000000001",
		SoftwareVersion: "test", CentralBaseURL: "https://central.example:9444",
		ClientCertificateFile: "/run/secrets/client.crt",
		ClientPrivateKeyFile:  "/run/secrets/client.key",
		CentralCAFile:         "/run/secrets/ca.crt",
		SigningKeyID:          "signing-key-1", SigningPrivateKeyFile: "/run/secrets/signing.key",
		SnapshotEncryptionKeyID: "snapshot-key-1",
		SnapshotPublicKeyFile:   "/run/secrets/snapshot.pub",
		ProtocolVersion:         "1", HeartbeatIntervalSeconds: 30, PollWorkers: 2,
		MaxInteractivePasswordBytes: 256,
		Operations: []OperationConfig{{
			Key: "school.account.authenticate", SchoolCode: "0000000001",
			Type: "school_account_authenticate", AdapterID: "school_ldap_bind",
			AdapterVersion: "1", UpstreamProtocol: "ldaps",
			TargetHost: "ldap.internal.example", TargetPort: 636,
			TLSServerName: "ldap.internal.example", TimeoutMilliseconds: 5000,
			LDAP: &LDAPOperationConfig{
				CAFile:                 "/run/secrets/ldap-ca.crt",
				UserDNTemplate:         "uid={student_id},ou=people,dc=example",
				SearchBaseDN:           "ou=people,dc=example",
				SystemBindDN:           "uid=reader,ou=system,dc=example",
				SystemBindPasswordFile: "/run/secrets/school-ldap-reader-password",
				UIDAttribute:           "uid", SubjectAttribute: "uid",
				AccountLockedAttribute:   "accountLocked",
				AccountDisabledAttribute: "accountDisabled",
			},
		}},
	}
}

func validOracleOperation() OperationConfig {
	return OperationConfig{
		Key: "school.roster.full", SchoolCode: "0000000001",
		Type: "roster_snapshot_upload", AdapterID: "school_oracle_roster",
		AdapterVersion: "1", UpstreamProtocol: "oracle_tls",
		TargetHost: "oracle.internal.example", TargetPort: 2484,
		TLSServerName: "oracle.internal.example", TimeoutMilliseconds: 120000,
		OracleRoster: &OracleRosterOperationConfig{
			ServiceName: "ORCL", UsernameFile: "/run/secrets/school-oracle-username",
			PasswordFile: "/run/secrets/school-oracle-password", CAFile: "/run/secrets/oracle-ca.crt",
			AllowedDialTargets: []string{"oracle.internal.example:2484", "10.20.30.41:2484"},
			ExpectedUsername:   "STUHELPER_RO", Schema: "ZHFWDB", Table: "T_XS_JBXX",
			ActiveFilterColumn: "DQXJ", ActiveFilterValue: "1", ActiveEligibilityCode: "CURRENT",
			FullSnapshotIntervalMinutes: 360, MappingVersion: "v1",
			DefaultDocumentType: "mainland_resident_id", MaximumRows: 200000,
			Columns: OracleRosterColumns{
				StudentID: "XH", Name: "XM", DocumentNumber: "SFZJH",
				EnrollmentYear: "RXNJ", CurrentMarker: "DQXJ", EligibilityCode: "XJZT",
			},
		},
	}
}

func validOracleSSHTunnelOperation() OperationConfig {
	operation := validOracleOperation()
	operation.UpstreamProtocol = "oracle_ssh_tunnel"
	operation.TargetHost = "127.0.0.1"
	operation.TargetPort = 61521
	operation.TLSServerName = ""
	operation.OracleRoster.CAFile = ""
	operation.OracleRoster.AllowPlaintextSSHTunnel = true
	operation.OracleRoster.AllowedDialTargets = []string{"127.0.0.1:61521"}
	return operation
}
