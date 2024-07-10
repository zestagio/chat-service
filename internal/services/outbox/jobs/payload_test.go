package jobs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sendclientmessagejob "github.com/zestagio/chat-service/internal/services/outbox/jobs"
	"github.com/zestagio/chat-service/internal/types"
)

func TestMarshalPayload_Smoke(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		p, err := sendclientmessagejob.MarshalPayload(types.NewMessageID())
		require.NoError(t, err)
		assert.NotEmpty(t, p)
	})

	t.Run("invalid input", func(t *testing.T) {
		p, err := sendclientmessagejob.MarshalPayload(types.MessageIDNil)
		require.Error(t, err)
		assert.Empty(t, p)
	})
}
