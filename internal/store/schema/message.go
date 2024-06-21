package schema

import (
	"entgo.io/ent/schema/index"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"

	"github.com/zestagio/chat-service/internal/types"
)

const messageBodyMaxLength = 3000

// Message holds the schema definition for the Message entity.
type Message struct {
	ent.Schema
}

// Fields of the Message.
func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", types.MessageID{}).Default(types.NewMessageID).Unique().Immutable(),
		field.UUID("chat_id", types.ChatID{}),
		field.UUID("problem_id", types.ProblemID{}),
		field.UUID("initial_request_id", types.RequestID{}).Optional().Unique(),
		field.UUID("author_id", types.UserID{}).Optional().Immutable(),
		field.Bool("is_visible_for_client").Default(false),
		field.Bool("is_visible_for_manager").Default(false),
		field.Text("body").NotEmpty().MaxLen(messageBodyMaxLength).Immutable(),
		field.Time("checked_at").Optional(),
		field.Bool("is_blocked").Default(false),
		field.Bool("is_service").Default(false).Immutable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Message.
func (Message) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("chat", Chat.Type).Ref("messages").Field("chat_id").Required().Unique(),
		edge.From("problem", Problem.Type).Ref("messages").Field("problem_id").Required().Unique(),
	}
}

func (Message) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("chat_id"),
		index.Fields("initial_request_id"),
		index.Fields("created_at"),
	}
}
