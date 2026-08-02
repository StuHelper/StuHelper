package oidc

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

var errDisallowedJWTAlgorithm = errors.New("oidc: disallowed jwt signing algorithm")

type jwtHeader struct {
	Algorithm string `json:"alg"`
}

func validateJWTSigningAlgorithm(rawJWT string) error {
	segments := strings.Split(rawJWT, ".")
	if len(segments) != 3 {
		return errors.New("oidc: invalid jwt compact serialization")
	}

	header, err := decodeJWTHeader(segments[0])
	if err != nil {
		return err
	}
	if !isAllowedJWTSigningAlgorithm(header.Algorithm) {
		return fmt.Errorf("%w: %s", errDisallowedJWTAlgorithm, header.Algorithm)
	}
	return nil
}

func isAllowedJWTSigningAlgorithm(algorithm string) bool {
	return slices.Contains(allowedJWTSigningAlgorithms(), algorithm)
}

func allowedJWTSigningAlgorithms() []string {
	return []string{"RS256", "ES256"}
}

func decodeJWTHeader(encoded string) (jwtHeader, error) {
	var header jwtHeader
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return header, fmt.Errorf("oidc: decode jwt header: %w", err)
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return header, fmt.Errorf("oidc: parse jwt header: %w", err)
	}
	if header.Algorithm == "" {
		return header, errors.New("oidc: jwt alg header is required")
	}
	return header, nil
}
