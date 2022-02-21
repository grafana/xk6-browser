package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/xk6-browser/common"
)

func TestWaitForFrameNavigationWithinDocument(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name, selector string
	}{
		{name: "history", selector: "a#nav-history"},
		{name: "anchor", selector: "a#nav-anchor"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			p := tb.NewPage(nil)

			resp := p.Goto(tb.staticURL("/nav_in_doc.html"), nil)
			require.NotNil(t, resp)

			el := p.Query(tc.selector)
			require.NotNil(t, el)
			// A click right away could possibly trigger navigation before we
			// had a chance to call WaitForNavigation below, so give it some
			// time to simulate the JS overhead, waiting for XHR response, etc.
			time.AfterFunc(200*time.Millisecond, func() {
				el.Click(nil)
			})

			done := make(chan struct{})
			go func() {
				defer close(done)
				require.NotPanics(t, func() {
					p.WaitForNavigation(tb.rt.ToValue(&common.FrameWaitForNavigationOptions{
						Timeout: 1000, // 1s
					}))
				})
			}()

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				// WaitForNavigation is stuck?
				close(done)
				t.Fatal("Test timed out")
			}
		})
	}
}
