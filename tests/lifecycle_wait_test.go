package tests

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/xk6-browser/api"
	"github.com/grafana/xk6-browser/common"
)

func TestLifecycleWaitForNavigationTimeout(t *testing.T) {
	// Test description
	//
	// 1. goto /home and wait for the networkidle lifecycle event.
	// 2. use WaitForNavigation with networkidle.
	//
	// Success criteria: Time out reached after navigation completed and
	//                   wait for lifecycle event set, to signify that
	//                   WaitForNavigation must be set before we navigate
	//                   to a new page.

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("prolonged_network_idle_10.html"), http.StatusMovedPermanently)
	})

	var counterMu sync.Mutex
	var counter int64
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	waitUntil := common.LifecycleEventNetworkIdle
	var resolved, rejected bool
	err := tb.await(func() error {
		opts := tb.toGojaValue(common.FrameGotoOptions{
			WaitUntil: waitUntil,
			Timeout:   30 * time.Second,
		})
		prm := tb.promise(p.Goto(tb.URL("/home"), opts)).then(
			func() testPromise {
				result := p.TextContent("#pingRequestText", nil)
				assert.EqualValues(t, "Waiting... pong 10 - for loop complete", result)

				waitForNav := p.WaitForNavigation(tb.toGojaValue(&common.FrameWaitForNavigationOptions{
					Timeout:   1000,
					WaitUntil: waitUntil,
				}))

				return tb.promise(waitForNav)
			},
		)
		prm.then(
			func() {
				resolved = true
			},
			func() {
				rejected = true
			},
		)

		return nil
	})
	require.NoError(t, err)

	assert.False(t, resolved)
	assert.True(t, rejected)
}

func TestLifecycleWaitForLoadStateLoad(t *testing.T) {
	// Test description
	//
	// 1. goto /home and wait for the load lifecycle event.
	// 2. use WaitForLoadState with load to ensure that load
	//    lifecycle event has already fired.
	//
	// Success criteria: We don't wait for all network requests to
	//                   complete, but we are interested in waiting
	//                   for all async scripts to have fully loaded
	//                   (which is when load is fired). We also want
	//                   to ensure that the load event is stored
	//                   internally, and we don't block on WaitForLoadState.

	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("wait_for_nav_lifecycle.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		time.Sleep(time.Millisecond * 100)

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";
			`)
	})

	waitUntil := common.LifecycleEventLoad
	assertHome(t, tb, p, waitUntil, func() testPromise {
		result := p.TextContent("#pingRequestText", nil)
		assert.NotEqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "ping.js loaded from server", result)

		// This shouldn't block and return after calling hasLifecycleEventFired.
		p.WaitForLoadState(waitUntil.String(), nil)

		return testPromise{}
	}, nil)
}

func TestLifecycleWaitForLoadStateDOMContentLoaded(t *testing.T) {
	// Test description
	//
	// 1. goto /home and wait for the domcontentloaded lifecycle event.
	// 2. use WaitForLoadState with domcontentloaded to ensure that
	//    domcontentloaded lifecycle event has already fired.
	//
	// Success criteria: We don't wait for all network requests or the
	//                   async scripts to complete, and we're only
	//                   interested in the html file being loaded. We
	//                   also want to ensure that the domcontentloaded
	//                   event is stored internally, and we don't block
	//                   on WaitForLoadState.

	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("wait_for_nav_lifecycle.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		time.Sleep(time.Millisecond * 100)

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				await new Promise(resolve => setTimeout(resolve, 1000));

				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";
			`)
	})

	waitUntil := common.LifecycleEventDOMContentLoad
	assertHome(t, tb, p, waitUntil, func() testPromise {
		result := p.TextContent("#pingRequestText", nil)
		assert.NotEqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "Waiting...", result)

		// This shouldn't block and return after calling hasLifecycleEventFired.
		p.WaitForLoadState(waitUntil.String(), nil)

		return testPromise{}
	}, nil)
}

func TestLifecycleWaitForLoadStateNetworkIdle(t *testing.T) {
	// Test description
	//
	// 1. goto /home and wait for the networkidle lifecycle event.
	// 2. use WaitForLoadState with networkidle to ensure that
	//    networkidle lifecycle event has already fired.
	//
	// Success criteria: We wait for all network requests and async
	//                   scripts to complete. We also want to ensure
	//                   that the networkidle event is stored internally,
	//                   and we don't block on WaitForLoadState.

	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("wait_for_nav_lifecycle.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";
			`)
	})

	waitUntil := common.LifecycleEventNetworkIdle
	assertHome(t, tb, p, waitUntil, func() testPromise {
		result := p.TextContent("#pingRequestText", nil)
		assert.EqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "ping.js loaded from server", result)

		// This shouldn't block and return after calling hasLifecycleEventFired.
		p.WaitForLoadState(waitUntil.String(), nil)

		return testPromise{}
	}, nil)
}

func TestLifecycleWaitForLoadStateDOMContentLoadedThenNetworkIdle(t *testing.T) {
	// Test description
	//
	// 1. goto /home and wait for the domcontentloaded lifecycle event.
	// 2. use WaitForLoadState with networkidle to now wait for the
	//    lifecycle event from the browser.
	//
	// Success criteria: We want to quickly move to calling WaitForLoadState
	//                   so that we block until a networkidle lifecycle
	//                   event is received from the browser.

	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("wait_for_nav_lifecycle.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		time.Sleep(time.Millisecond * 100)

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";
			`)
	})

	assertHome(t, tb, p, common.LifecycleEventDOMContentLoad, func() testPromise {
		p.WaitForLoadState(common.LifecycleEventNetworkIdle.String(), nil)

		result := p.TextContent("#pingRequestText", nil)
		assert.EqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "ping.js loaded from server", result)

		return testPromise{}
	}, nil)
}

func TestLifecycleReloadLoad(t *testing.T) {
	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("reload_lifecycle.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		time.Sleep(time.Millisecond * 100)

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";
			`)
	})

	waitUntil := common.LifecycleEventLoad
	assertHome(t, tb, p, waitUntil, func() testPromise {
		result := p.TextContent("#pingRequestText", nil)
		assert.NotEqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "ping.js loaded from server", result)

		opts := tb.toGojaValue(common.PageReloadOptions{
			WaitUntil: waitUntil,
			Timeout:   30 * time.Second,
		})
		p.Reload(opts)

		result = p.TextContent("#pingRequestText", nil)
		assert.NotEqualValues(t, "Waiting... pong 20 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "ping.js loaded from server", result)

		return testPromise{}
	}, nil)
}

func TestLifecycleReloadDOMContentLoaded(t *testing.T) {
	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("reload_lifecycle.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		time.Sleep(time.Millisecond * 100)

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				await new Promise(resolve => setTimeout(resolve, 1000));

				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";
			`)
	})

	waitUntil := common.LifecycleEventDOMContentLoad
	assertHome(t, tb, p, waitUntil, func() testPromise {
		result := p.TextContent("#pingRequestText", nil)
		assert.NotEqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "Waiting...", result)

		opts := tb.toGojaValue(common.PageReloadOptions{
			WaitUntil: waitUntil,
			Timeout:   30 * time.Second,
		})
		p.Reload(opts)

		result = p.TextContent("#pingRequestText", nil)
		assert.NotEqualValues(t, "Waiting... pong 20 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "Waiting...", result)

		return testPromise{}
	}, nil)
}

func TestLifecycleReloadNetworkIdle(t *testing.T) {
	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("reload_lifecycle.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";
			`)
	})

	waitUntil := common.LifecycleEventNetworkIdle
	assertHome(t, tb, p, waitUntil, func() testPromise {
		result := p.TextContent("#pingRequestText", nil)
		assert.EqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "ping.js loaded from server", result)

		opts := tb.toGojaValue(common.PageReloadOptions{
			WaitUntil: waitUntil,
			Timeout:   30 * time.Second,
		})
		p.Reload(opts)

		result = p.TextContent("#pingRequestText", nil)
		assert.EqualValues(t, "Waiting... pong 20 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "ping.js loaded from server", result)

		return testPromise{}
	}, nil)
}

func TestLifecycleLoadWithSubFrame(t *testing.T) {
	// Test description
	//
	// 1. goto /home (which also has a iframe) and wait for the
	//    load lifecycle event.
	//
	// Success criteria: Once main and subframe (iframe) have both
	//                   loaded the html and the async scripts it
	//                   should unblock. We don't wait for the
	//                   other network requests to complete. We assert
	//                   that the sub frame has amended the main
	//                   frame's DOM.

	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("lifecycle_main_frame.html"), http.StatusMovedPermanently)
	})
	tb.withHandler("/sub", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("lifecycle_subframe.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		time.Sleep(time.Millisecond * 100)

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				var pingJSTextOutput = document.getElementById("pingJSText");
				var parentOutputServerMsg = window.parent.document.getElementById('subFramePingJSText');
				pingJSTextOutput.innerText = "ping.js loaded from server";
				parentOutputServerMsg.innerText = pingJSTextOutput.innerText;
			`)
	})

	assertHome(t, tb, p, common.LifecycleEventLoad, func() testPromise {
		result := p.TextContent("#subFramePingRequestText", nil)
		assert.NotEqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#subFramePingJSText", nil)
		assert.EqualValues(t, "ping.js loaded from server", result)

		return testPromise{}
	}, nil)
}

func TestLifecycleDOMContentLoadedWithSubFrame(t *testing.T) {
	// Test description
	//
	// 1. goto /home (which also has a iframe) and wait for the
	//    domcontentloaded lifecycle event.
	//
	// Success criteria: Once main and subframe (iframe) have both
	//                   loaded the html, it should unblock. We
	//                   don't wait for async scripts and other
	//                   network requests to complete. We assert
	//                   that the sub frame has amended the main
	//                   frame's DOM.

	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("lifecycle_main_frame.html"), http.StatusMovedPermanently)
	})
	tb.withHandler("/sub", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("lifecycle_subframe.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		time.Sleep(time.Millisecond * 100)

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				await new Promise(resolve => setTimeout(resolve, 1000));

				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";

				var parentOutputServerMsg = window.parent.document.getElementById('subFramePingJSText');
				parentOutputServerMsg.innerText = pingJSTextOutput.innerText;
			`)
	})

	assertHome(t, tb, p, common.LifecycleEventDOMContentLoad, func() testPromise {
		result := p.TextContent("#subFramePingRequestText", nil)
		assert.NotEqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#subFramePingJSText", nil)
		assert.EqualValues(t, "Waiting...", result)

		return testPromise{}
	}, nil)
}

func TestLifecycleNetworkIdleWithSubFrame(t *testing.T) {
	// Test description
	//
	// 1. goto /home (which also has a iframe) and wait for the
	//    networkidle lifecycle event.
	//
	// Success criteria: Once main and subframe (iframe) have both
	//                   loaded the html, async scripts and all other
	//                   network requests, it should unblock. We
	//                   assert that the sub frame has amended the
	//                   main frame's DOM.

	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("lifecycle_main_frame.html"), http.StatusMovedPermanently)
	})
	tb.withHandler("/sub", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("lifecycle_subframe.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";

				var parentOutputServerMsg = window.parent.document.getElementById('subFramePingJSText');
				parentOutputServerMsg.innerText = pingJSTextOutput.innerText;
			`)
	})

	assertHome(t, tb, p, common.LifecycleEventNetworkIdle, func() testPromise {
		result := p.TextContent("#subFramePingRequestText", nil)
		assert.EqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#subFramePingJSText", nil)
		assert.EqualValues(t, "ping.js loaded from server", result)

		return testPromise{}
	}, nil)
}

func TestLifecycleLoad(t *testing.T) {
	// Test description
	//
	// 1. goto /home and wait for the load lifecycle event.
	//
	// Success criteria: Once the HTML and the async scripts
	//                   have loaded it should unblock, which
	//                   will be before the other network
	//                   requests complete.

	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("wait_for_nav_lifecycle.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		time.Sleep(time.Millisecond * 100)

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";
			`)
	})

	assertHome(t, tb, p, common.LifecycleEventLoad, func() testPromise {
		result := p.TextContent("#pingRequestText", nil)
		assert.NotEqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "ping.js loaded from server", result)

		return testPromise{}
	}, nil)
}

func TestLifecycleDOMContentLoaded(t *testing.T) {
	// Test description
	//
	// 1. goto /home and wait for the domcontentloaded lifecycle event.
	//
	// Success criteria: Once the HTML is loaded it should unblock, which
	//                   will unblock before the async script and other
	//                   network requests complete.

	t.Parallel()

	tb := newTestBrowser(t, withFileServer())
	p := tb.NewPage(nil)
	tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tb.staticURL("wait_for_nav_lifecycle.html"), http.StatusMovedPermanently)
	})

	var counter int64
	var counterMu sync.Mutex
	tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
		counterMu.Lock()
		defer counterMu.Unlock()

		time.Sleep(time.Millisecond * 100)

		counter++
		fmt.Fprintf(w, "pong %d", counter)
	})

	tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `
				await new Promise(resolve => setTimeout(resolve, 1000));
				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";
			`)
	})

	assertHome(t, tb, p, common.LifecycleEventDOMContentLoad, func() testPromise {
		result := p.TextContent("#pingRequestText", nil)
		assert.NotEqualValues(t, "Waiting... pong 10 - for loop complete", result)

		result = p.TextContent("#pingJSText", nil)
		assert.EqualValues(t, "Waiting...", result)

		return testPromise{}
	}, nil)
}

func TestLifecycleNetworkIdle(t *testing.T) {
	t.Parallel()

	t.Run("doesn't timeout waiting for networkIdle", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withHTTPServer())
		p := tb.NewPage(nil)
		tb.withHandler("/home", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(w, `
			<html>
				<head></head>
				<body>
					<div id="pingJSText">Waiting...</div>
					<script src="/ping.js" async></script>
				</body>
			</html>
			`)
		})

		tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(w, `
				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";
			`)
		})

		assertHome(t, tb, p, common.LifecycleEventNetworkIdle, func() testPromise {
			result := p.TextContent("#pingJSText", nil)
			assert.EqualValues(t, "ping.js loaded from server", result)

			return testPromise{}
		}, nil)
	})

	t.Run("doesn't unblock wait for networkIdle too early", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withFileServer())
		p := tb.NewPage(nil)
		tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, tb.staticURL("prolonged_network_idle.html"), http.StatusMovedPermanently)
		})

		var counter int64
		ch := make(chan bool)
		var counterMu sync.Mutex
		tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
			<-ch

			counterMu.Lock()
			defer counterMu.Unlock()

			time.Sleep(time.Millisecond * 50)

			counter++
			fmt.Fprintf(w, "pong %d", counter)
		})

		tb.withHandler("/ping.js", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(w, `
				var pingJSTextOutput = document.getElementById("pingJSText");
				pingJSTextOutput.innerText = "ping.js loaded from server";
			`)
			close(ch)
		})

		assertHome(t, tb, p, common.LifecycleEventNetworkIdle, func() testPromise {
			result := p.TextContent("#pingRequestText", nil)
			assert.EqualValues(t, "Waiting... pong 4 - for loop complete", result)

			result = p.TextContent("#pingJSText", nil)
			assert.EqualValues(t, "ping.js loaded from server", result)

			return testPromise{}
		}, nil)
	})

	t.Run("doesn't unblock wait on networkIdle early when load and domcontentloaded complete at once", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withFileServer())
		p := tb.NewPage(nil)
		tb.withHandler("/home", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, tb.staticURL("prolonged_network_idle_10.html"), http.StatusMovedPermanently)
		})

		var counterMu sync.Mutex
		var counter int64
		tb.withHandler("/ping", func(w http.ResponseWriter, _ *http.Request) {
			counterMu.Lock()
			defer counterMu.Unlock()

			time.Sleep(time.Millisecond * 50)

			counter++
			fmt.Fprintf(w, "pong %d", counter)
		})

		assertHome(t, tb, p, common.LifecycleEventNetworkIdle, func() testPromise {
			result := p.TextContent("#pingRequestText", nil)
			assert.EqualValues(t, "Waiting... pong 10 - for loop complete", result)

			return testPromise{}
		}, nil)
	})
}

func assertHome(
	t *testing.T,
	tb *testBrowser,
	p api.Page,
	waitUntil common.LifecycleEvent,
	check func() testPromise,
	secondCheck func(),
) {
	t.Helper()

	var resolved, rejected bool
	err := tb.await(func() error {
		opts := tb.toGojaValue(common.FrameGotoOptions{
			WaitUntil: waitUntil,
			Timeout:   30 * time.Second,
		})
		prm := tb.promise(p.Goto(tb.URL("/home"), opts)).then(
			func() testPromise {
				resolved = true
				return check()
			},
			func() {
				rejected = true
			},
		)
		if secondCheck != nil {
			prm.then(func() {
				secondCheck()
			})
		}

		return nil
	})
	require.NoError(t, err)

	assert.True(t, resolved)
	assert.False(t, rejected)
}
