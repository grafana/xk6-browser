package api

import (
	"context"

	"github.com/dop251/goja"
)

// Browser is the public interface of a CDP browser.
type Browser interface {
	Close()
	Context() BrowserContext
	IsConnected() bool
	NewContext(ctx context.Context, opts goja.Value) (BrowserContext, error)
	NewPage(ctx context.Context, opts goja.Value) (Page, error)
	On(string) (bool, error)
	UserAgent() string
	Version() string
}
