package managerevents_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managerevents "github.com/zestagio/chat-service/internal/server-manager/events"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	"github.com/zestagio/chat-service/internal/types"
)

func TestAdapter_Adapt(t *testing.T) {
	cases := []struct {
		name    string
		ev      eventstream.Event
		expJSON string
	}{
		{
			name: "smoke",
			ev: eventstream.NewNewChatEvent(
				types.MustParse[types.EventID]("d0ffbd36-bc30-11ed-8286-461e464ebed8"),
				types.MustParse[types.RequestID]("cee5f290-bc30-11ed-b7fe-461e464ebed8"),
				types.MustParse[types.ChatID]("31b4dc06-bc31-11ed-93cc-461e464ebed8"),
				types.MustParse[types.UserID]("a322bb38-bc31-11ed-83a2-461e464ebed8"),
				false,
			),
			expJSON: `{
				"eventId": "d0ffbd36-bc30-11ed-8286-461e464ebed8",
				"eventType": "NewChatEvent",
				"canTakeMoreProblems": false,
				"chatId": "31b4dc06-bc31-11ed-93cc-461e464ebed8",
				"clientId": "a322bb38-bc31-11ed-83a2-461e464ebed8",
				"requestId": "cee5f290-bc30-11ed-b7fe-461e464ebed8"
			}`,
		},

		{
			name: "new chat message",
			ev: eventstream.NewNewMessageEvent(
				types.MustParse[types.EventID]("d0ffbd36-bc30-11ed-8286-461e464ebed8"),
				types.MustParse[types.RequestID]("cee5f290-bc30-11ed-b7fe-461e464ebed8"),
				types.MustParse[types.ChatID]("31b4dc06-bc31-11ed-93cc-461e464ebed8"),
				types.MustParse[types.MessageID]("cb36a888-bc30-11ed-b843-461e464ebed8"),
				types.MustParse[types.UserID]("a322bb38-bc31-11ed-83a2-461e464ebed8"),
				time.Unix(2, 2).UTC(),
				"Hello, manager!",
				false,
			),
			expJSON: `{
				"authorId": "a322bb38-bc31-11ed-83a2-461e464ebed8",
				"body": "Hello, manager!",
				"chatId": "31b4dc06-bc31-11ed-93cc-461e464ebed8",
				"createdAt": "1970-01-01T00:00:02.000000002Z",
				"eventId": "d0ffbd36-bc30-11ed-8286-461e464ebed8",
				"eventType": "NewMessageEvent",
				"messageId": "cb36a888-bc30-11ed-b843-461e464ebed8",
				"requestId": "cee5f290-bc30-11ed-b7fe-461e464ebed8"
			}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			adapted, err := managerevents.Adapter{}.Adapt(tt.ev)
			require.NoError(t, err)

			raw, err := json.Marshal(adapted)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expJSON, string(raw))
		})
	}
}
