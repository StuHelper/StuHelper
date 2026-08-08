package node

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"

	"github.com/StuHelper/StuHelper/server/internal/modules/campusconnector"
	connectorprotocol "github.com/StuHelper/StuHelper/server/internal/pkg/campusconnectorprotocol"
)

type LDAPAdapter struct {
	operation OperationConfig
	tlsConfig *tls.Config
}

func NewLDAPAdapter(operation OperationConfig) (*LDAPAdapter, error) {
	if err := operation.Validate(); err != nil {
		return nil, err
	}
	if operation.Type != "school_account_authenticate" || operation.LDAP == nil {
		return nil, errors.New("operation is not an LDAP school-account adapter")
	}
	var rootCAs *x509.CertPool
	if operation.UpstreamProtocol != "ldap_plain_private_network" {
		var err error
		rootCAs, err = loadCertificatePool(operation.LDAP.CAFile)
		if err != nil {
			return nil, err
		}
	}
	var tlsConfig *tls.Config
	if rootCAs != nil {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: operation.TLSServerName,
			RootCAs:    rootCAs,
		}
	}
	return &LDAPAdapter{
		operation: operation,
		tlsConfig: tlsConfig,
	}, nil
}

// Authenticate performs exactly one transport-policy-protected user bind followed
// by one independent system lookup. LDAPS/StartTLS is the default; the only
// plaintext exception is an explicitly enabled, fixed RFC1918 IPv4:389 operation
// accepted by OperationConfig.Validate. The browser request never owns the host,
// port, DN template, filter, or attributes.
func (a *LDAPAdapter) Authenticate(
	ctx context.Context,
	metadata connectorprotocol.InteractiveMetadata,
	password []byte,
) connectorprotocol.InteractiveResult {
	result := connectorprotocol.InteractiveResult{RequestID: metadata.RequestID}
	if metadata.OperationKey != a.operation.Key || metadata.AdapterID != a.operation.AdapterID ||
		metadata.AdapterVersion != a.operation.AdapterVersion || metadata.StudentID == "" ||
		len(password) == 0 || !metadata.DeadlineAt.After(time.Now()) {
		result.ResultCode = campusconnector.ResultSchemaUnknown
		return result
	}
	userConnection, err := a.dial(ctx)
	if err != nil {
		result.ResultCode = mapLDAPDependencyError(err)
		return result
	}
	userDN := strings.ReplaceAll(
		a.operation.LDAP.UserDNTemplate,
		"{student_id}",
		ldap.EscapeDN(metadata.StudentID),
	)
	// go-ldap's Bind API requires string. This immutable copy is scoped to this
	// call, is never logged or persisted, and is discarded immediately after
	// the bind returns; the mutable transport buffer is wiped by the runner.
	err = userConnection.Bind(userDN, string(password))
	closeErr := userConnection.Close()
	if err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			result.ResultCode = campusconnector.ResultRejected
		} else {
			result.ResultCode = mapLDAPDependencyError(err)
		}
		return result
	}
	if closeErr != nil {
		result.ResultCode = mapLDAPDependencyError(closeErr)
		return result
	}

	entry, code := a.lookupAccount(ctx, metadata.StudentID)
	if code != campusconnector.ResultSuccess {
		result.ResultCode = code
		return result
	}
	result.ResultCode = campusconnector.ResultSuccess
	result.StudentID = entry.GetAttributeValue(a.operation.LDAP.UIDAttribute)
	result.AccountSubject = entry.GetAttributeValue(a.operation.LDAP.SubjectAttribute)
	if result.StudentID != metadata.StudentID || strings.TrimSpace(result.AccountSubject) == "" {
		result.ResultCode = campusconnector.ResultSchemaUnknown
		result.StudentID = ""
		result.AccountSubject = ""
	}
	return result
}

func (a *LDAPAdapter) Health(ctx context.Context) (health string) {
	secret, err := readSecretFile(a.operation.LDAP.SystemBindPasswordFile)
	if err != nil {
		return "secret_unavailable"
	}
	defer wipe(secret)
	connection, err := a.dial(ctx)
	if err != nil {
		return mapLDAPDependencyError(err)
	}
	health = campusconnector.ResultUnavailable
	defer func() {
		if closeErr := connection.Close(); closeErr != nil && health == "ok" {
			health = mapLDAPDependencyError(closeErr)
		}
	}()
	err = connection.Bind(a.operation.LDAP.SystemBindDN, string(secret))
	if err != nil {
		return mapLDAPDependencyError(err)
	}
	return "ok"
}

func (a *LDAPAdapter) lookupAccount(ctx context.Context, studentID string) (entryResult *ldap.Entry, resultCode string) {
	secret, err := readSecretFile(a.operation.LDAP.SystemBindPasswordFile)
	if err != nil {
		return nil, "secret_unavailable"
	}
	defer wipe(secret)
	connection, err := a.dial(ctx)
	if err != nil {
		return nil, mapLDAPDependencyError(err)
	}
	resultCode = campusconnector.ResultUnavailable
	defer func() {
		if closeErr := connection.Close(); closeErr != nil && resultCode == campusconnector.ResultSuccess {
			entryResult = nil
			resultCode = mapLDAPDependencyError(closeErr)
		}
	}()
	err = connection.Bind(a.operation.LDAP.SystemBindDN, string(secret))
	if err != nil {
		return nil, mapLDAPDependencyError(err)
	}
	attributes := []string{
		a.operation.LDAP.UIDAttribute,
		a.operation.LDAP.SubjectAttribute,
		a.operation.LDAP.AccountLockedAttribute,
		a.operation.LDAP.AccountDisabledAttribute,
	}
	if marker := a.operation.LDAP.StudentMarkerAttribute; marker != "" {
		attributes = append(attributes, marker)
	}
	attributes = compactStrings(attributes)
	filter := fmt.Sprintf("(%s=%s)",
		a.operation.LDAP.UIDAttribute,
		ldap.EscapeFilter(studentID),
	)
	request := ldap.NewSearchRequest(
		a.operation.LDAP.SearchBaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		2,
		int(a.timeout()/time.Second),
		false,
		filter,
		attributes,
		nil,
	)
	response, err := connection.Search(request)
	if err != nil {
		return nil, mapLDAPDependencyError(err)
	}
	if len(response.Entries) == 0 {
		return nil, campusconnector.ResultNotStudent
	}
	if len(response.Entries) != 1 {
		return nil, campusconnector.ResultSchemaUnknown
	}
	entry := response.Entries[0]
	locked, lockedOK := ldapFalseAttribute(entry, a.operation.LDAP.AccountLockedAttribute)
	disabled, disabledOK := ldapFalseAttribute(entry, a.operation.LDAP.AccountDisabledAttribute)
	if !lockedOK || !disabledOK {
		return nil, campusconnector.ResultSchemaUnknown
	}
	if !locked || !disabled {
		return nil, campusconnector.ResultAccountLocked
	}
	if marker := a.operation.LDAP.StudentMarkerAttribute; marker != "" {
		value := strings.TrimSpace(entry.GetAttributeValue(marker))
		if !slices.Contains(a.operation.LDAP.StudentMarkerValues, value) {
			return nil, campusconnector.ResultNotStudent
		}
	}
	return entry, campusconnector.ResultSuccess
}

func (a *LDAPAdapter) dial(ctx context.Context) (*ldap.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scheme := "ldaps"
	if a.operation.UpstreamProtocol == "ldap_starttls" || a.operation.UpstreamProtocol == "ldap_plain_private_network" {
		scheme = "ldap"
	}
	address := fmt.Sprintf("%s://%s:%d", scheme, a.operation.TargetHost, a.operation.TargetPort)
	var dialOptions []ldap.DialOpt
	if a.tlsConfig != nil && a.operation.UpstreamProtocol != "ldap_plain_private_network" {
		dialOptions = append(dialOptions, ldap.DialWithTLSConfig(a.tlsConfig.Clone()))
	}
	connection, err := ldap.DialURL(address, dialOptions...)
	if err != nil {
		return nil, err
	}
	connection.SetTimeout(a.timeout())
	if a.operation.UpstreamProtocol == "ldap_starttls" {
		if err := connection.StartTLS(a.tlsConfig.Clone()); err != nil {
			return nil, errors.Join(err, connection.Close())
		}
	}
	return connection, nil
}

func (a *LDAPAdapter) timeout() time.Duration {
	return time.Duration(a.operation.TimeoutMilliseconds) * time.Millisecond
}

func ldapFalseAttribute(entry *ldap.Entry, attribute string) (bool, bool) {
	values := entry.GetAttributeValues(attribute)
	if len(values) != 1 {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(values[0])) {
	case "false", "0", "no":
		return true, true
	case "true", "1", "yes":
		return false, true
	default:
		return false, false
	}
}

func mapLDAPDependencyError(err error) string {
	if err == nil {
		return campusconnector.ResultSuccess
	}
	var certificateError x509.UnknownAuthorityError
	if errors.As(err, &certificateError) || strings.Contains(strings.ToLower(err.Error()), "certificate") ||
		strings.Contains(strings.ToLower(err.Error()), "tls") {
		return campusconnector.ResultTLSFailure
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return campusconnector.ResultCancelled
	}
	return campusconnector.ResultUnavailable
}

func compactStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func loadCertificatePool(path string) (*x509.CertPool, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("operation TLS CA file reference is missing")
	}
	// #nosec G304 -- the CA path is immutable adapter configuration, never request input.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer wipe(raw)
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(raw) {
		return nil, errors.New("operation TLS CA file contains no certificate")
	}
	return pool, nil
}
