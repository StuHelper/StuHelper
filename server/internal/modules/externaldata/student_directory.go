package externaldata

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrStudentSourceNotConfigured = errors.New("student source not configured")

type StudentRecord struct {
	SchoolCode  string
	StudentID   string
	StudentName string
}

type StudentDirectory interface {
	LookupStudent(ctx context.Context, studentID string) (*StudentRecord, error)
}

type StudentSource struct {
	Name       string
	SchoolCode string
	Directory  StudentDirectory
}

type StudentDirectoryRegistry struct {
	sources map[string]StudentSource
	closers []io.Closer
}

func NewStudentDirectoryRegistry(sources []StudentSource) (*StudentDirectoryRegistry, error) {
	registry := &StudentDirectoryRegistry{
		sources: map[string]StudentSource{},
	}
	for _, source := range sources {
		normalized, err := normalizeSchoolCode(source.SchoolCode)
		if err != nil {
			return nil, fmt.Errorf("register student source %q: %w", source.Name, err)
		}
		if source.Directory == nil {
			return nil, fmt.Errorf("register student source %q: %w", source.Name, ErrStudentSourceNotConfigured)
		}
		if _, exists := registry.sources[normalized]; exists {
			return nil, fmt.Errorf("duplicate student source for school code %s", normalized)
		}
		source.SchoolCode = normalized
		registry.sources[normalized] = source
		if closer, ok := source.Directory.(io.Closer); ok {
			registry.closers = append(registry.closers, closer)
		}
	}
	if len(registry.sources) == 0 {
		return nil, ErrStudentSourceNotConfigured
	}
	return registry, nil
}

func (r *StudentDirectoryRegistry) LookupStudent(
	ctx context.Context,
	schoolCode string,
	studentID string,
) (*StudentRecord, bool, error) {
	if r == nil {
		return nil, false, nil
	}
	normalized, err := normalizeSchoolCode(schoolCode)
	if err != nil {
		return nil, false, err
	}
	source, ok := r.sources[normalized]
	if !ok {
		return nil, false, nil
	}
	record, err := source.Directory.LookupStudent(ctx, studentID)
	if err != nil {
		return nil, true, err
	}
	if record != nil {
		record.SchoolCode = normalized
	}
	return record, true, nil
}

func (r *StudentDirectoryRegistry) Close() error {
	if r == nil {
		return nil
	}
	var joined error
	for _, closer := range r.closers {
		if err := closer.Close(); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func normalizeSchoolCode(value string) (string, error) {
	code := strings.TrimSpace(value)
	if len(code) != 10 {
		return "", fmt.Errorf("school code must be 10 digits")
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return "", fmt.Errorf("school code must be 10 digits")
		}
	}
	return code, nil
}
