package review

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	sensitiveWordCategoryPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,49}$`)
	allowedSensitiveWordLevels   = map[string]struct{}{
		sensitiveWordLevelBlock: {},
		ContentFlagWarn:         {},
		ContentFlagReview:       {},
	}
)

const (
	maxSensitiveWordRunes   = 100
	sensitiveWordLevelBlock = "block"
)

func validateSensitiveWordText(word string) (string, error) {
	word = strings.TrimSpace(word)
	if word == "" || utf8.RuneCountInString(word) > maxSensitiveWordRunes {
		return "", ErrSensitiveWordInvalid
	}
	return word, nil
}

func validateSensitiveWordCategory(category string) error {
	trimmed := strings.TrimSpace(category)
	if trimmed == "" {
		return nil
	}
	if !sensitiveWordCategoryPattern.MatchString(trimmed) {
		return fmt.Errorf("%w: category must match ^[a-z][a-z0-9_-]{0,49}$", ErrSensitiveWordInvalid)
	}
	return nil
}

func validateSensitiveWordLevel(level string) error {
	trimmed := strings.TrimSpace(level)
	if trimmed == "" {
		return nil
	}
	if _, ok := allowedSensitiveWordLevels[trimmed]; !ok {
		return fmt.Errorf("%w: level must be one of: block, warn, review", ErrSensitiveWordInvalid)
	}
	return nil
}
