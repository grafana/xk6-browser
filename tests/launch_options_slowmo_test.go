package tests

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/xk6-browser/browser"
)

func TestBrowserOptionsSlowMo(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	t.Run("Page", func(t *testing.T) {
		t.Parallel()
		t.Run("check", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.Check(".check", nil)
			})
		})
		t.Run("click", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				err := p.Click("button", nil)
				assert.NoError(t, err)
			})
		})
		t.Run("dblClick", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.Dblclick("button", nil)
			})
		})
		t.Run("dispatchEvent", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.DispatchEvent("button", "click", goja.Null(), nil)
			})
		})
		t.Run("emulateMedia", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.EmulateMedia(tb.toGojaValue(struct {
					Media string `js:"media"`
				}{
					Media: "print",
				}))
			})
		})
		t.Run("evaluate", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.Evaluate(tb.toGojaValue("() => void 0"))
			})
		})
		t.Run("evaluateHandle", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.EvaluateHandle(tb.toGojaValue("() => window"))
			})
		})
		t.Run("fill", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.Fill(".fill", "foo", nil)
			})
		})
		t.Run("focus", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.Focus("button", nil)
			})
		})
		t.Run("goto", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				_, err := p.Goto(browser.BlankPage, nil)
				require.NoError(t, err)
			})
		})
		t.Run("hover", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.Hover("button", nil)
			})
		})
		t.Run("press", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.Press("button", "Enter", nil)
			})
		})
		t.Run("reload", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.Reload(nil)
			})
		})
		t.Run("setContent", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.SetContent("hello world", nil)
			})
		})
		/*t.Run("setInputFiles", func(t *testing.T) {
			testPageSlowMoImpl(t, tb, func(_ *Browser, p *common.Page) {
				p.SetInputFiles(".file", nil, nil)
			})
		})*/
		t.Run("selectOption", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.SelectOption("select", tb.toGojaValue("foo"), nil)
			})
		})
		t.Run("setViewportSize", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.SetViewportSize(nil)
			})
		})
		t.Run("type", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.Type(".fill", "a", nil)
			})
		})
		t.Run("uncheck", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testPageSlowMoImpl(t, tb, func(_ *testBrowser, p *browser.Page) {
				p.Uncheck(".uncheck", nil)
			})
		})
	})

	t.Run("Frame", func(t *testing.T) {
		t.Parallel()
		t.Run("check", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.Check(".check", nil)
			})
		})
		t.Run("click", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				err := f.Click("button", nil)
				assert.NoError(t, err)
			})
		})
		t.Run("dblClick", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.Dblclick("button", nil)
			})
		})
		t.Run("dispatchEvent", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.DispatchEvent("button", "click", goja.Null(), nil)
			})
		})
		t.Run("evaluate", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.Evaluate(tb.toGojaValue("() => void 0"))
			})
		})
		t.Run("evaluateHandle", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.EvaluateHandle(tb.toGojaValue("() => window"))
			})
		})
		t.Run("fill", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.Fill(".fill", "foo", nil)
			})
		})
		t.Run("focus", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.Focus("button", nil)
			})
		})
		t.Run("goto", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				_, _ = f.Goto(browser.BlankPage, nil)
			})
		})
		t.Run("hover", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.Hover("button", nil)
			})
		})
		t.Run("press", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.Press("button", "Enter", nil)
			})
		})
		t.Run("setContent", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.SetContent("hello world", nil)
			})
		})
		/*t.Run("setInputFiles", func(t *testing.T) {
			testFrameSlowMoImpl(t, tb, func(_ *Browser, f common.Frame) {
				f.SetInputFiles(".file", nil, nil)
			})
		})*/
		t.Run("selectOption", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.SelectOption("select", tb.toGojaValue("foo"), nil)
			})
		})
		t.Run("type", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.Type(".fill", "a", nil)
			})
		})
		t.Run("uncheck", func(t *testing.T) {
			t.Parallel()
			tb := newTestBrowser(t, withFileServer())
			testFrameSlowMoImpl(t, tb, func(_ *testBrowser, f *browser.Frame) {
				f.Uncheck(".uncheck", nil)
			})
		})
	})

	// TODO implement this
	t.Run("ElementHandle", func(t *testing.T) {
	})
}

func testSlowMoImpl(t *testing.T, tb *testBrowser, fn func(*testBrowser)) {
	t.Helper()

	hooks := browser.GetHooks(tb.ctx)
	currentHook := hooks.Get(browser.HookApplySlowMo)
	chCalled := make(chan bool, 1)
	defer hooks.Register(browser.HookApplySlowMo, currentHook)
	hooks.Register(browser.HookApplySlowMo, func(ctx context.Context) {
		currentHook(ctx)
		chCalled <- true
	})

	didSlowMo := false
	go fn(tb)
	select {
	case <-tb.ctx.Done():
	case <-chCalled:
		didSlowMo = true
	}

	require.True(t, didSlowMo, "expected action to have been slowed down")
}

func testPageSlowMoImpl(t *testing.T, tb *testBrowser, fn func(*testBrowser, *browser.Page)) {
	t.Helper()

	p := tb.NewPage(nil)
	p.SetContent(`
		<button>a</button>
		<input type="checkbox" class="check">
		<input type="checkbox" checked=true class="uncheck">
		<input class="fill">
		<select>
		<option>foo</option>
		</select>
		<input type="file" class="file">
    	`, nil)
	testSlowMoImpl(t, tb, func(tb *testBrowser) { fn(tb, p) })
}

func testFrameSlowMoImpl(t *testing.T, tb *testBrowser, fn func(bt *testBrowser, f *browser.Frame)) {
	t.Helper()

	p := tb.NewPage(nil)

	pageFn := `
	async (frameId, url) => {
		const frame = document.createElement('iframe');
		frame.src = url;
		frame.id = frameId;
		document.body.appendChild(frame);
		await new Promise(x => frame.onload = x);
		return frame;
	}
	`

	h, err := p.EvaluateHandle(
		tb.toGojaValue(pageFn),
		tb.toGojaValue("frame1"),
		tb.toGojaValue(tb.staticURL("empty.html")))
	require.NoError(tb.t, err)

	f, err := h.AsElement().ContentFrame()
	require.NoError(tb.t, err)

	f.SetContent(`
		<button>a</button>
		<input type="checkbox" class="check">
		<input type="checkbox" checked=true class="uncheck">
		<input class="fill">
		<select>
		  <option>foo</option>
		</select>
		<input type="file" class="file">
    	`, nil)
	testSlowMoImpl(t, tb, func(tb *testBrowser) { fn(tb, f) })
}
