package studentverification

import (
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/StuHelper/StuHelper/server/internal/pkg/schoolauth"
)

const (
	BUAAAdapterID  = "buaa"
	BUAASchoolCode = "4111010006"
)

type BUAAAdapter struct{}

func (BUAAAdapter) NormalizeStudentID(value string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" || !utf8.ValidString(normalized) || utf8.RuneCountInString(normalized) > 64 {
		return "", false
	}
	if strings.ContainsFunc(normalized, disallowedIdentifierRune) {
		return "", false
	}
	return normalized, true
}

func (BUAAAdapter) NormalizeName(value string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" || !utf8.ValidString(normalized) || utf8.RuneCountInString(normalized) > 100 {
		return "", false
	}
	if strings.ContainsFunc(normalized, disallowedNameRune) {
		return "", false
	}
	return normalized, true
}

func (BUAAAdapter) NormalizeMainlandDocumentNumber(value string) (string, bool) {
	return schoolauth.NormalizeMainlandDocumentNumber(value)
}

func (BUAAAdapter) SupportsMainlandDocumentType(raw string, enrollmentPolicy json.RawMessage) bool {
	var policy struct {
		MainlandDocumentTypes []string `json:"mainlandDocumentTypes"`
	}
	if len(enrollmentPolicy) == 0 || json.Unmarshal(enrollmentPolicy, &policy) != nil {
		return false
	}
	normalized := strings.TrimSpace(raw)
	for _, allowed := range policy.MainlandDocumentTypes {
		if normalized == strings.TrimSpace(allowed) && normalized != "" {
			return true
		}
	}
	return false
}

func disallowedIdentifierRune(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Co, unicode.Cs)
}

func disallowedNameRune(r rune) bool {
	return unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Co, unicode.Cs)
}

func maskStudentID(value string) string {
	runes := []rune(value)
	if len(runes) <= 2 {
		return strings.Repeat("*", len(runes))
	}
	if len(runes) <= 4 {
		return string(runes[:1]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1:])
	}
	return string(runes[:2]) + strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-2:])
}
