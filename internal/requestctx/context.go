package requestctx

import (
	"context"

	"marketplace/internal/domain"
)

type contextKey string

const (
	actorKey       contextKey = "actor"
	requestInfoKey contextKey = "request-info"
)

type Actor struct {
	ID   int64
	Role domain.Role
}

type RequestInfo struct {
	ID     string
	UserID *int64
}

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorKey, actor)
}

func ActorFrom(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorKey).(Actor)
	return actor, ok
}

func WithRequestInfo(ctx context.Context, info *RequestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey, info)
}

func RequestInfoFrom(ctx context.Context) (*RequestInfo, bool) {
	info, ok := ctx.Value(requestInfoKey).(*RequestInfo)
	return info, ok
}
