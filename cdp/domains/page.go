package domains

import (
	"context"
	"fmt"

	"github.com/chromedp/cdproto/cdp"
	cdpp "github.com/chromedp/cdproto/page"
)

// Page exposes all the CDP Page domain actions.
type Page interface {
	Enable(context.Context) error
	Navigate(ctx context.Context, url, referrer, frameID string) (docID string, err error)
}

var _ Page = &page{}

type page struct {
	exec cdp.Executor
}

// NewPage returns a new CDP Page domain wrapper.
func NewPage(exec cdp.Executor) Page {
	return &page{exec}
}

func (p *page) Enable(ctx context.Context) error {
	action := cdpp.Enable()
	if err := action.Do(cdp.WithExecutor(ctx, p.exec)); err != nil {
		return fmt.Errorf("enabling page CDP domain: %w", err)
	}

	return nil
}

func (p *page) Navigate(ctx context.Context, url, referrer, frameID string) (string, error) {
	action := cdpp.Navigate(url).WithReferrer(referrer).WithFrameID(cdp.FrameID(frameID))

	_, documentID, errorText, err := action.Do(cdp.WithExecutor(ctx, p.exec))
	if err != nil {
		err = fmt.Errorf("%s at %q: %w", errorText, url, err)
	}

	return documentID.String(), err
}
