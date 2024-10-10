package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/sobek"

	"github.com/grafana/xk6-browser/common"
	"github.com/grafana/xk6-browser/k6ext"
)

// mapPage to the JS module.
//
//nolint:funlen
func mapPage(vu moduleVU, p *common.Page) mapping { //nolint:gocognit,cyclop
	rt := vu.Runtime()
	maps := mapping{
		"bringToFront": func() *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.BringToFront() //nolint:wrapcheck
			})
		},
		"check": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.Check(selector, opts) //nolint:wrapcheck
			})
		},
		"click": func(selector string, opts sobek.Value) (*sobek.Promise, error) {
			popts, err := parseFrameClickOptions(vu.Context(), opts, p.Timeout())
			if err != nil {
				return nil, err
			}

			return k6ext.Promise(vu.Context(), func() (any, error) {
				err := p.Click(selector, popts)
				return nil, err //nolint:wrapcheck
			}), nil
		},
		"close": func(opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				// It's safe to close the taskqueue for this targetID (if one
				// exists).
				vu.taskQueueRegistry.close(p.TargetID())

				return nil, p.Close(opts) //nolint:wrapcheck
			})
		},
		"content": func() *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.Content() //nolint:wrapcheck
			})
		},
		"context": func() mapping {
			return mapBrowserContext(vu, p.Context())
		},
		"dblclick": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.Dblclick(selector, opts) //nolint:wrapcheck
			})
		},
		"dispatchEvent": func(selector, typ string, eventInit, opts sobek.Value) (*sobek.Promise, error) {
			popts := common.NewFrameDispatchEventOptions(p.Timeout())
			if err := popts.Parse(vu.Context(), opts); err != nil {
				return nil, fmt.Errorf("parsing page dispatch event options: %w", err)
			}
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.DispatchEvent(selector, typ, exportArg(eventInit), popts) //nolint:wrapcheck
			}), nil
		},
		"emulateMedia": func(opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.EmulateMedia(opts) //nolint:wrapcheck
			})
		},
		"emulateVisionDeficiency": func(typ string) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.EmulateVisionDeficiency(typ) //nolint:wrapcheck
			})
		},
		"evaluate": func(pageFunction sobek.Value, gargs ...sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.Evaluate(pageFunction.String(), exportArgs(gargs)...) //nolint:wrapcheck
			})
		},
		"evaluateHandle": func(pageFunc sobek.Value, gargs ...sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				jsh, err := p.EvaluateHandle(pageFunc.String(), exportArgs(gargs)...)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}
				return mapJSHandle(vu, jsh), nil
			})
		},
		"fill": func(selector string, value string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.Fill(selector, value, opts) //nolint:wrapcheck
			})
		},
		"focus": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.Focus(selector, opts) //nolint:wrapcheck
			})
		},
		"frames": func() *sobek.Object {
			var (
				mfrs []mapping
				frs  = p.Frames()
			)
			for _, fr := range frs {
				mfrs = append(mfrs, mapFrame(vu, fr))
			}
			return rt.ToValue(mfrs).ToObject(rt)
		},
		"getAttribute": func(selector string, name string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				s, ok, err := p.GetAttribute(selector, name, opts)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}
				if !ok {
					return nil, nil
				}
				return s, nil
			})
		},
		"goto": func(url string, opts sobek.Value) (*sobek.Promise, error) {
			gopts := common.NewFrameGotoOptions(
				p.Referrer(),
				p.NavigationTimeout(),
			)
			if err := gopts.Parse(vu.Context(), opts); err != nil {
				return nil, fmt.Errorf("parsing page navigation options to %q: %w", url, err)
			}
			return k6ext.Promise(vu.Context(), func() (any, error) {
				resp, err := p.Goto(url, gopts)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}

				return mapResponse(vu, resp), nil
			}), nil
		},
		"hover": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.Hover(selector, opts) //nolint:wrapcheck
			})
		},
		"innerHTML": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.InnerHTML(selector, opts) //nolint:wrapcheck
			})
		},
		"innerText": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.InnerText(selector, opts) //nolint:wrapcheck
			})
		},
		"inputValue": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.InputValue(selector, opts) //nolint:wrapcheck
			})
		},
		"isChecked": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.IsChecked(selector, opts) //nolint:wrapcheck
			})
		},
		"isClosed": p.IsClosed,
		"isDisabled": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.IsDisabled(selector, opts) //nolint:wrapcheck
			})
		},
		"isEditable": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.IsEditable(selector, opts) //nolint:wrapcheck
			})
		},
		"isEnabled": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.IsEnabled(selector, opts) //nolint:wrapcheck
			})
		},
		"isHidden": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.IsHidden(selector, opts) //nolint:wrapcheck
			})
		},
		"isVisible": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.IsVisible(selector, opts) //nolint:wrapcheck
			})
		},
		"keyboard": mapKeyboard(vu, p.GetKeyboard()),
		"locator": func(selector string, opts sobek.Value) *sobek.Object {
			ml := mapLocator(vu, p.Locator(selector, opts))
			return rt.ToValue(ml).ToObject(rt)
		},
		"mainFrame": func() *sobek.Object {
			mf := mapFrame(vu, p.MainFrame())
			return rt.ToValue(mf).ToObject(rt)
		},
		"mouse": mapMouse(vu, p.GetMouse()),
		"on":    mapPageOn(vu, p),
		"opener": func() *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.Opener(), nil
			})
		},
		"press": func(selector string, key string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.Press(selector, key, opts) //nolint:wrapcheck
			})
		},
		"reload": func(opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				resp, err := p.Reload(opts)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}

				if resp == nil {
					return nil, nil
				}

				r := mapResponse(vu, resp)

				return rt.ToValue(r).ToObject(rt), nil
			})
		},
		"screenshot": func(opts sobek.Value) (*sobek.Promise, error) {
			popts := common.NewPageScreenshotOptions()
			if err := popts.Parse(vu.Context(), opts); err != nil {
				return nil, fmt.Errorf("parsing page screenshot options: %w", err)
			}

			return k6ext.Promise(vu.Context(), func() (any, error) {
				bb, err := p.Screenshot(popts, vu.filePersister)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}

				ab := rt.NewArrayBuffer(bb)

				return &ab, nil
			}), nil
		},
		"selectOption": func(selector string, values sobek.Value, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.SelectOption(selector, values, opts) //nolint:wrapcheck
			})
		},
		"setChecked": func(selector string, checked bool, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.SetChecked(selector, checked, opts) //nolint:wrapcheck
			})
		},
		"setContent": func(html string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.SetContent(html, opts) //nolint:wrapcheck
			})
		},
		"setDefaultNavigationTimeout": p.SetDefaultNavigationTimeout,
		"setDefaultTimeout":           p.SetDefaultTimeout,
		"setExtraHTTPHeaders": func(headers map[string]string) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.SetExtraHTTPHeaders(headers) //nolint:wrapcheck
			})
		},
		"setInputFiles": func(selector string, files sobek.Value, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.SetInputFiles(selector, files, opts) //nolint:wrapcheck
			})
		},
		"setViewportSize": func(viewportSize sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.SetViewportSize(viewportSize) //nolint:wrapcheck
			})
		},
		"tap": func(selector string, opts sobek.Value) (*sobek.Promise, error) {
			popts := common.NewFrameTapOptions(p.Timeout())
			if err := popts.Parse(vu.Context(), opts); err != nil {
				return nil, fmt.Errorf("parsing page tap options: %w", err)
			}
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.Tap(selector, popts) //nolint:wrapcheck
			}), nil
		},
		"textContent": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				s, ok, err := p.TextContent(selector, opts)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}
				if !ok {
					return nil, nil
				}
				return s, nil
			})
		},
		"throttleCPU": func(cpuProfile common.CPUProfile) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.ThrottleCPU(cpuProfile) //nolint:wrapcheck
			})
		},
		"throttleNetwork": func(networkProfile common.NetworkProfile) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.ThrottleNetwork(networkProfile) //nolint:wrapcheck
			})
		},
		"title": func() *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return p.Title() //nolint:wrapcheck
			})
		},
		"touchscreen": mapTouchscreen(vu, p.GetTouchscreen()),
		"type": func(selector string, text string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.Type(selector, text, opts) //nolint:wrapcheck
			})
		},
		"uncheck": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.Uncheck(selector, opts) //nolint:wrapcheck
			})
		},
		"url":          p.URL,
		"viewportSize": p.ViewportSize,
		"waitForFunction": func(pageFunc, opts sobek.Value, args ...sobek.Value) (*sobek.Promise, error) {
			js, popts, pargs, err := parseWaitForFunctionArgs(
				vu.Context(), p.Timeout(), pageFunc, opts, args...,
			)
			if err != nil {
				return nil, fmt.Errorf("page waitForFunction: %w", err)
			}

			return k6ext.Promise(vu.Context(), func() (result any, reason error) {
				return p.WaitForFunction(js, popts, pargs...) //nolint:wrapcheck
			}), nil
		},
		"waitForLoadState": func(state string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, p.WaitForLoadState(state, opts) //nolint:wrapcheck
			})
		},
		"waitForNavigation": func(opts sobek.Value) (*sobek.Promise, error) {
			popts := common.NewFrameWaitForNavigationOptions(p.Timeout())
			if err := popts.Parse(vu.Context(), opts); err != nil {
				return nil, fmt.Errorf("parsing page wait for navigation options: %w", err)
			}

			return k6ext.Promise(vu.Context(), func() (result any, reason error) {
				resp, err := p.WaitForNavigation(popts)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}
				return mapResponse(vu, resp), nil
			}), nil
		},
		"waitForSelector": func(selector string, opts sobek.Value) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				eh, err := p.WaitForSelector(selector, opts)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}
				return mapElementHandle(vu, eh), nil
			})
		},
		"waitForTimeout": func(timeout int64) *sobek.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				p.WaitForTimeout(timeout)
				return nil, nil
			})
		},
		"workers": func() *sobek.Object {
			var mws []mapping
			for _, w := range p.Workers() {
				mw := mapWorker(vu, w)
				mws = append(mws, mw)
			}
			return rt.ToValue(mws).ToObject(rt)
		},
	}
	maps["$"] = func(selector string) *sobek.Promise {
		return k6ext.Promise(vu.Context(), func() (any, error) {
			eh, err := p.Query(selector)
			if err != nil {
				return nil, err //nolint:wrapcheck
			}
			// ElementHandle can be null when the selector does not match any elements.
			// We do not want to map nil elementHandles since the expectation is a
			// null result in the test script for this case.
			if eh == nil {
				return nil, nil
			}
			ehm := mapElementHandle(vu, eh)

			return ehm, nil
		})
	}
	maps["$$"] = func(selector string) *sobek.Promise {
		return k6ext.Promise(vu.Context(), func() (any, error) {
			ehs, err := p.QueryAll(selector)
			if err != nil {
				return nil, err //nolint:wrapcheck
			}
			var mehs []mapping
			for _, eh := range ehs {
				ehm := mapElementHandle(vu, eh)
				mehs = append(mehs, ehm)
			}
			return mehs, nil
		})
	}

	return maps
}

// mapPageOn maps the requested page.on event to the Sobek runtime.
// It generalizes the handling of page.on events on a taskqueue.
func mapPageOn(vu moduleVU, p *common.Page) func(common.PageOnEventName, sobek.Callable) error { //nolint:funlen
	rt := vu.Runtime()

	return func(eventName common.PageOnEventName, handler sobek.Callable) error {
		tq := vu.taskQueueRegistry.get(vu.Context(), p.TargetID())

		onEventPageConsoleAPICalled := func(event common.PageOnEvent) {
			tq.Queue(func() error {
				mapping := mapConsoleMessage(vu, event)
				_, err := handler(sobek.Undefined(), rt.ToValue(mapping))
				if err != nil {
					return fmt.Errorf("executing page.on handler: %w", err)
				}
				return nil
			})
		}

		onEventPageMetricCalled := func(event common.PageOnEvent) {
			// The function on the taskqueue runs in its own goroutine
			// so we need to use a channel to wait for it to complete
			// since we're waiting for updates from the handler which
			// will be written to the ExportedMetric.
			done := make(chan struct{})
			tq.Queue(func() error {
				defer close(done)

				mapping := mapMetricEvent(vu, event)
				_, err := handler(sobek.Undefined(), rt.ToValue(mapping))
				if err != nil {
					return fmt.Errorf("executing page.on('metric') handler: %w", err)
				}

				return nil
			})
			<-done
		}

		var mapHandler func(common.PageOnEvent)
		switch eventName {
		case common.EventPageConsoleAPICalled:
			mapHandler = onEventPageConsoleAPICalled
		case common.EventPageMetricCalled:
			mapHandler = onEventPageMetricCalled
		default:
			return fmt.Errorf("unknown page event: %q", eventName)
		}

		if eventName == common.EventPageMetricCalled {
			// Register a custom regex function for the metric event
			// that will be used to check URLs against the patterns.
			// This is needed because we want to use the JavaScript regex
			// to comply with what users expect when using the `tag` method.
			_, err := rt.RunString(`
				function _k6BrowserCheckRegEx(pattern, url) {
					let r = pattern;
					if (typeof pattern === 'string') {
						r = new RegExp(pattern);
					}
					return r.test(url);
				}
			`)
			if err != nil {
				return fmt.Errorf("evaluating regex function: %w", err)
			}
		}

		return p.On(eventName, mapHandler) //nolint:wrapcheck
	}
}

func parseWaitForFunctionArgs(
	ctx context.Context, timeout time.Duration, pageFunc, opts sobek.Value, gargs ...sobek.Value,
) (string, *common.FrameWaitForFunctionOptions, []any, error) {
	popts := common.NewFrameWaitForFunctionOptions(timeout)
	err := popts.Parse(ctx, opts)
	if err != nil {
		return "", nil, nil, fmt.Errorf("parsing waitForFunction options: %w", err)
	}

	js := pageFunc.ToString().String()
	_, isCallable := sobek.AssertFunction(pageFunc)
	if !isCallable {
		js = fmt.Sprintf("() => (%s)", js)
	}

	return js, popts, exportArgs(gargs), nil
}
