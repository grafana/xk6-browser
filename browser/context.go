package browser

import (
	"context"
)

type ctxKey int

const (
	ctxKeyOptions ctxKey = iota
	ctxKeyHooks
	ctxKeyIterationID
)

func WithHooks(ctx context.Context, hooks *Hooks) context.Context {
	return context.WithValue(ctx, ctxKeyHooks, hooks)
}

func GetHooks(ctx context.Context) *Hooks {
	v := ctx.Value(ctxKeyHooks)
	if v == nil {
		return nil
	}
	return v.(*Hooks)
}

// WithIterationID adds an identifier for the current iteration to the context.
func WithIterationID(ctx context.Context, iterID string) context.Context {
	return context.WithValue(ctx, ctxKeyIterationID, iterID)
}

// GetIterationID returns the iteration identifier attached to the context.
func GetIterationID(ctx context.Context) string {
	s, _ := ctx.Value(ctxKeyIterationID).(string)
	return s
}

// WithOptions adds the browser options to the context.
func WithOptions(ctx context.Context, opts *Options) context.Context {
	return context.WithValue(ctx, ctxKeyOptions, opts)
}

// GetOptions returns the browser options attached to the context.
func GetOptions(ctx context.Context) *Options {
	v := ctx.Value(ctxKeyOptions)
	if v == nil {
		return nil
	}
	if bo, ok := v.(*Options); ok {
		return bo
	}
	return nil
}

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
