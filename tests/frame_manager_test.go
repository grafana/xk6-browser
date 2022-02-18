package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/xk6-browser/common"
)

//nolint: funlen
func TestWaitForFrameNavigationWithinDocument(t *testing.T) {
	t.Parallel()

	navHTML := `
<html>
  <head>
    <title>Navigation test within the same document</title>
  </head>
  <body>
    <a id="nav-history" href="#">Navigate with History API</a>
    <a id="nav-anchor" href="#anchor">Navigate with anchor link</a>
    <div id="anchor">Some div...</div>
    <script>
      const el = document.querySelector('a#nav-history');
      el.addEventListener('click', function(evt) {
        evt.preventDefault();
        history.pushState({}, 'navigated', '/nav2');
      });
    </script>
  </body>
</html>
`

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
			tb := newTestBrowser(t, withHTTPServer())
			tb.withHandler("/nav", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(navHTML))
				require.NoError(t, err)
			}))
			p := tb.NewPage(nil)

			resp := p.Goto(tb.URL("/nav"), nil)
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
