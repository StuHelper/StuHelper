package go_ora

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStuHelperAutomaticConnectionPathIsAuthenticationAndSelectOnly(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve policy test source path")
	}
	directory := filepath.Dir(currentFile)
	for _, name := range []string{"auth_object.go", "driver.go"} {
		contents, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		upper := strings.ToUpper(string(contents))
		if strings.Contains(upper, "AUTH_"+"ALTER_SESSION") ||
			strings.Contains(upper, "ALTER"+" SESSION") {
			t.Fatalf("%s contains a forbidden automatic session mutation", name)
		}
	}

	contents, err := os.ReadFile(filepath.Join(directory, "connection.go"))
	if err != nil {
		t.Fatalf("read connection.go: %v", err)
	}
	text := string(contents)
	start := strings.Index(text, "func (conn *Connection) GetNLS()")
	end := strings.Index(text, "func (conn *Connection) Prepare(")
	if start < 0 || end <= start {
		t.Fatal("locate GetNLS implementation")
	}
	getNLS := strings.ToUpper(text[start:end])
	if !strings.Contains(getNLS, "SELECT") || strings.Contains(getNLS, "BEGIN"+"\n") ||
		strings.Contains(getNLS, "END"+";") {
		t.Fatal("GetNLS must remain one fixed SELECT without a PL/SQL block")
	}
}

func TestStuHelperOracleAuthenticationRequiresModernVerifier(t *testing.T) {
	if err := validateStuHelperOraclePasswordVerifier(stuhelperOraclePasswordVerifier); err != nil {
		t.Fatalf("accept modern Oracle password verifier: %v", err)
	}
	for _, legacyVerifier := range []int{2361, 6949} {
		if err := validateStuHelperOraclePasswordVerifier(legacyVerifier); err == nil {
			t.Fatalf("legacy Oracle password verifier %d was accepted", legacyVerifier)
		}
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve policy test source path")
	}
	contents, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "auth_object.go"))
	if err != nil {
		t.Fatalf("read auth_object.go: %v", err)
	}
	for _, forbidden := range []string{"crypto/des", "crypto/md5", "crypto/sha1"} {
		if strings.Contains(string(contents), forbidden) {
			t.Fatalf("auth_object.go retains forbidden legacy primitive %q", forbidden)
		}
	}
}

func TestParseOracleUint32PropertyRejectsInvalidValues(t *testing.T) {
	parsed, err := parseOracleUint32Property("AUTH_SESSION_ID", "42")
	if err != nil || parsed != 42 {
		t.Fatalf("parse valid Oracle session identifier: value=%d err=%v", parsed, err)
	}
	for _, invalid := range []string{"-1", "4294967296", "not-a-number"} {
		if _, err := parseOracleUint32Property("AUTH_SESSION_ID", invalid); err == nil {
			t.Fatalf("invalid Oracle session identifier %q was accepted", invalid)
		}
	}
}

func TestStuHelperSelectOnlyRuntimeGate(t *testing.T) {
	for _, query := range []string{
		"SELECT XH, XM FROM ZHFWDB.T_XS_JBXX WHERE XH = :1",
		"SELECT OBJECT_TYPE FROM ALL_OBJECTS WHERE OWNER = :1 AND OBJECT_NAME = :2",
		"SELECT COLUMN_NAME, DATA_TYPE, NULLABLE FROM ALL_TAB_COLUMNS WHERE OWNER = :1 AND TABLE_NAME = :2",
	} {
		if err := validateStuHelperSelectOnly(query); err != nil {
			t.Fatalf("allow fixed SELECT %q: %v", query, err)
		}
	}
	for _, query := range []string{
		"UPDATE ZHFWDB.T_XS_JBXX SET XH = XH",
		"SELECT XH FROM ZHFWDB.T_XS_JBXX FOR UPDATE",
		"SELECT SEQ.NEXTVAL FROM DUAL",
		"SELECT DBMS_RANDOM.VALUE FROM DUAL",
		"SELECT XH FROM ZHFWDB.T_XS_JBXX; DELETE FROM ZHFWDB.T_XS_JBXX",
	} {
		if err := validateStuHelperSelectOnly(query); err == nil {
			t.Fatalf("reject query outside SELECT-only policy: %q", query)
		}
	}
	if _, err := (&Connection{}).Begin(); err == nil {
		t.Fatal("transaction begin must be disabled")
	}
	if _, err := (&Stmt{}).Exec(nil); err == nil {
		t.Fatal("statement Exec must be disabled")
	}
	if err := (&Transaction{}).Commit(); err == nil {
		t.Fatal("transaction commit must be disabled")
	}
	if err := (&BulkCopy{}).StartStream(); err == nil {
		t.Fatal("bulk copy must be disabled")
	}
	if err := AddSessionParam(nil, "NLS_LANGUAGE", "AMERICAN"); err == nil {
		t.Fatal("session parameters must be disabled")
	}
	databaseURL := BuildUrl("oracle.example.test", 1521, "ORCLPDB1", "existing_user", "secret", nil)
	connection, err := NewConnection(databaseURL, nil)
	if err != nil {
		t.Fatalf("construct policy connection: %v", err)
	}
	if connection.autoCommit {
		t.Fatal("Oracle auto-commit flags must remain disabled")
	}
}
