package externaldata

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildOracleFullRosterQueryUsesOnlyValidatedIdentifiers(t *testing.T) {
	cfg := OracleRosterSnapshotConfig{
		Schema: "ZHFWDB", Table: "T_XS_JBXX",
		ActiveFilterColumn: "SFZX", ActiveFilterValue: "1",
		Columns: OracleRosterSnapshotColumns{
			StudentID: "XH", Name: "XM", DocumentNumber: "SFZJH",
			EnrollmentYear: "RXNJ", CurrentMarker: "SFZX", EligibilityCode: "SFZX",
		},
	}
	query := buildOracleFullRosterQuery(cfg)
	require.Contains(t, query, "FROM ZHFWDB.T_XS_JBXX")
	require.Contains(t, query, "WHERE SFZX = :1")
	require.Contains(t, query, "ORDER BY XH")
	require.NotContains(t, strings.ToUpper(query), "UPDATE ")
	require.NotContains(t, strings.ToUpper(query), "DELETE ")
	require.NotContains(t, strings.ToUpper(query), "INSERT ")
}

func TestMapOracleRosterRowDropsInvalidOptionalBUAAProjections(t *testing.T) {
	values := make([]sql.NullString, 16)
	values[0] = sql.NullString{String: "20990001", Valid: true}
	values[1] = sql.NullString{String: "测试学生", Valid: true}
	values[3] = sql.NullString{String: "not-an-identity-card", Valid: true}
	values[4] = sql.NullString{String: "10000000000", Valid: true}
	values[10] = sql.NullString{String: "0", Valid: true}
	values[13] = sql.NullString{String: "1", Valid: true}
	values[14] = sql.NullString{String: "1", Valid: true}

	record, quality, err := mapOracleRosterRowWithQuality(values, "mainland_resident_id", "CURRENT")
	require.NoError(t, err)
	require.Empty(t, record.DocumentType)
	require.Empty(t, record.DocumentNumber)
	require.Empty(t, record.Phone)
	require.Nil(t, record.EnrollmentYear)
	require.Equal(t, int64(1), quality.InvalidDocumentNumber)
	require.Equal(t, int64(1), quality.InvalidPhone)
	require.Equal(t, int64(1), quality.InvalidEnrollmentYear)
}

func TestOracleAllowlistedDialerRejectsUnapprovedRACRedirect(t *testing.T) {
	cfg := OracleRosterSnapshotConfig{
		Host: "oracle.service.internal", Port: 1521,
		AllowedDialTargets: []string{
			"oracle.service.internal:1521",
			"10.20.30.41:1521",
		},
		ConnectTimeout: time.Second,
	}
	dialer, err := newOracleAllowlistedDialer(cfg)
	require.NoError(t, err)
	_, err = dialer.DialContext(context.Background(), "tcp", "10.20.30.99:1521")
	require.ErrorContains(t, err, "not approved")
}

func TestValidateOracleRosterSnapshotConfigAllowsExplicitExistingSchemaOwner(t *testing.T) {
	err := validateOracleRosterSnapshotConfig(OracleRosterSnapshotConfig{
		Host: "oracle.service.internal", Port: 2484,
		TransportMode: "oracle_tls",
		TLSServerName: "oracle.service.internal", ServiceName: "ORCLPDB1",
		Username: "ZHFWDB", Password: "secret", ExpectedUsername: "ZHFWDB",
		CAFile: "/run/secrets/oracle-ca.crt", Schema: "ZHFWDB", Table: "T_XS_JBXX",
		AllowedDialTargets: []string{"oracle.service.internal:2484"},
		ActiveFilterColumn: "SFZX", ActiveFilterValue: "1", ActiveEligibilityCode: "CURRENT",
		MaximumRows: 200000, ConnectTimeout: time.Second, QueryTimeout: time.Minute,
		Columns: OracleRosterSnapshotColumns{
			StudentID: "XH", Name: "XM", CurrentMarker: "SFZX", EligibilityCode: "SFZX",
		},
	})

	require.NoError(t, err)
}

func TestValidateOracleRosterSnapshotConfigAllowsOnlyBoundedSSHTunnel(t *testing.T) {
	base := OracleRosterSnapshotConfig{
		Host: "127.0.0.1", Port: 61521, TransportMode: "oracle_ssh_tunnel",
		ServiceName: "ORCLPDB1", Username: "ZHFWDB", Password: "secret", ExpectedUsername: "ZHFWDB",
		Schema: "ZHFWDB", Table: "T_XS_JBXX", AllowedDialTargets: []string{"127.0.0.1:61521"},
		ActiveFilterColumn: "SFZX", ActiveFilterValue: "1", ActiveEligibilityCode: "CURRENT",
		MaximumRows: 200000, ConnectTimeout: time.Second, QueryTimeout: time.Minute,
		Columns: OracleRosterSnapshotColumns{
			StudentID: "XH", Name: "XM", CurrentMarker: "SFZX", EligibilityCode: "SFZX",
		},
	}
	require.NoError(t, validateOracleRosterSnapshotConfig(base))

	for _, mutate := range []func(*OracleRosterSnapshotConfig){
		func(cfg *OracleRosterSnapshotConfig) { cfg.Host = "10.20.30.40" },
		func(cfg *OracleRosterSnapshotConfig) { cfg.Host = "localhost" },
		func(cfg *OracleRosterSnapshotConfig) { cfg.Port = 443 },
		func(cfg *OracleRosterSnapshotConfig) { cfg.TLSServerName = "oracle.internal.example" },
		func(cfg *OracleRosterSnapshotConfig) { cfg.CAFile = "/run/secrets/oracle-ca.crt" },
		func(cfg *OracleRosterSnapshotConfig) {
			cfg.AllowedDialTargets = append(cfg.AllowedDialTargets, "127.0.0.1:61522")
		},
	} {
		cfg := base
		cfg.AllowedDialTargets = append([]string(nil), base.AllowedDialTargets...)
		mutate(&cfg)
		require.Error(t, validateOracleRosterSnapshotConfig(cfg))
	}
}

func TestMapOracleRosterRowRequiresExplicitCurrentAndEligibilityEvidence(t *testing.T) {
	values := make([]sql.NullString, 16)
	for _, index := range []int{0, 1, 3, 10, 13, 14} {
		values[index] = sql.NullString{String: "value", Valid: true}
	}
	values[0].String = "20990001"
	values[1].String = "测试学生"
	values[3].String = "110105200001010011"
	values[10].String = "2023"
	values[13].String = "Y"
	values[14].String = "ACTIVE"
	record, err := mapOracleRosterRow(values, "mainland_resident_id")
	require.NoError(t, err)
	require.Equal(t, 2023, *record.EnrollmentYear)
	require.True(t, *record.CurrentMarker)
	require.Equal(t, "ACTIVE", record.EligibilityCode)

	values[13].String = "UNKNOWN"
	_, err = mapOracleRosterRow(values, "mainland_resident_id")
	require.Error(t, err)
}

func TestValidateConfiguredOracleRosterColumnsAcceptsExistingOwnerTable(t *testing.T) {
	cfg := OracleRosterSnapshotConfig{
		ActiveFilterColumn: "SFZX",
		Columns: OracleRosterSnapshotColumns{
			StudentID: "XH", Name: "XM", DocumentNumber: "SFZJH",
			CurrentMarker: "SFZX", EligibilityCode: "SFZX",
		},
	}
	metadata := oracleRosterSourceMetadata{
		ObjectType: "TABLE",
		Columns: []OracleRosterSourceColumn{
			{Name: "XH", DataType: "VARCHAR2"},
			{Name: "XM", DataType: "VARCHAR2"},
			{Name: "SFZJH", DataType: "VARCHAR2", Nullable: true},
			{Name: "SFZX", DataType: "VARCHAR2"},
		},
	}
	require.NoError(t, validateConfiguredOracleRosterColumns(metadata, cfg))

	cfg.Columns.Phone = "SJH"
	require.ErrorContains(t, validateConfiguredOracleRosterColumns(metadata, cfg), "absent")
}
