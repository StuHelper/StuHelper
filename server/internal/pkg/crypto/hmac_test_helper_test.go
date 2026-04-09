package crypto

import "sync"

// ResetHMACForTesting resets the HMAC key and sync.Once so that tests
// can call InitHMACKey with different keys in isolation.
// This function is only available in test binaries (_test.go).
func ResetHMACForTesting() {
	hmacKeyOnce = sync.Once{}
	hmacKey = nil
}
