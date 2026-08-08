package studentverification

import (
	"fmt"
	"strings"

	appcrypto "github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
)

const (
	RosterHMACKeyVersion = 1
	rosterBlindIndexV1   = "student-roster:v1"
)

type BlindIndexScope string

const (
	BlindIndexStudentID      BlindIndexScope = "student_id"
	BlindIndexName           BlindIndexScope = "name"
	BlindIndexDocumentNumber BlindIndexScope = "document_number"
	BlindIndexPhone          BlindIndexScope = "phone"
	BlindIndexSubject        BlindIndexScope = "enrollment_subject"
)

// ComputeRosterBlindIndex is shared by the isolated roster writer and online
// equality checks. Values must already be normalized by the selected school
// adapter. Domain separation prevents blind indexes from being reused across
// fields or unrelated features.
func ComputeRosterBlindIndex(key []byte, schoolID int64, scope BlindIndexScope, normalized string) (string, error) {
	if schoolID <= 0 || strings.TrimSpace(string(scope)) == "" || normalized == "" {
		return "", fmt.Errorf("invalid roster blind index input")
	}
	payload := fmt.Sprintf("%s:%d:%s:%s", rosterBlindIndexV1, schoolID, scope, normalized)
	return appcrypto.HMACHashWithKey(payload, key)
}
