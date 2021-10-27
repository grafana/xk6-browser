/*
 *
 * xk6-browser - a browser automation extension for k6
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package chromium

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/grafana/xk6-browser/api"
	"github.com/grafana/xk6-browser/common"
	"github.com/pkg/errors"
	k6common "go.k6.io/k6/js/common"
	k6lib "go.k6.io/k6/lib"
)

// Ensure BrowserType implements the api.BrowserType interface.
var _ api.BrowserType = &BrowserType{}

type BrowserType struct {
	Ctx             context.Context
	CancelFn        context.CancelFunc
	hooks           *common.Hooks
	fieldNameMapper *common.FieldNameMapper
}

func NewBrowserType(ctx context.Context) api.BrowserType {
	rt := k6common.GetRuntime(ctx)
	state := k6lib.GetState(ctx)
	hooks := common.NewHooks()

	// Create extension master context. If this context is cancelled we'll
	// initiate an extension wide cancellation and shutdown.
	extensionCtx := context.Background()
	extensionCtx, extensionCancelFn := context.WithCancel(extensionCtx)
	extensionCtx = k6common.WithRuntime(extensionCtx, rt)
	extensionCtx = k6lib.WithState(extensionCtx, state)
	extensionCtx = common.WithHooks(extensionCtx, hooks)

	b := BrowserType{
		Ctx:             extensionCtx,
		CancelFn:        extensionCancelFn,
		hooks:           hooks,
		fieldNameMapper: common.NewFieldNameMapper(),
	}
	rt.SetFieldNameMapper(b.fieldNameMapper)
	return &b
}

func (b *BrowserType) Connect(opts goja.Value) {
	rt := k6common.GetRuntime(b.Ctx)
	k6common.Throw(rt, errors.Errorf("BrowserType.connect() has not been implemented yet!"))
}

func (b *BrowserType) ExecutablePath() string {
	return "chromium"
}

// Launch creates a new client to remote control a Chrome browser.
func (b *BrowserType) Launch(opts goja.Value) api.Browser {
	rt := k6common.GetRuntime(b.Ctx)

	launchOpts := common.NewLaunchOptions()
	launchOpts.Parse(b.Ctx, opts)

	b.Ctx = common.WithLaunchOptions(b.Ctx, launchOpts)

	flags := map[string]interface{}{
		//chromedp.ProxyServer(""),
		//chromedp.UserAgent(""),
		//chromedp.UserDataDir(""),
		//chromedp.DisableGPU,

		"no-first-run":                true,
		"no-default-browser-check":    true,
		"no-sandbox":                  true,
		"headless":                    launchOpts.Headless,
		"auto-open-devtools-for-tabs": launchOpts.Devtools,
		"window-size":                 fmt.Sprintf("%d,%d", 800, 600),

		// After Puppeteer's and Playwright's default behavior.
		"disable-background-networking":                      true,
		"enable-features":                                    "NetworkService,NetworkServiceInProcess",
		"disable-background-timer-throttling":                true,
		"disable-backgrounding-occluded-windows":             true,
		"disable-breakpad":                                   true,
		"disable-client-side-phishing-detection":             true,
		"disable-component-extensions-with-background-pages": true,
		"disable-default-apps":                               true,
		"disable-dev-shm-usage":                              true,
		"disable-extensions":                                 true,
		"disable-features":                                   "TranslateUI,BlinkGenPropertyTrees,ImprovedCookieControls,SameSiteByDefaultCookies,LazyFrameLoading",
		"disable-hang-monitor":                               true,
		"disable-ipc-flooding-protection":                    true,
		"disable-popup-blocking":                             true,
		"disable-prompt-on-repost":                           true,
		"disable-renderer-backgrounding":                     true,
		"disable-sync":                                       true,
		"force-color-profile":                                "srgb",
		"metrics-recording-only":                             true,
		"safebrowsing-disable-auto-update":                   true,
		"enable-automation":                                  true,
		"password-store":                                     "basic",
		"use-mock-keychain":                                  true,
		"no-startup-window":                                  true,
	}

	envs := make([]string, len(launchOpts.Env))
	for k, v := range launchOpts.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	allocator := NewAllocator(flags, envs)
	browserProc, err := allocator.Allocate(b.Ctx, launchOpts)
	if browserProc == nil {
		k6common.Throw(rt, fmt.Errorf("unable to allocate browser: %w", err))
	}

	browser, err := common.NewBrowser(b.Ctx, b.CancelFn, browserProc, launchOpts)
	if err != nil {
		k6common.Throw(rt, err)
	}
	return browser
}

func (b *BrowserType) LaunchPersistentContext(userDataDir string, opts goja.Value) api.Browser {
	rt := k6common.GetRuntime(b.Ctx)
	k6common.Throw(rt, errors.Errorf("BrowserType.LaunchPersistentContext(userDataDir, opts) has not been implemented yet!"))
	return nil
}

func (b *BrowserType) Name() string {
	return "chromium"
}
