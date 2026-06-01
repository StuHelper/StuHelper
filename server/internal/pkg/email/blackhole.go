package email

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
)

type BlackholeSender struct{}

func NewBlackholeSender() *BlackholeSender {
	return &BlackholeSender{}
}

func (s *BlackholeSender) Send(_ context.Context, to string, subject string, textBody string) error {
	if s == nil {
		return fmt.Errorf("blackhole sender is nil")
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(to)); err != nil {
		return fmt.Errorf("parse recipient: %w", err)
	}
	if strings.TrimSpace(subject) == "" {
		return fmt.Errorf("email subject is required")
	}
	if strings.TrimSpace(textBody) == "" {
		return fmt.Errorf("email body is required")
	}
	return nil
}
