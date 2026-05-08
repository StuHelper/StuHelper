package fga

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	relationNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	objectTypePattern   = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	tupleObjectPattern  = regexp.MustCompile(`^[a-z][a-z0-9_-]*:[A-Za-z0-9_-]+$`)
	tupleUserPattern    = regexp.MustCompile(`^[a-z][a-z0-9_-]*:[A-Za-z0-9_-]+(?:#[a-z][a-z0-9_]*)?$`)
)

// validateTupleField 验证 OpenFGA 输入字段格式。
func validateTupleField(field, name string) error {
	if field == "" {
		return fmt.Errorf("fga: %s must not be empty", name)
	}
	if strings.ContainsAny(field, "\x00\n\r") {
		return fmt.Errorf("fga: %s contains invalid characters", name)
	}

	switch name {
	case "user":
		if !tupleUserPattern.MatchString(field) {
			return fmt.Errorf("fga: %s must use type:id or type:id#relation format", name)
		}
	case "object":
		if !tupleObjectPattern.MatchString(field) {
			return fmt.Errorf("fga: %s must use type:id format", name)
		}
	case "relation":
		if !relationNamePattern.MatchString(field) {
			return fmt.Errorf("fga: %s must match [a-z][a-z0-9_]*", name)
		}
	case "object type":
		if !objectTypePattern.MatchString(field) {
			return fmt.Errorf("fga: %s must match [a-z][a-z0-9_]*", name)
		}
	}

	return nil
}
