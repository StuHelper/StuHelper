package casdoor

import (
	"fmt"
	"strings"
)

func validateName(label, value string) error {
	if value == "" {
		return fmt.Errorf("casdoor: %s is required", label)
	}
	if strings.ContainsAny(value, "\x00\n\r\t/\\") {
		return fmt.Errorf("casdoor: %s contains invalid characters", label)
	}
	return nil
}

func normalizeNonEmptyList(label string, values []string) ([]string, error) {
	normalized, err := normalizeList(label, values)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("casdoor: at least one %s is required", label)
	}
	return normalized, nil
}

func normalizeList(label string, values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" {
			return nil, fmt.Errorf("casdoor: blank %s is forbidden", label)
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}
