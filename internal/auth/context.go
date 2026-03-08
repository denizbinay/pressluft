package auth

import "context"

type ActorType string

const (
	ActorTypeOperator ActorType = "operator"
)

type Role string

const (
	RoleAdmin Role = "admin"
)

type Actor struct {
	ID            string       `json:"id"`
	Type          ActorType    `json:"type"`
	Email         string       `json:"email"`
	Role          Role         `json:"role"`
	Capabilities  []Capability `json:"capabilities,omitempty"`
	Authenticated bool         `json:"authenticated"`
	AuthSource    string       `json:"auth_source,omitempty"`
}

func AnonymousActor() Actor {
	return Actor{Type: ActorTypeOperator}
}

func (a Actor) IsAuthenticated() bool {
	return a.Authenticated && a.ID != ""
}

type contextKey string

const actorContextKey contextKey = "pressluft/auth/actor"

func ContextWithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey, actor)
}

func ActorFromContext(ctx context.Context) Actor {
	if ctx == nil {
		return AnonymousActor()
	}
	actor, ok := ctx.Value(actorContextKey).(Actor)
	if !ok {
		return AnonymousActor()
	}
	return actor
}
