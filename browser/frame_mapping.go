package browser

import (
	"fmt"

	"github.com/dop251/goja"

	"github.com/grafana/xk6-browser/common"
	"github.com/grafana/xk6-browser/k6ext"
)

// mapFrame to the JS module.
//
//nolint:funlen
func mapFrame(vu moduleVU, f *common.Frame) mapping { //nolint:gocognit,cyclop
	rt := vu.Runtime()
	maps := mapping{
		"check": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, f.Check(selector, opts) //nolint:wrapcheck
			})
		},
		"childFrames": func() *goja.Object {
			var (
				mcfs []mapping
				cfs  = f.ChildFrames()
			)
			for _, fr := range cfs {
				mcfs = append(mcfs, mapFrame(vu, fr))
			}
			return rt.ToValue(mcfs).ToObject(rt)
		},
		"click": func(selector string, opts goja.Value) (*goja.Promise, error) {
			popts, err := parseFrameClickOptions(vu.Context(), opts, f.Timeout())
			if err != nil {
				return nil, err
			}

			return k6ext.Promise(vu.Context(), func() (any, error) {
				err := f.Click(selector, popts)
				return nil, err //nolint:wrapcheck
			}), nil
		},
		"content": func() *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.Content() //nolint:wrapcheck
			})
		},
		"dblclick": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, f.Dblclick(selector, opts) //nolint:wrapcheck
			})
		},
		"dispatchEvent": func(selector, typ string, eventInit, opts goja.Value) (*goja.Promise, error) {
			popts := common.NewFrameDispatchEventOptions(f.Timeout())
			if err := popts.Parse(vu.Context(), opts); err != nil {
				return nil, fmt.Errorf("parsing frame dispatch event options: %w", err)
			}
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, f.DispatchEvent(selector, typ, exportArg(eventInit), popts) //nolint:wrapcheck
			}), nil
		},
		"evaluate": func(pageFunction goja.Value, gargs ...goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.Evaluate(pageFunction.String(), exportArgs(gargs)...) //nolint:wrapcheck
			})
		},
		"evaluateHandle": func(pageFunction goja.Value, gargs ...goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				jsh, err := f.EvaluateHandle(pageFunction.String(), exportArgs(gargs)...)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}
				return mapJSHandle(vu, jsh), nil
			})
		},
		"fill": func(selector, value string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, f.Fill(selector, value, opts) //nolint:wrapcheck
			})
		},
		"focus": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, f.Focus(selector, opts) //nolint:wrapcheck
			})
		},
		"frameElement": func() (mapping, error) {
			fe, err := f.FrameElement()
			if err != nil {
				return nil, err //nolint:wrapcheck
			}
			return mapElementHandle(vu, fe), nil
		},
		"getAttribute": func(selector, name string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.GetAttribute(selector, name, opts) //nolint:wrapcheck
			})
		},
		"goto": func(url string, opts goja.Value) (*goja.Promise, error) {
			gopts := common.NewFrameGotoOptions(
				f.Referrer(),
				f.NavigationTimeout(),
			)
			if err := gopts.Parse(vu.Context(), opts); err != nil {
				return nil, fmt.Errorf("parsing frame navigation options to %q: %w", url, err)
			}
			return k6ext.Promise(vu.Context(), func() (any, error) {
				resp, err := f.Goto(url, gopts)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}

				return mapResponse(vu, resp), nil
			}), nil
		},
		"hover": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, f.Hover(selector, opts) //nolint:wrapcheck
			})
		},
		"innerHTML": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.InnerHTML(selector, opts) //nolint:wrapcheck
			})
		},
		"innerText": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.InnerText(selector, opts) //nolint:wrapcheck
			})
		},
		"inputValue": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.InputValue(selector, opts) //nolint:wrapcheck
			})
		},
		"isChecked": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.IsChecked(selector, opts) //nolint:wrapcheck
			})
		},
		"isDetached": f.IsDetached,
		"isDisabled": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.IsDisabled(selector, opts) //nolint:wrapcheck
			})
		},
		"isEditable": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.IsEditable(selector, opts) //nolint:wrapcheck
			})
		},
		"isEnabled": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.IsEnabled(selector, opts) //nolint:wrapcheck
			})
		},
		"isHidden": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.IsHidden(selector, opts) //nolint:wrapcheck
			})
		},
		"isVisible": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.IsVisible(selector, opts) //nolint:wrapcheck
			})
		},
		"locator": func(selector string, opts goja.Value) *goja.Object {
			ml := mapLocator(vu, f.Locator(selector, opts))
			return rt.ToValue(ml).ToObject(rt)
		},
		"name": f.Name,
		"page": func() *goja.Object {
			mp := mapPage(vu, f.Page())
			return rt.ToValue(mp).ToObject(rt)
		},
		"parentFrame": func() *goja.Object {
			mf := mapFrame(vu, f.ParentFrame())
			return rt.ToValue(mf).ToObject(rt)
		},
		"press": func(selector, key string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, f.Press(selector, key, opts) //nolint:wrapcheck
			})
		},
		"selectOption": func(selector string, values goja.Value, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.SelectOption(selector, values, opts) //nolint:wrapcheck
			})
		},
		"setContent": func(html string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, f.SetContent(html, opts) //nolint:wrapcheck
			})
		},
		"setInputFiles": func(selector string, files goja.Value, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, f.SetInputFiles(selector, files, opts) //nolint:wrapcheck
			})
		},
		"tap": func(selector string, opts goja.Value) (*goja.Promise, error) {
			popts := common.NewFrameTapOptions(f.Timeout())
			if err := popts.Parse(vu.Context(), opts); err != nil {
				return nil, fmt.Errorf("parsing frame tap options: %w", err)
			}
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return nil, f.Tap(selector, popts) //nolint:wrapcheck
			}), nil
		},
		"textContent": func(selector string, opts goja.Value) *goja.Promise {
			return k6ext.Promise(vu.Context(), func() (any, error) {
				return f.TextContent(selector, opts) //nolint:wrapcheck
			})
		},
		"title":   f.Title,
		"type":    f.Type,
		"uncheck": f.Uncheck,
		"url":     f.URL,
		"waitForFunction": func(pageFunc, opts goja.Value, args ...goja.Value) (*goja.Promise, error) {
			js, popts, pargs, err := parseWaitForFunctionArgs(
				vu.Context(), f.Timeout(), pageFunc, opts, args...,
			)
			if err != nil {
				return nil, fmt.Errorf("frame waitForFunction: %w", err)
			}

			return k6ext.Promise(vu.Context(), func() (result any, reason error) {
				return f.WaitForFunction(js, popts, pargs...) //nolint:wrapcheck
			}), nil
		},
		"waitForLoadState": f.WaitForLoadState,
		"waitForNavigation": func(opts goja.Value) (*goja.Promise, error) {
			popts := common.NewFrameWaitForNavigationOptions(f.Timeout())
			if err := popts.Parse(vu.Context(), opts); err != nil {
				return nil, fmt.Errorf("parsing frame wait for navigation options: %w", err)
			}

			return k6ext.Promise(vu.Context(), func() (result any, reason error) {
				resp, err := f.WaitForNavigation(popts)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}
				return mapResponse(vu, resp), nil
			}), nil
		},
		"waitForSelector": func(selector string, opts goja.Value) (mapping, error) {
			eh, err := f.WaitForSelector(selector, opts)
			if err != nil {
				return nil, err //nolint:wrapcheck
			}
			return mapElementHandle(vu, eh), nil
		},
		"waitForTimeout": f.WaitForTimeout,
	}
	maps["$"] = func(selector string) (mapping, error) {
		eh, err := f.Query(selector, common.StrictModeOff)
		if err != nil {
			return nil, err //nolint:wrapcheck
		}
		// ElementHandle can be null when the selector does not match any elements.
		// We do not want to map nil elementHandles since the expectation is a
		// null result in the test script for this case.
		if eh == nil {
			return nil, nil //nolint:nilnil
		}
		ehm := mapElementHandle(vu, eh)

		return ehm, nil
	}
	maps["$$"] = func(selector string) ([]mapping, error) {
		ehs, err := f.QueryAll(selector)
		if err != nil {
			return nil, err //nolint:wrapcheck
		}
		var mehs []mapping
		for _, eh := range ehs {
			ehm := mapElementHandle(vu, eh)
			mehs = append(mehs, ehm)
		}
		return mehs, nil
	}

	return maps
}
