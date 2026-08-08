package schoolauth

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	EmailIdentityPolicyAcademicStudentEmail = "academic_student_email"
	MaxStudentIDRunes                       = 50
	MaxAcademicNameRunes                    = 80
)

var studentIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,49}$`)

var mainlandDocumentChecksumWeights = [...]int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}

const mainlandDocumentChecksumChars = "10X98765432"

type AdmissionSettings struct {
	EmailDomains        []string
	SSOLoginURL         string
	EmailIdentityPolicy *EmailIdentityPolicy
}

type EmailIdentityPolicy struct {
	Type                 string `json:"type,omitempty"`
	StudentIDEmailDomain string `json:"studentIDEmailDomain,omitempty"`
	RequireStudentName   bool   `json:"requireStudentName,omitempty"`
}

func (p *EmailIdentityPolicy) IsAcademicStudentEmail() bool {
	if p == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(p.Type), EmailIdentityPolicyAcademicStudentEmail)
}

func (p *EmailIdentityPolicy) Normalize() *EmailIdentityPolicy {
	if p == nil {
		return nil
	}
	normalized := &EmailIdentityPolicy{
		Type:                 strings.TrimSpace(p.Type),
		StudentIDEmailDomain: NormalizeEmailDomain(p.StudentIDEmailDomain),
		RequireStudentName:   p.RequireStudentName,
	}
	if normalized.Type == "" {
		return nil
	}
	return normalized
}

func ParseAdmissionSettings(raw []byte) AdmissionSettings {
	var settings AdmissionSettings
	var envelope struct {
		Admission struct {
			EmailDomains        []string             `json:"emailDomains"`
			SSOLoginURL         string               `json:"ssoLoginURL"`
			EmailIdentityPolicy *EmailIdentityPolicy `json:"emailIdentityPolicy"`
		} `json:"admission"`
	}
	if strings.TrimSpace(string(raw)) == "" || json.Unmarshal(raw, &envelope) != nil {
		return settings
	}
	settings.EmailDomains = NormalizeEmailDomains(envelope.Admission.EmailDomains)
	settings.SSOLoginURL = strings.TrimSpace(envelope.Admission.SSOLoginURL)
	settings.EmailIdentityPolicy = envelope.Admission.EmailIdentityPolicy.Normalize()
	return settings
}

func NormalizeEmailDomains(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		domain := NormalizeEmailDomain(value)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		result = append(result, domain)
	}
	return result
}

func NormalizeEmailDomain(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func NormalizeEmailAddress(value string) string {
	email := strings.ToLower(strings.TrimSpace(value))
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	local := strings.TrimSpace(parts[0])
	domain := NormalizeEmailDomain(parts[1])
	if local == "" || domain == "" {
		return ""
	}
	if strings.ContainsFunc(local, unicode.IsSpace) || strings.ContainsFunc(domain, unicode.IsSpace) {
		return ""
	}
	if local != parts[0] || domain != parts[1] {
		return ""
	}
	return local + "@" + domain
}

func NormalizeStudentID(value string) string {
	return strings.TrimSpace(value)
}

func IsValidStudentID(value string) bool {
	normalized := NormalizeStudentID(value)
	return utf8.ValidString(normalized) && studentIDPattern.MatchString(normalized)
}

func NormalizeAcademicName(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), "")
}

func IsValidAcademicName(value string) bool {
	normalized := NormalizeAcademicName(value)
	if normalized == "" || !utf8.ValidString(normalized) ||
		utf8.RuneCountInString(normalized) > MaxAcademicNameRunes {
		return false
	}
	return !strings.ContainsFunc(normalized, func(r rune) bool {
		return unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Co, unicode.Cs)
	})
}

// NormalizeMainlandDocumentNumber returns the canonical upper-case mainland
// resident identity-card number only when its shape, birth date and checksum
// are all valid. Keeping this in the shared school policy package prevents the
// campus snapshot adapter and the online verification path from drifting.
func NormalizeMainlandDocumentNumber(value string) (string, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if len(normalized) != 18 || normalized[0] == '0' {
		return "", false
	}
	for index := 0; index < 17; index++ {
		if normalized[index] < '0' || normalized[index] > '9' {
			return "", false
		}
	}
	last := normalized[17]
	if (last < '0' || last > '9') && last != 'X' {
		return "", false
	}
	birthDate := normalized[6:14]
	parsed, err := time.Parse("20060102", birthDate)
	if err != nil || parsed.Format("20060102") != birthDate {
		return "", false
	}
	checksum := 0
	for index, weight := range mainlandDocumentChecksumWeights {
		checksum += int(normalized[index]-'0') * weight
	}
	if mainlandDocumentChecksumChars[checksum%11] != last {
		return "", false
	}
	return normalized, true
}

func DeriveStudentEmail(studentID string, domain string) string {
	normalizedStudentID := NormalizeStudentID(studentID)
	normalizedDomain := NormalizeEmailDomain(domain)
	if !IsValidStudentID(normalizedStudentID) || normalizedDomain == "" {
		return ""
	}
	return strings.ToLower(normalizedStudentID + "@" + normalizedDomain)
}

func EmailDomainAllowed(email string, domains []string) bool {
	normalized := NormalizeEmailAddress(email)
	if normalized == "" {
		return false
	}
	parts := strings.Split(normalized, "@")
	if len(parts) != 2 {
		return false
	}
	for _, domain := range domains {
		if parts[1] == NormalizeEmailDomain(domain) {
			return true
		}
	}
	return false
}
