package domains

import (
	"context"

	cdpb "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
)

type Browser interface {
	Close(ctx context.Context) error
	GetVersion(ctx context.Context) (
		protocolVersion, product, revision, userAgent, jsVersion string, err error,
	)
}

var _ Browser = &browser{}

type browser struct {
	exec cdp.Executor
}

// NewBrowser returns a new CDP Browser domain wrapper.
func NewBrowser(exec cdp.Executor) Browser {
	return &browser{exec}
}

func (b *browser) Close(ctx context.Context) error {
	action := cdpb.Close()
	return action.Do(cdp.WithExecutor(ctx, b.exec))
}

func (b *browser) GetVersion(ctx context.Context) (
	protocolVersion, product, revision, userAgent, jsVersion string, err error,
) {
	action := cdpb.GetVersion()
	return action.Do(cdp.WithExecutor(ctx, b.exec))
}
