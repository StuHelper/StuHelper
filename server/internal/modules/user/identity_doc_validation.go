package user

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var mainlandIDChecksumWeights = [...]int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}

const mainlandIDChecksumChars = "10X98765432"

func normalizeIdentityDocNumber(docType, docNumber string) string {
	trimmed := strings.TrimSpace(docNumber)
	if docType == DocTypeMainlandID {
		return strings.ToUpper(trimmed)
	}
	return trimmed
}

func isValidIdentityDocNumber(docType, docNumber string) bool {
	switch docType {
	case DocTypeMainlandID:
		return isValidMainlandIDNumber(docNumber)
	case DocTypeHKMacau, DocTypeTW, DocTypePassport:
		return isValidNonMainlandIdentityDocNumber(docNumber)
	default:
		return false
	}
}

func isValidNonMainlandIdentityDocNumber(docNumber string) bool {
	if docNumber == "" || !utf8.ValidString(docNumber) || utf8.RuneCountInString(docNumber) > 50 {
		return false
	}
	return !strings.ContainsFunc(docNumber, func(r rune) bool {
		return unicode.IsSpace(r) ||
			unicode.IsControl(r) ||
			unicode.In(r, unicode.Cf, unicode.Co, unicode.Cs)
	})
}

func isValidMainlandIDNumber(id string) bool {
	if len(id) != 18 {
		return false
	}
	if id[0] == '0' {
		return false
	}
	for i := 0; i < 17; i++ {
		if id[i] < '0' || id[i] > '9' {
			return false
		}
	}
	last := id[17]
	if (last < '0' || last > '9') && last != 'X' {
		return false
	}

	birthDate := id[6:14]
	parsed, err := time.Parse("20060102", birthDate)
	if err != nil || parsed.Format("20060102") != birthDate {
		return false
	}

	sum := 0
	for i, weight := range mainlandIDChecksumWeights {
		sum += int(id[i]-'0') * weight
	}
	return mainlandIDChecksumChars[sum%11] == last
}
