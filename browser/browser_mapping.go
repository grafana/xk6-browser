package browser

import (
	"fmt"
	"path/filepath"

	"github.com/grafana/sobek"

	"github.com/grafana/xk6-browser/common"
	"github.com/grafana/xk6-browser/k6ext"
)

// mapBrowser to the JS module.
func mapBrowser(vu moduleVU) mapping { //nolint:funlen,cyclop,gocognit
	return mapping{
		"context": func() (mapping, error) {
			pauseOnBreakpoint(vu.breakpointRegistry, vu.Runtime())

			b, err := vu.browser()
			if err != nil {
				return nil, err
			}
			return mapBrowserContext(vu, b.Context()), nil
		},
		"closeContext": func() *sobek.Promise {
			pauseOnBreakpoint(vu.breakpointRegistry, vu.Runtime())

			return k6ext.Promise(vu.Context(), func() (any, error) {
				b, err := vu.browser()
				if err != nil {
					return nil, err
				}
				return nil, b.CloseContext() //nolint:wrapcheck
			})
		},
		"isConnected": func() (bool, error) {
			pauseOnBreakpoint(vu.breakpointRegistry, vu.Runtime())

			b, err := vu.browser()
			if err != nil {
				return false, err
			}
			return b.IsConnected(), nil
		},
		"newContext": func(opts sobek.Value) (*sobek.Promise, error) {
			pauseOnBreakpoint(vu.breakpointRegistry, vu.Runtime())

			popts, err := parseBrowserContextOptions(vu.Runtime(), opts)
			if err != nil {
				return nil, fmt.Errorf("parsing browser.newContext options: %w", err)
			}
			return k6ext.Promise(vu.Context(), func() (any, error) {
				b, err := vu.browser()
				if err != nil {
					return nil, err
				}
				bctx, err := b.NewContext(popts)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}
				if err := initBrowserContext(bctx, vu.testRunID); err != nil {
					return nil, err
				}

				return mapBrowserContext(vu, bctx), nil
			}), nil
		},
		"userAgent": func() (string, error) {
			pauseOnBreakpoint(vu.breakpointRegistry, vu.Runtime())

			b, err := vu.browser()
			if err != nil {
				return "", err
			}
			return b.UserAgent(), nil
		},
		"version": func() (string, error) {
			pauseOnBreakpoint(vu.breakpointRegistry, vu.Runtime())

			b, err := vu.browser()
			if err != nil {
				return "", err
			}
			return b.Version(), nil
		},
		"newPage": func(opts sobek.Value) (*sobek.Promise, error) {
			pauseOnBreakpoint(vu.breakpointRegistry, vu.Runtime())

			pos := getCurrentLineNumber(vu.Runtime())
			fileNameWithExt := filepath.Base(pos.Filename)
			fileExt := filepath.Ext(pos.Filename)
			fileNameWithoutExt := fileNameWithExt[:len(fileNameWithExt)-len(fileExt)]

			popts, err := parseBrowserContextOptions(vu.Runtime(), opts)
			if err != nil {
				return nil, fmt.Errorf("parsing browser.newPage options: %w", err)
			}
			return k6ext.Promise(vu.Context(), func() (any, error) {
				b, err := vu.browser()
				if err != nil {
					return nil, err
				}
				page, err := b.NewPage(popts)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}
				if err := initBrowserContext(b.Context(), vu.testRunID); err != nil {
					return nil, err
				}

				// currently the variable won't be garbage collected. perfect...
				pageVar := func() (any, error) {
					uri, err := page.URL()
					if err != nil {
						return nil, fmt.Errorf("getting page URL: %w", err)
					}
					return struct {
						URL string `json:"url"`
					}{
						URL: uri,
					}, nil
				}
				if err := vu.breakpointRegistry.setVar("page", pageVar); err != nil {
					return nil, err
				}

				tq := vu.taskQueueRegistry.get(vu.Context(), page.TargetID())
				page.SetScreenshotPersister(vu.filePersister)
				page.SetScriptName(fileNameWithoutExt)
				page.SetTaskQueue(tq)
				return mapPage(vu, page), nil
			}), nil
		},
	}
}

func initBrowserContext(bctx *common.BrowserContext, testRunID string) error {
	// Setting a k6 object which will contain k6 specific metadata
	// on the current test run. This allows external applications
	// (such as Grafana Faro) to identify that the session is a k6
	// automated one and not one driven by a real person.
	if err := bctx.AddInitScript(
		fmt.Sprintf(`window.k6 = { testRunId: %q }`, testRunID),
	); err != nil {
		return fmt.Errorf("adding k6 object to new browser context: %w", err)
	}

	return nil
}

// parseBrowserContextOptions parses the [common.BrowserContext] options from a Sobek value.
func parseBrowserContextOptions(rt *sobek.Runtime, opts sobek.Value) (*common.BrowserContextOptions, error) {
	b := common.DefaultBrowserContextOptions()
	if err := mergeWith(rt, b, opts); err != nil {
		return nil, err
	}
	return b, nil
}
