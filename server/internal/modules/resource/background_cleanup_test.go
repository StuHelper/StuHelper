package resource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessCleanupJobNormalizesPayloadBeforeDelete(t *testing.T) {
	store := &fakeObjectStore{}
	svc := &Service{storage: store}

	err := svc.processCleanupJob(context.Background(), cleanupJob{
		JobType: resourceCleanupJobType,
		Payload: []byte(`{"mountID":42,"objectKey":" resources/1/file.txt "}`),
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"resources/1/file.txt"}, store.deletedKeys)
}

func TestProcessCleanupJobRejectsInvalidPayloadBeforeDelete(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
	}{
		{name: "missing mount", payload: `{"objectKey":"resources/1/file.txt"}`},
		{name: "blank object key", payload: `{"mountID":42,"objectKey":" "}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeObjectStore{}
			svc := &Service{storage: store}

			err := svc.processCleanupJob(context.Background(), cleanupJob{
				JobType: resourceCleanupJobType,
				Payload: []byte(tc.payload),
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid resource cleanup payload")
			assert.Empty(t, store.deletedKeys)
		})
	}
}
