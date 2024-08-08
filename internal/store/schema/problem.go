package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"

	"github.com/zestagio/chat-service/internal/types"
)

// Problem holds the schema definition for the Problem entity.
type Problem struct {
	ent.Schema
}

// Fields of the Problem.
func (Problem) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", types.ProblemID{}).Default(types.NewProblemID).Unique().Immutable(),
		field.UUID("chat_id", types.ChatID{}),
		field.UUID("manager_id", types.UserID{}).Optional(),
		field.Time("resolved_at").Optional(),
		field.UUID("resolve_request_id", types.RequestID{}).Optional().Unique(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Problem.
func (Problem) Edges() []ent.Edge {
	return []ent.Edge{
		// The problem has one chat.
		edge.From("chat", Chat.Type).
			Ref("problems").
			Field("chat_id").
			Required().Unique(),

		// The problem has many messages.
		edge.To("messages", Message.Type),
	}
}

func (Problem) Indexes() []ent.Index {
	return []ent.Index{
		// The most part of queries in framework of specific chat.
		index.Fields("chat_id"),

		// Getting open problems for manager.
		index.Fields("manager_id"),
	}
}
