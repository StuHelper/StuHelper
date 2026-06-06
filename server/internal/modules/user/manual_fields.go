package user

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

var allowedManualFieldTypes = map[string]struct{}{
	"text":     {},
	"textarea": {},
	"select":   {},
	"date":     {},
}

func decodeManualFieldDescriptors(raw json.RawMessage) ([]ManualFieldDescriptor, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &object); err == nil {
			if len(object) == 0 {
				return nil, nil
			}
			if _, ok := object["admission"]; ok {
				return nil, nil
			}
		}
	}

	var fields []ManualFieldDescriptor
	if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
		return nil, fmt.Errorf("decode manual field descriptors: %w", err)
	}
	if err := validateManualFieldDescriptors(fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func validateManualFieldDescriptors(fields []ManualFieldDescriptor) error {
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			return fmt.Errorf("manual field key must not be empty")
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate manual field key: %s", key)
		}
		seen[key] = struct{}{}

		if strings.TrimSpace(field.Label) == "" {
			return fmt.Errorf("manual field label must not be empty: %s", key)
		}
		if _, ok := allowedManualFieldTypes[field.Type]; !ok {
			return fmt.Errorf("manual field type must be one of %s: %s", strings.Join(sortedManualFieldTypes(), ", "), key)
		}
		if field.Type == "select" && len(field.Options) == 0 {
			return fmt.Errorf("manual select field options must not be empty: %s", key)
		}
	}
	return nil
}

func sortedManualFieldTypes() []string {
	types := slices.Collect(maps.Keys(allowedManualFieldTypes))
	slices.Sort(types)
	return types
}

func getManualFormValueAsString(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func sanitizeManualFormData(data map[string]any) map[string]any {
	if len(data) == 0 {
		return nil
	}
	sanitized := make(map[string]any, len(data))
	for key, value := range data {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			sanitized[trimmedKey] = strings.TrimSpace(typed)
		default:
			sanitized[trimmedKey] = value
		}
	}
	if len(sanitized) == 0 {
		return nil
	}
	return sanitized
}
