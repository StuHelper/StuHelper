package review

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateReply_PreflightContentValidation(t *testing.T) {
	svc := &Service{}

	_, err := svc.CreateReply(context.Background(), CreateReplyParams{Content: "   "})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentEmpty)

	_, err = svc.CreateReply(context.Background(), CreateReplyParams{Content: `<script>alert(1)</script>`})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDangerousContent)

	_, err = svc.CreateReply(context.Background(), CreateReplyParams{
		Content: strings.Repeat("字", maxReplyContentRunes+1),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrContentTooLong)
}
