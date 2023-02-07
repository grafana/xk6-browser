package browserprocess

import (
	"context"
)

type ctxKey int

const (
	ctxKeyIterationID ctxKey = iota
)

// WithIterationID saves the current iteration ID to the context.
func WithIterationID(ctx context.Context, iID string) context.Context {
	return context.WithValue(ctx, ctxKeyIterationID, iID)
}

// GetIterationID returns the current iteration ID from the context.
func GetIterationID(ctx context.Context) string {
	iID, _ := ctx.Value(ctxKeyIterationID).(string)
	return iID
}
