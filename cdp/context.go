package cdp

import "context"

type ctxKey int

const (
	ctxKeySessionID ctxKey = iota
)

func withSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, ctxKeySessionID, sessionID)
}

func getSessionID(ctx context.Context) string {
	v := ctx.Value(ctxKeySessionID)
	if sid, ok := v.(string); ok {
		return sid
	}
	return ""
}
