package resource

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResourceObjectKeyUsesUUIDAndSanitizedSegments(t *testing.T) {
	key, err := newResourceObjectKey("oidc/user 1", "lecture notes/intro.txt")

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(key, "resources/oidc-user_1/"))
	assert.True(t, strings.HasSuffix(key, "-lecture_notes-intro.txt"))
	assert.NotContains(t, key, "../")
	assert.Len(t, strings.Split(key, "/"), 3)

	objectID := strings.TrimSuffix(strings.TrimPrefix(key, "resources/oidc-user_1/"), "-lecture_notes-intro.txt")
	_, err = uuid.Parse(objectID)
	require.NoError(t, err)
}

func TestSanitizeObjectKeySegmentDefaultsUnsafeEmptyValues(t *testing.T) {
	assert.Equal(t, "unknown", sanitizeObjectKeySegment(" "))
	assert.Equal(t, "unknown", sanitizeObjectKeySegment("."))
	assert.Equal(t, "unknown", sanitizeObjectKeySegment(".."))
}

func TestResourceOperationsRejectInvalidResourceIDBeforeDependencies(t *testing.T) {
	ctx := context.Background()
	svc := &Service{}

	for _, resourceID := range []int64{0, -1} {
		item, err := svc.GetResource(ctx, resourceID, "viewer")
		require.ErrorIs(t, err, ErrResourceIDInvalid)
		assert.Nil(t, item)

		updated, err := svc.UpdateResource(ctx, resourceID, "owner", UpdateRequest{Title: "title"})
		require.ErrorIs(t, err, ErrResourceIDInvalid)
		assert.Nil(t, updated)

		err = svc.DeleteResource(ctx, resourceID, "owner")
		require.ErrorIs(t, err, ErrResourceIDInvalid)

		url, err := svc.GetDownloadURL(ctx, resourceID, "viewer")
		require.ErrorIs(t, err, ErrResourceIDInvalid)
		assert.Empty(t, url)
	}
}

func TestResourceWritesRejectUnknownVisibilityBeforeExternalEffects(t *testing.T) {
	ctx := context.Background()
	svc := &Service{}

	created, err := svc.CreateResource(ctx, "owner", CreateRequest{
		Title:       "private notes",
		Visibility:  "privtae",
		Filename:    "notes.txt",
		ContentType: "text/plain",
		DataBase64:  base64.StdEncoding.EncodeToString([]byte("private")),
	})
	require.ErrorIs(t, err, ErrResourceVisibilityInvalid)
	assert.Nil(t, created)

	updated, err := svc.UpdateResource(ctx, 1, "owner", UpdateRequest{
		Title:      "private notes",
		Visibility: "internal",
	})
	require.ErrorIs(t, err, ErrResourceVisibilityInvalid)
	assert.Nil(t, updated)
}

func TestDecodePayloadPreservesSupportedMIMERefinements(t *testing.T) {
	zipContent := []byte("PK\x03\x04resource-container")
	oleContent := append(
		[]byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1},
		[]byte("legacy-office-container")...,
	)

	tests := []struct {
		name         string
		provided     string
		content      []byte
		expectedMIME string
	}{
		{
			name:         "exact plain text",
			provided:     "text/plain",
			content:      []byte("plain text"),
			expectedMIME: "text/plain; charset=utf-8",
		},
		{
			name:         "CSV refinement",
			provided:     "text/csv; charset=utf-8",
			content:      []byte("course,score\nmath,95\n"),
			expectedMIME: "text/csv",
		},
		{
			name:         "Markdown refinement",
			provided:     "text/markdown",
			content:      []byte("# Lecture notes\n"),
			expectedMIME: "text/markdown",
		},
		{
			name:         "tab separated refinement",
			provided:     "text/tab-separated-values",
			content:      []byte("course\tscore\nmath\t95\n"),
			expectedMIME: "text/tab-separated-values",
		},
		{
			name:         "validated JSON refinement",
			provided:     "application/json",
			content:      append([]byte{0xef, 0xbb, 0xbf}, []byte(`{"course":"math"}`)...),
			expectedMIME: "application/json",
		},
		{
			name:         "Windows ZIP alias",
			provided:     "application/x-zip-compressed",
			content:      zipContent,
			expectedMIME: "application/zip",
		},
		{
			name:         "DOCX container",
			provided:     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			content:      zipContent,
			expectedMIME: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
		{
			name:         "XLSX container",
			provided:     "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			content:      zipContent,
			expectedMIME: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		},
		{
			name:         "PPTX container",
			provided:     "application/vnd.openxmlformats-officedocument.presentationml.presentation",
			content:      zipContent,
			expectedMIME: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		},
		{
			name:         "OpenDocument text container",
			provided:     "application/vnd.oasis.opendocument.text",
			content:      zipContent,
			expectedMIME: "application/vnd.oasis.opendocument.text",
		},
		{
			name:         "OpenDocument spreadsheet container",
			provided:     "application/vnd.oasis.opendocument.spreadsheet",
			content:      zipContent,
			expectedMIME: "application/vnd.oasis.opendocument.spreadsheet",
		},
		{
			name:         "OpenDocument presentation container",
			provided:     "application/vnd.oasis.opendocument.presentation",
			content:      zipContent,
			expectedMIME: "application/vnd.oasis.opendocument.presentation",
		},
		{
			name:         "EPUB container",
			provided:     "application/epub+zip",
			content:      zipContent,
			expectedMIME: "application/epub+zip",
		},
		{
			name:         "Java archive container",
			provided:     "application/java-archive",
			content:      zipContent,
			expectedMIME: "application/java-archive",
		},
		{
			name:         "legacy DOC container",
			provided:     "application/msword",
			content:      oleContent,
			expectedMIME: "application/msword",
		},
		{
			name:         "legacy XLS container",
			provided:     "application/vnd.ms-excel",
			content:      oleContent,
			expectedMIME: "application/vnd.ms-excel",
		},
		{
			name:         "legacy PPT container",
			provided:     "application/vnd.ms-powerpoint",
			content:      oleContent,
			expectedMIME: "application/vnd.ms-powerpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, contentType, err := decodePayload(
				tt.provided,
				base64.StdEncoding.EncodeToString(tt.content),
			)

			require.NoError(t, err)
			assert.Equal(t, tt.content, content)
			assert.Equal(t, tt.expectedMIME, contentType)
		})
	}
}

func TestDecodePayloadRejectsUnsupportedMIMEClaims(t *testing.T) {
	tests := []struct {
		name     string
		provided string
		content  []byte
	}{
		{
			name:     "contradictory image declaration",
			provided: "image/png",
			content:  []byte("%PDF-1.7\n"),
		},
		{
			name:     "arbitrary text subtype",
			provided: "text/x-python",
			content:  []byte("print('not an allowlisted resource type')"),
		},
		{
			name:     "invalid JSON refinement",
			provided: "application/json",
			content:  []byte("not valid JSON"),
		},
		{
			name:     "arbitrary ZIP refinement",
			provided: "application/vnd.example+zip",
			content:  []byte("PK\x03\x04unknown-container"),
		},
		{
			name:     "legacy Office declaration without OLE signature",
			provided: "application/msword",
			content:  []byte{0x00, 0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "invalid media type",
			provided: "not a media type",
			content:  []byte("plain text"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, contentType, err := decodePayload(
				tt.provided,
				base64.StdEncoding.EncodeToString(tt.content),
			)

			require.ErrorIs(t, err, ErrResourceContentTypeMismatch)
			assert.Nil(t, content)
			assert.Empty(t, contentType)
		})
	}
}
