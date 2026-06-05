package email

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	UseTLS   bool
	StartTLS bool
	Timeout  time.Duration
}

type SMTPSender struct {
	cfg  SMTPConfig
	from mail.Address
}

func NewSMTPSender(cfg SMTPConfig) (*SMTPSender, error) {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.From = strings.TrimSpace(cfg.From)
	cfg.FromName = strings.TrimSpace(cfg.FromName)
	cfg.Username = strings.TrimSpace(cfg.Username)
	if cfg.Host == "" {
		return nil, errors.New("smtp host is required")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("smtp port out of range: %d", cfg.Port)
	}
	if cfg.From == "" {
		return nil, errors.New("email from is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	from := mail.Address{Name: cfg.FromName, Address: cfg.From}
	return &SMTPSender{cfg: cfg, from: from}, nil
}

func (s *SMTPSender) Send(ctx context.Context, to string, subject string, textBody string) (err error) {
	if s == nil {
		return errors.New("smtp sender is nil")
	}
	recipient, err := mail.ParseAddress(strings.TrimSpace(to))
	if err != nil {
		return fmt.Errorf("parse recipient: %w", err)
	}
	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))
	conn, err := s.dial(ctx, addr)
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			return fmt.Errorf("smtp client: %w; close connection: %v", err, closeErr)
		}
		return fmt.Errorf("smtp client: %w", err)
	}
	clientClosed := false
	defer func() {
		if clientClosed {
			return
		}
		if closeErr := client.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("smtp close client: %w", closeErr)
		}
	}()

	if s.cfg.StartTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}
	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(s.from.Address); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	message, err := buildTextMessage(s.from, *recipient, subject, textBody)
	if err != nil {
		return fmt.Errorf("build smtp message: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		if closeErr := writer.Close(); closeErr != nil {
			return fmt.Errorf("smtp write: %w; close data: %v", err, closeErr)
		}
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	clientClosed = true
	return nil
}

func (s *SMTPSender) dial(ctx context.Context, addr string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: s.cfg.Timeout}
	if s.cfg.UseTLS {
		tlsDialer := tls.Dialer{
			NetDialer: &dialer,
			Config:    &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12},
		}
		conn, err := tlsDialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("smtp tls dial: %w", err)
		}
		return conn, nil
	}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("smtp dial: %w", err)
	}
	return conn, nil
}

func buildTextMessage(from mail.Address, to mail.Address, subject string, textBody string) ([]byte, error) {
	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)
	if err := writeHeader(writer, "From", from.String()); err != nil {
		return nil, err
	}
	if err := writeHeader(writer, "To", to.String()); err != nil {
		return nil, err
	}
	if err := writeHeader(writer, "Subject", mime.QEncoding.Encode("UTF-8", strings.TrimSpace(subject))); err != nil {
		return nil, err
	}
	if err := writeHeader(writer, "MIME-Version", "1.0"); err != nil {
		return nil, err
	}
	if err := writeHeader(writer, "Content-Type", `text/plain; charset="UTF-8"`); err != nil {
		return nil, err
	}
	if err := writeHeader(writer, "Content-Transfer-Encoding", "8bit"); err != nil {
		return nil, err
	}
	if _, err := writer.WriteString("\r\n"); err != nil {
		return nil, err
	}
	if _, err := writer.WriteString(strings.TrimSpace(textBody)); err != nil {
		return nil, err
	}
	if _, err := writer.WriteString("\r\n"); err != nil {
		return nil, err
	}
	if err := writer.Flush(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeHeader(writer *bufio.Writer, key string, value string) error {
	if _, err := writer.WriteString(key); err != nil {
		return err
	}
	if _, err := writer.WriteString(": "); err != nil {
		return err
	}
	if _, err := writer.WriteString(strings.ReplaceAll(value, "\n", " ")); err != nil {
		return err
	}
	if _, err := writer.WriteString("\r\n"); err != nil {
		return err
	}
	return nil
}
