package main

import (
	"fmt"
	"strconv"
	"strings"

	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
)

func providerSpec(getenv envReader, prefix string) (casdoor.ProviderSpec, bool, error) {
	enabled, err := boolEnv(getenv, prefix+"_ENABLED")
	if err != nil || !enabled {
		return casdoor.ProviderSpec{}, false, err
	}
	fields, err := requiredProviderFields(getenv, prefix)
	if err != nil {
		return casdoor.ProviderSpec{}, false, err
	}
	spec := casdoor.ProviderSpec{
		Name:        fields.name,
		DisplayName: fields.displayName,
		Category:    fields.category,
		Type:        fields.providerType,
		SubType:     strings.TrimSpace(getenv(prefix + "_SUB_TYPE")),
		Method:      strings.TrimSpace(getenv(prefix + "_METHOD")),
		ProviderURL: strings.TrimSpace(getenv(prefix + "_PROVIDER_URL")),
		Endpoint:    strings.TrimSpace(getenv(prefix + "_ENDPOINT")),
		Host:        strings.TrimSpace(getenv(prefix + "_HOST")),
		Title:       strings.TrimSpace(getenv(prefix + "_TITLE")),
		Content:     strings.TrimSpace(getenv(prefix + "_CONTENT")),
		Metadata:    strings.TrimSpace(getenv(prefix + "_METADATA")),
	}
	return finishProviderSpec(getenv, prefix, spec)
}

type providerRequiredFields struct {
	name         string
	displayName  string
	category     string
	providerType string
}

func requiredProviderFields(getenv envReader, prefix string) (providerRequiredFields, error) {
	fields := providerRequiredFields{}
	var err error
	if fields.name, err = requiredValue(getenv, prefix+"_NAME"); err != nil {
		return providerRequiredFields{}, err
	}
	if fields.displayName, err = requiredValue(getenv, prefix+"_DISPLAY_NAME"); err != nil {
		return providerRequiredFields{}, err
	}
	if fields.category, err = requiredValue(getenv, prefix+"_CATEGORY"); err != nil {
		return providerRequiredFields{}, err
	}
	fields.providerType, err = requiredValue(getenv, prefix+"_TYPE")
	return fields, err
}

func finishProviderSpec(getenv envReader, prefix string, spec casdoor.ProviderSpec) (casdoor.ProviderSpec, bool, error) {
	port, err := optionalPort(getenv(prefix + "_PORT"))
	if err != nil {
		return casdoor.ProviderSpec{}, false, err
	}
	spec.Port = port
	spec.DisableSSL, err = boolEnv(getenv, prefix+"_DISABLE_SSL")
	return spec, true, err
}

func optionalPort(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid provider port %q: %w", raw, err)
	}
	return port, nil
}

func boolEnv(getenv envReader, key string) (bool, error) {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", key)
	}
	return parsed, nil
}
