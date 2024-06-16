package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"

	"github.com/zestagio/chat-service/internal/types"
)

// Message holds the schema definition for the Message entity.
type Message struct {
	ent.Schema
}

// Fields of the Message.
func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", types.MessageID{}).Immutable().Default(types.NewMessageID).Unique(),
		field.UUID("chat_id", types.ChatID{}).Immutable(),
		field.UUID("problem_id", types.ProblemID{}).Immutable(),
		field.UUID("author_id", types.UserID{}).Optional(),
		field.Bool("is_visible_for_client").Default(true),
		field.Bool("is_visible_for_manager").Default(false),
		field.String("body").MaxLen(3000).NotEmpty(),
		field.Time("checked_at").Optional(),
		field.Bool("is_blocked").Immutable().Default(false),
		field.Bool("is_service").Immutable().Default(false),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

// Edges of the Message.
func (Message) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("chat", Chat.Type).Ref("messages").Unique().Required().Field("chat_id").Immutable(),
		edge.From("problem", Problem.Type).Ref("messages").Unique().Required().Field("problem_id").Immutable(),
	}
}
