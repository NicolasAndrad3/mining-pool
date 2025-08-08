package utils

import (
	"context"
)

type contextKey string

const requestIDKey contextKey = "request_id"

func GetRequestID(ctx context.Context) string {
	val := ctx.Value(requestIDKey)
	if id, ok := val.(string); ok {
		return id
	}
	return ""
}

func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}
