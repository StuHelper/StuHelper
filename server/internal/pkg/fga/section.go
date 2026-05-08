package fga

import (
	"strconv"
	"strings"
)

const (
	sectionIDPrefix = "school_"
)

type SectionKind string

const SectionKindReviewModeration SectionKind = "review_moderation"

var knownSectionKinds = []SectionKind{
	SectionKindReviewModeration,
}

// ReviewModerationSectionID returns the synthetic OpenFGA section ID used for
// review moderation within one school.
func ReviewModerationSectionID(schoolID string) string {
	return SyntheticSectionID(schoolID, SectionKindReviewModeration)
}

// ParseReviewModerationSectionID extracts the school ID from a synthetic
// review-moderation section ID.
func ParseReviewModerationSectionID(sectionID string) (string, bool) {
	schoolID, kind, ok := ParseSyntheticSectionID(sectionID)
	return schoolID, ok && kind == SectionKindReviewModeration
}

func SyntheticSectionID(schoolID string, kind SectionKind) string {
	return sectionIDPrefix + strings.TrimSpace(schoolID) + "_" + string(kind)
}

func ParseSyntheticSectionID(sectionID string) (string, SectionKind, bool) {
	raw, ok := strings.CutPrefix(sectionID, sectionIDPrefix)
	if !ok {
		return "", "", false
	}
	for _, kind := range knownSectionKinds {
		suffix := "_" + string(kind)
		schoolID, ok := strings.CutSuffix(raw, suffix)
		if !ok {
			continue
		}
		if !validNumericSchoolID(schoolID) {
			return "", "", false
		}
		return schoolID, kind, true
	}
	return "", "", false
}

func validNumericSchoolID(raw string) bool {
	if raw == "" {
		return false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	return err == nil && id > 0 && strconv.FormatInt(id, 10) == raw
}
