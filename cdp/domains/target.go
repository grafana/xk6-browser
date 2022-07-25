package domains

import (
	"context"
	"fmt"

	"github.com/chromedp/cdproto/cdp"
	cdpt "github.com/chromedp/cdproto/target"
)

type Target interface {
	CreateBrowserContext(ctx context.Context, disposeOnDetach bool) (id string, err error)
	DisposeBrowserContext(ctx context.Context, id string) error
	SetAutoAttach(ctx context.Context, autoAttach, waitForDebuggerOnStart, flatten bool) error
}

var _ Target = &target{}

type target struct {
	exec cdp.Executor
}

// NewTarget returns a new CDP Target domain wrapper.
func NewTarget(exec cdp.Executor) Target {
	return &target{exec}
}

func (t *target) CreateBrowserContext(ctx context.Context, disposeOnDetach bool) (id string, err error) {
	action := cdpt.CreateBrowserContext().WithDisposeOnDetach(disposeOnDetach)
	bctxID, err := action.Do(cdp.WithExecutor(ctx, t.exec))
	if err != nil {
		return "", err
	}

	return string(bctxID), nil
}

func (t *target) DisposeBrowserContext(ctx context.Context, id string) error {
	action := cdpt.DisposeBrowserContext(cdp.BrowserContextID(id))
	if err := action.Do(cdp.WithExecutor(ctx, t.exec)); err != nil {
		return err
	}

	return nil
}

// SetAutoAttach executes the CDP Target.setAutoAttach command.
func (t *target) SetAutoAttach(ctx context.Context, autoAttach, waitForDebuggerOnStart, flatten bool) error {
	action := cdpt.SetAutoAttach(autoAttach, waitForDebuggerOnStart).WithFlatten(flatten)
	if err := action.Do(cdp.WithExecutor(ctx, t.exec)); err != nil {
		return fmt.Errorf("executing setAutoAttach: %w", err)
	}

	// Target.setAutoAttach has a bug where it does not wait for new Targets being attached.
	// However making a dummy call afterwards fixes this.
	// This can be removed after https://chromium-review.googlesource.com/c/chromium/src/+/2885888 lands in stable.
	action2 := cdpt.GetTargetInfo()
	if _, err := action2.Do(cdp.WithExecutor(ctx, t.exec)); err != nil {
		return fmt.Errorf("executing getTargetInfo: %w", err)
	}

	return nil
}
