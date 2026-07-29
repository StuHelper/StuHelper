package email

import (
	"bufio"
	"bytes"
	"net/mail"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSMTPSenderValidatesSenderHeaders(t *testing.T) {
	valid, err := NewSMTPSender(SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		From:     "sender@example.com",
		FromName: "StuHelper 通知",
	})
	require.NoError(t, err)
	assert.Equal(t, "sender@example.com", valid.from.Address)
	assert.Equal(t, "StuHelper 通知", valid.from.Name)

	for name, cfg := range map[string]SMTPConfig{
		"address injection": {
			Host: "smtp.example.com",
			Port: 587,
			From: "sender@example.com\r\nBcc: attacker@example.com",
		},
		"display name injection": {
			Host:     "smtp.example.com",
			Port:     587,
			From:     "sender@example.com",
			FromName: "StuHelper\r\nBcc: attacker@example.com",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewSMTPSender(cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "forbidden line break")
		})
	}
}

func TestBuildTextMessageRejectsHeaderInjection(t *testing.T) {
	from := mail.Address{Name: "StuHelper", Address: "sender@example.com"}

	message, err := buildTextMessage(from, "选课提醒", "正文")
	require.NoError(t, err)
	assert.Contains(t, string(message), "Subject: =?UTF-8?")
	assert.Contains(t, string(message), "To: "+undisclosedRecipientsHeader)
	assert.True(t, strings.HasSuffix(string(message), "正文\r\n"))

	for name, mutate := range map[string]func() (mail.Address, string){
		"sender address": func() (mail.Address, string) {
			return mail.Address{
				Name:    "StuHelper",
				Address: "sender@example.com\r\nBcc: attacker@example.com",
			}, "subject"
		},
		"sender display name": func() (mail.Address, string) {
			return mail.Address{
				Name:    "StuHelper\r\nBcc: attacker@example.com",
				Address: "sender@example.com",
			}, "subject"
		},
		"subject": func() (mail.Address, string) {
			return from, "subject\r\nBcc: attacker@example.com"
		},
	} {
		t.Run(name, func(t *testing.T) {
			gotFrom, subject := mutate()
			_, err := buildTextMessage(gotFrom, subject, "body")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "forbidden line break")
		})
	}
}

func TestParseHeaderAddressRejectsRecipientInjection(t *testing.T) {
	recipient, err := parseHeaderAddress("recipient", "Student <student@example.com>")
	require.NoError(t, err)
	assert.Equal(t, "student@example.com", recipient.Address)

	for _, value := range []string{
		"student@example.com\r\nBcc: attacker@example.com",
		"Student\r\nBcc: attacker@example.com <student@example.com>",
	} {
		_, err := parseHeaderAddress("recipient", value)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden line break")
	}
}

func TestWriteHeaderRejectsInvalidNamesAndValues(t *testing.T) {
	for _, testCase := range []struct {
		key   string
		value string
	}{
		{key: "X-Test\r\nBcc", value: "value"},
		{key: "X-Test", value: "value\nBcc: attacker@example.com"},
	} {
		var output bytes.Buffer
		err := writeHeader(bufio.NewWriter(&output), testCase.key, testCase.value)
		require.Error(t, err)
		assert.Empty(t, output.String())
	}
}
