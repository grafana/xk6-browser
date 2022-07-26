package cdp

import "context"

type ctxKey int

const (
	ctxKeySessionID ctxKey = iota
)

func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, ctxKeySessionID, sessionID)
}

func GetSessionID(ctx context.Context) string {
	v := ctx.Value(ctxKeySessionID)
	if sid, ok := v.(string); ok {
		return sid
	}
	return ""
}

// TODO: Copied from common/context.go. DRY?
// contextWithDoneChan returns a new context that is canceled either
// when the done channel is closed or ctx is canceled.
func contextWithDoneChan(ctx context.Context, done chan struct{}) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()
		select {
		case <-done:
		case <-ctx.Done():
		}
	}()
	return ctx
}
