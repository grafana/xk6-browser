package cdp

import (
	"fmt"

	"github.com/chromedp/cdproto/cdp"
	cdppage "github.com/chromedp/cdproto/page"
)

// TODO: Break actions apart into separate CDP domains.

// PageNavigate executes the CDP Page.navigate command.
func (c *Client) PageNavigate(url, referrer, frameID, sessionID string) (string, error) {
	ctx := withSessionID(c.ctx, sessionID)
	action := cdppage.Navigate(url).WithReferrer(referrer).WithFrameID(cdp.FrameID(frameID))

	_, documentID, errorText, err := action.Do(cdp.WithExecutor(ctx, c))
	if err != nil {
		err = fmt.Errorf("%s at %q: %w", errorText, url, err)
	}

	return documentID.String(), err
}
