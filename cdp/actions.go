package cdp

import (
	"fmt"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
)

// TODO: Break actions apart into separate CDP domains.

// PageEnable enables the CDP page domain notifications.
func (c *Client) PageEnable(sessionID string) error {
	ctx := withSessionID(c.ctx, sessionID)
	action := page.Enable()
	if err := action.Do(cdp.WithExecutor(ctx, c)); err != nil {
		return fmt.Errorf("enabling page CDP domain: %w", err)
	}

	return nil
}

// PageNavigate executes the CDP Page.navigate command.
func (c *Client) PageNavigate(url, referrer, frameID, sessionID string) (string, error) {
	ctx := withSessionID(c.ctx, sessionID)
	action := page.Navigate(url).WithReferrer(referrer).WithFrameID(cdp.FrameID(frameID))

	_, documentID, errorText, err := action.Do(cdp.WithExecutor(ctx, c))
	if err != nil {
		err = fmt.Errorf("%s at %q: %w", errorText, url, err)
	}

	return documentID.String(), err
}

// TargetSetAutoAttach executes the CDP Target.setAutoAttach command.
func (c *Client) TargetSetAutoAttach(autoAttach, waitForDebuggerOnStart, flatten bool) error {
	action := target.SetAutoAttach(autoAttach, waitForDebuggerOnStart).WithFlatten(flatten)
	if err := action.Do(cdp.WithExecutor(c.ctx, c)); err != nil {
		return fmt.Errorf("executing setAutoAttach: %w", err)
	}

	// Target.setAutoAttach has a bug where it does not wait for new Targets being attached.
	// However making a dummy call afterwards fixes this.
	// This can be removed after https://chromium-review.googlesource.com/c/chromium/src/+/2885888 lands in stable.
	action2 := target.GetTargetInfo()
	if _, err := action2.Do(cdp.WithExecutor(c.ctx, c)); err != nil {
		return fmt.Errorf("executing getTargetInfo: %w", err)
	}

	return nil
}
