package tests

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/xk6-browser/api"
	"github.com/grafana/xk6-browser/common"
)

func TestFramePress(t *testing.T) {
	tb := newTestBrowser(t)

	p := tb.NewPage(nil)

	p.SetContent(`<input id="text1">`, nil)

	f := p.Frames()[0]

	f.Press("#text1", "Shift+KeyA", nil)
	f.Press("#text1", "KeyB", nil)
	f.Press("#text1", "Shift+KeyC", nil)

	require.Equal(t, "AbC", f.InputValue("#text1", nil))
}

func TestLifecycleNetworkIdle(t *testing.T) {
	t.Parallel()

	assertHome := func(tb *testBrowser, p api.Page, check func()) {
		var resolved, rejected bool
		err := tb.await(func() error {
			opts := tb.toGojaValue(common.FrameGotoOptions{
				WaitUntil: common.LifecycleEventNetworkIdle,
				Timeout:   30 * time.Second,
			})
			tb.promise(p.Goto(tb.URL("/home"), opts)).then(
				func() {
					check()
					resolved = true
				},
				func() {
					rejected = true
				},
			)

			return nil
		})
		require.NoError(t, err)

		assert.True(t, resolved)
		assert.False(t, rejected)
	}

	t.Run("doesn't timeout waiting for networkIdle", func(t *testing.T) {
		t.Parallel()

		var (
			tb = newTestBrowser(t, withHTTPServer())
			p  = tb.NewPage(nil)
		)
		tb.withHandler("/home", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(w, `
			<html>
				<head></head>
				<body>
					<div id="serverMsg">Waiting...</div>
					<script src="/ping.js" async></script>
				</body>
			</html>
			`)
		})
		tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(w, `
				var serverMsgOutput = document.getElementById("serverMsg");
				serverMsgOutput.innerText = "ping.js loaded from server";
			`)
		})
		assertHome(tb, p, func() {
			result := p.TextContent("#serverMsg", nil)
			assert.EqualValues(t, "ping.js loaded from server", result)
		})
	})

	t.Run("doesn't unblock wait for networkIdle too early", func(t *testing.T) {
		t.Parallel()

		var (
			counter int64
			tb      = newTestBrowser(t, withFileServer())
			p       = tb.NewPage(nil)
			ch      = make(chan bool)
		)
		tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, tb.staticURL("prolonged_network_idle.html"), http.StatusMovedPermanently)
		})
		tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
			<-ch

			time.Sleep(time.Millisecond * 50)
			atomic.AddInt64(&counter, 1)
			fmt.Fprintf(w, "pong %d", atomic.LoadInt64(&counter))
		})
		tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(w, `
				var serverMsgOutput = document.getElementById("serverMsg");
				serverMsgOutput.innerText = "ping.js loaded from server";
			`)
			close(ch)
		})
		assertHome(tb, p, func() {
			result := p.TextContent("#prolongNetworkIdleLoad", nil)
			assert.EqualValues(t, "Waiting... pong 4 - for loop complete", result)

			result = p.TextContent("#serverMsg", nil)
			assert.EqualValues(t, "ping.js loaded from server", result)
		})
	})

	t.Run("doesn't unblock wait on networkIdle early when load and domcontentloaded complete at once", func(t *testing.T) {
		t.Parallel()

		var (
			counter int64
			tb      = newTestBrowser(t, withFileServer())
			p       = tb.NewPage(nil)
		)
		tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, tb.staticURL("prolonged_network_idle_10.html"), http.StatusMovedPermanently)
		})
		tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(time.Millisecond * 50)
			atomic.AddInt64(&counter, 1)
			fmt.Fprintf(w, "pong %d", atomic.LoadInt64(&counter))
		})
		assertHome(tb, p, func() {
			result := p.TextContent("#prolongNetworkIdleLoad", nil)
			assert.EqualValues(t, "Waiting... pong 10 - for loop complete", result)
		})
	})
}
