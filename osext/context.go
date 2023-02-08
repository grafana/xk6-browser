package osext

import (
	"context"
)

type ctxKey int

const (
	ctxKeyRunID ctxKey = iota
)

// WithRunID saves the current iteration ID to the context.
func WithRunID(ctx context.Context, rID string) context.Context {
	return context.WithValue(ctx, ctxKeyRunID, rID)
}

// GetRunID returns the current iteration ID from the context.
func GetRunID(ctx context.Context) string {
	rID, _ := ctx.Value(ctxKeyRunID).(string)
	return rID
}
