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

package common

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/grafana/xk6-browser/api"
	"github.com/grafana/xk6-browser/cdp"
	"github.com/grafana/xk6-browser/k6ext"
	"github.com/grafana/xk6-browser/log"

	k6modules "go.k6.io/k6/js/modules"

	"github.com/chromedp/cdproto"
	cdpt "github.com/chromedp/cdproto/target"
	"github.com/dop251/goja"
	"github.com/gorilla/websocket"
)

// Ensure Browser implements the EventEmitter and Browser interfaces.
var _ EventEmitter = &Browser{}
var _ api.Browser = &Browser{}

const (
	BrowserStateOpen int64 = iota
	BrowserStateClosing
	BrowserStateClosed
)

// Browser stores a Browser context.
type Browser struct {
	BaseEventEmitter

	ctx      context.Context
	cancelFn context.CancelFunc

	state int64

	browserProc *BrowserProcess
	launchOpts  *LaunchOptions

	cdpClient *cdp.Client

	contextsMu     sync.RWMutex
	contexts       map[string]*BrowserContext
	defaultContext *BrowserContext

	// Cancel function to stop event listening
	evCancelFn context.CancelFunc

	// Needed as the targets map will be accessed from multiple Go routines,
	// the main VU/JS go routine and the Go routine listening for CDP messages.
	pagesMu sync.RWMutex
	pages   map[string]*Page
	newPage chan *Page

	sessionsMu            sync.RWMutex
	sessions              map[string]*Session
	sessionIDtoTargetIDMu sync.RWMutex
	sessionIDtoTargetID   map[string]string

	vu k6modules.VU

	logger *log.Logger
}

// NewBrowser creates a new browser, connects to it, then returns it.
func NewBrowser(
	ctx context.Context,
	cancel context.CancelFunc,
	browserProc *BrowserProcess,
	launchOpts *LaunchOptions,
	logger *log.Logger,
) (*Browser, error) {
	b := newBrowser(ctx, cancel, browserProc, launchOpts, logger)
	if err := b.connect(); err != nil {
		return nil, err
	}
	return b, nil
}

// newBrowser returns a ready to use Browser without connecting to an actual browser.
func newBrowser(
	ctx context.Context,
	cancelFn context.CancelFunc,
	browserProc *BrowserProcess,
	launchOpts *LaunchOptions,
	logger *log.Logger,
) *Browser {
	return &Browser{
		BaseEventEmitter:    NewBaseEventEmitter(ctx),
		ctx:                 ctx,
		cdpClient:           cdp.NewClient(ctx, logger),
		cancelFn:            cancelFn,
		state:               int64(BrowserStateOpen),
		browserProc:         browserProc,
		launchOpts:          launchOpts,
		contexts:            make(map[string]*BrowserContext),
		pages:               make(map[string]*Page),
		newPage:             make(chan *Page),
		sessions:            make(map[string]*Session),
		sessionIDtoTargetID: make(map[string]string),
		vu:                  k6ext.GetVU(ctx),
		logger:              logger,
	}
}

func (b *Browser) connect() (err error) {
	b.logger.Debugf("Browser:connect", "wsURL:%q", b.browserProc.WsURL())
	if err = b.cdpClient.Connect(b.browserProc.WsURL()); err != nil {
		return fmt.Errorf("connecting to browser DevTools URL: %w", err)
	}
	fmt.Printf(">>> connected to browser at %s with client\n", b.browserProc.WsURL())

	// We don't need to lock this because `connect()` is called only in NewBrowser
	b.defaultContext = NewBrowserContext(b.ctx, b, "", NewBrowserContextOptions(), b.logger)

	return b.initEvents()
}

func (b *Browser) disposeContext(id string) error {
	b.logger.Debugf("Browser:disposeContext", "bctxid:%v", id)

	if err := b.cdpClient.Target.DisposeBrowserContext(b.ctx, id); err != nil {
		return fmt.Errorf("disposing browser context ID %s: %w", id, err)
	}

	b.contextsMu.Lock()
	defer b.contextsMu.Unlock()
	delete(b.contexts, id)

	return nil
}

func (b *Browser) getPages() []*Page {
	b.pagesMu.RLock()
	defer b.pagesMu.RUnlock()
	pages := make([]*Page, 0, len(b.pages))
	for _, p := range b.pages {
		pages = append(pages, p)
	}
	return pages
}

func (b *Browser) initEvents() error {
	var cancelCtx context.Context
	cancelCtx, b.evCancelFn = context.WithCancel(b.ctx)
	// chHandler := make(chan Event)

	// b.conn.on(cancelCtx, []string{
	// 	cdproto.EventTargetAttachedToTarget,
	// 	cdproto.EventTargetDetachedFromTarget,
	// 	EventConnectionClose,
	// }, chHandler)

	// go func() {
	// 	defer func() {
	// 		b.logger.Debugf("Browser:initEvents:defer", "ctx err: %v", cancelCtx.Err())
	// 		b.browserProc.didLoseConnection()
	// 		if b.cancelFn != nil {
	// 			b.cancelFn()
	// 		}
	// 	}()
	// 	for {
	// 		select {
	// 		case <-cancelCtx.Done():
	// 			return
	// 		case event := <-chHandler:
	// 			if ev, ok := event.data.(*cdpt.EventAttachedToTarget); ok {
	// 				b.logger.Debugf("Browser:initEvents:onAttachedToTarget", "sid:%v tid:%v", ev.SessionID, ev.TargetInfo.TargetID)
	// 				b.onAttachedToTarget(ev)
	// 			} else if ev, ok := event.data.(*cdpt.EventDetachedFromTarget); ok {
	// 				b.logger.Debugf("Browser:initEvents:onDetachedFromTarget", "sid:%v", ev.SessionID)
	// 				b.onDetachedFromTarget(ev)
	// 			} else if event.typ == EventConnectionClose {
	// 				b.logger.Debugf("Browser:initEvents:EventConnectionClose", "")
	// 				return
	// 			}
	// 		}
	// 	}
	// }()

	evtCh, _ := b.cdpClient.Subscribe(
		// TODO: Maybe have a separate Subscribe() method for non-session
		// event subscriptions?
		b.ctx, "",
		cdproto.EventTargetAttachedToTarget,
		cdproto.EventTargetDetachedFromTarget,
	)
	// TODO: Handle session creation (maybe in BrowserContext?)
	go func() {
		for {
			select {
			case event := <-evtCh:
				fmt.Printf(">>> got browser event: %#+v\n", event)
				if ev, ok := event.Data.(*cdpt.EventAttachedToTarget); ok {
					b.logger.Debugf("Browser:initEvents:onAttachedToTarget new", "sid:%v tid:%v", ev.SessionID, ev.TargetInfo.TargetID)
					b.onAttachedToTarget(ev)
				} else if ev, ok := event.Data.(*cdpt.EventDetachedFromTarget); ok {
					b.logger.Debugf("Browser:initEvents:onDetachedFromTarget new", "sid:%v", ev.SessionID)
					b.onDetachedFromTarget(ev)
				}
			case <-b.browserProc.lostConnection:
				b.logger.Debugf("Browser:initEvents", "lost browser connection")
				return
			case <-cancelCtx.Done():
				return
			}
		}
	}()
	// TODO: Handle error?
	b.cdpClient.Target.SetAutoAttach(b.ctx, true, true, true)

	return nil
}

func (b *Browser) createPage(
	bctx *BrowserContext, sessionID, targetID, openerID string, background bool,
) *Page {
	var opener *Page
	if !background {
		// Opener is nil for the initial page
		b.pagesMu.RLock()
		if t, ok := b.pages[openerID]; ok {
			opener = t
		}
		b.pagesMu.RUnlock()
	}

	page, err := NewPage(b.ctx, bctx, sessionID, cdpt.ID(targetID), opener, background, b.logger)
	if err != nil {
		isRunning := atomic.LoadInt64(&b.state) == BrowserStateOpen && b.IsConnected() // b.conn.isConnected()
		if _, ok := err.(*websocket.CloseError); !ok && !isRunning {
			// If we're no longer connected to browser, then ignore WebSocket errors
			b.logger.Debugf("Browser:createPage:return", "sid:%v tid:%v websocket err:%v", sessionID, targetID, err)
			return nil
		}
		select {
		case <-b.ctx.Done():
			b.logger.Debugf("Browser:createPage:return:<-ctx.Done",
				"sid:%v tid:%v err:%v",
				sessionID, targetID, b.ctx.Err())
			return nil // ignore
		default:
			k6ext.Panic(b.ctx, "creating a new page: %w", err)
		}
	}

	b.pagesMu.Lock()
	b.logger.Debugf("Browser:onAttachedToTarget:createPage:addTarget", "sid:%v tid:%v", sessionID, targetID)
	b.pages[targetID] = page
	b.pagesMu.Unlock()

	b.sessionIDtoTargetIDMu.Lock()
	b.logger.Debugf("Browser:createPage:sidToTid", "sid:%v tid:%v", sessionID, targetID)
	b.sessionIDtoTargetID[sessionID] = targetID
	b.sessionIDtoTargetIDMu.Unlock()

	return page
}

func (b *Browser) onAttachedToTarget(ev *cdpt.EventAttachedToTarget) {
	evti := ev.TargetInfo

	b.contextsMu.RLock()
	browserCtx := b.defaultContext
	bctx, ok := b.contexts[string(evti.BrowserContextID)]
	if ok {
		browserCtx = bctx
	}
	b.contextsMu.RUnlock()

	fmt.Printf(">>> got browser context ID from browser: %q\n", browserCtx.id)
	b.logger.Debugf("Browser:onAttachedToTarget", "sid:%v tid:%v bctxid:%v bctx nil:%t",
		ev.SessionID, evti.TargetID, evti.BrowserContextID, browserCtx == nil)

	// We're not interested in the top-level browser target, other targets or DevTools targets right now.
	isDevTools := strings.HasPrefix(evti.URL, "devtools://devtools")
	if evti.Type == "browser" || evti.Type == "other" || isDevTools {
		b.logger.Debugf("Browser:onAttachedToTarget:return", "sid:%v tid:%v (devtools)", ev.SessionID, evti.TargetID)
		return
	}

	session := b.getSession(string(ev.SessionID))
	if session == nil {
		fmt.Printf(">>> creating session ID %s\n", ev.SessionID)
		b.sessionsMu.Lock()
		session = NewSession(b.ctx, string(ev.SessionID), evti.TargetID, b.logger)
		b.logger.Debugf("Browser:onAttachedToTarget", "sid:%v tid:%v url:%q", ev.SessionID, evti.TargetID, evti.URL)
		b.sessions[string(ev.SessionID)] = session
		b.sessionsMu.Unlock()
	}

	switch evti.Type {
	case "background_page":
		b.createPage(browserCtx, string(ev.SessionID), string(evti.TargetID), string(evti.OpenerID), true)
	case "page":
		p := b.createPage(browserCtx, string(ev.SessionID), string(evti.TargetID), string(evti.OpenerID), false)
		select {
		case b.newPage <- p:
		default:
		}
	default:
		b.logger.Warnf(
			"Browser:onAttachedToTarget", "sid:%v tid:%v bctxid:%v bctx nil:%t, unknown target type: %q",
			ev.SessionID, evti.TargetID, evti.BrowserContextID, browserCtx == nil, evti.Type)
	}
}

// onDetachedFromTarget event can be issued multiple times per target if multiple
// sessions have been attached to it. So we'll remove the page only once.
func (b *Browser) onDetachedFromTarget(ev *cdpt.EventDetachedFromTarget) {
	b.sessionIDtoTargetIDMu.RLock()
	targetID, ok := b.sessionIDtoTargetID[string(ev.SessionID)]

	b.logger.Debugf("Browser:onDetachedFromTarget", "sid:%v tid:%v", ev.SessionID, targetID)
	defer b.logger.Debugf("Browser:onDetachedFromTarget:return", "sid:%v tid:%v", ev.SessionID, targetID)

	b.sessionIDtoTargetIDMu.RUnlock()
	if !ok {
		// We don't track targets of type "browser", "other" and "devtools",
		// so ignore if we don't recognize target.
		return
	}

	b.pagesMu.Lock()
	defer b.pagesMu.Unlock()
	if t, ok := b.pages[string(targetID)]; ok {
		b.logger.Debugf("Browser:onDetachedFromTarget:deletePage", "sid:%v tid:%v", ev.SessionID, targetID)

		delete(b.pages, string(targetID))
		t.didClose()
	}
}

func (b *Browser) newPageInContext(id string) (*Page, error) {
	ctx, cancel := context.WithTimeout(b.ctx, b.launchOpts.Timeout)
	defer cancel()

	var (
		page     *Page
		err      error
		targetID = make(chan string)
		wg       sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		var tid string
		select {
		case <-ctx.Done():
			err = &k6ext.UserFriendlyError{
				Err:     ctx.Err(),
				Timeout: b.launchOpts.Timeout,
			}
			return
		case tid = <-targetID:
		}

		for {
			select {
			case p := <-b.newPage:
				// We only care about the specific page we made the CreateTarget
				// request for.
				if string(p.targetID) == tid {
					page = p
					return
				}
			case <-ctx.Done():
				err = &k6ext.UserFriendlyError{
					Err:     ctx.Err(),
					Timeout: b.launchOpts.Timeout,
				}
				return
			}
		}
	}()

	tid, err := b.cdpClient.Target.CreateTarget(ctx, "about:blank", id)
	if err != nil {
		return nil, fmt.Errorf("creating a new blank page: %w", err)
	}

	// wait for the new page to be created in onAttachedToTarget
	targetID <- tid
	wg.Wait()

	return page, err
}

func (b *Browser) getSession(sid string) *Session {
	b.sessionsMu.RLock()
	defer b.sessionsMu.RUnlock()
	return b.sessions[sid]
}

func (b *Browser) closeSession(sid string) {
	b.logger.Debugf("Browser:closeSession", "sid:%v", sid)
	b.sessionsMu.Lock()
	// if session, ok := b.sessions[sid]; ok {
	// 	session.close()
	// }
	delete(b.sessions, sid)
	b.sessionsMu.Unlock()
}

// Close shuts down the browser.
func (b *Browser) Close() {
	defer func() {
		if err := b.browserProc.userDataDir.Cleanup(); err != nil {
			b.logger.Errorf("Browser:Close", "cleaning up the user data directory: %v", err)
		}
	}()

	b.logger.Debugf("Browser:Close", "")
	if !atomic.CompareAndSwapInt64(&b.state, b.state, BrowserStateClosing) {
		// If we're already in a closing state then no need to continue.
		b.logger.Debugf("Browser:Close", "already in a closing state")
		return
	}

	atomic.CompareAndSwapInt64(&b.state, b.state, BrowserStateClosed)

	if err := b.cdpClient.Browser.Close(b.ctx); err != nil {
		if _, ok := err.(*websocket.CloseError); !ok {
			k6ext.Panic(b.ctx, "closing the browser: %v", err)
		}
	}

	b.cdpClient.Disconnect()
	b.browserProc.GracefulClose()
	b.browserProc.Terminate()
}

// Contexts returns list of browser contexts.
func (b *Browser) Contexts() []api.BrowserContext {
	b.contextsMu.RLock()
	defer b.contextsMu.RUnlock()

	contexts := make([]api.BrowserContext, 0, len(b.contexts))
	for _, b := range b.contexts {
		contexts = append(contexts, b)
	}

	return contexts
}

// IsConnected returns whether the WebSocket connection to the browser process
// is active or not.
func (b *Browser) IsConnected() bool {
	return b.browserProc.isConnected()
}

// NewContext creates a new incognito-like browser context.
func (b *Browser) NewContext(opts goja.Value) api.BrowserContext {
	bctxID, err := b.cdpClient.Target.CreateBrowserContext(b.ctx, true)
	if err != nil {
		k6ext.Panic(b.ctx, "creating a new browser context: %w", err)
	}

	browserCtxOpts := NewBrowserContextOptions()
	if err := browserCtxOpts.Parse(b.ctx, opts); err != nil {
		k6ext.Panic(b.ctx, "parsing newContext options: %w", err)
	}

	b.contextsMu.Lock()
	defer b.contextsMu.Unlock()
	browserCtx := NewBrowserContext(b.ctx, b, bctxID, browserCtxOpts, b.logger)
	b.contexts[bctxID] = browserCtx

	return browserCtx
}

// NewPage creates a new tab in the browser window.
func (b *Browser) NewPage(opts goja.Value) api.Page {
	browserCtx := b.NewContext(opts)
	return browserCtx.NewPage()
}

// On returns a Promise that is resolved when the browser process is disconnected.
// The only accepted event value is "disconnected".
func (b *Browser) On(event string) *goja.Promise {
	if event != EventBrowserDisconnected {
		k6ext.Panic(b.ctx, "unknown browser event: %q, must be %q", event, EventBrowserDisconnected)
	}
	return k6ext.Promise(b.ctx, func() (interface{}, error) {
		select {
		case <-b.browserProc.lostConnection:
			return true, nil
		case <-b.ctx.Done():
			return nil, fmt.Errorf("browser.on promise rejected: %w", b.ctx.Err())
		}
	})
}

// UserAgent returns the controlled browser's user agent string.
func (b *Browser) UserAgent() string {
	_, _, _, ua, _, err := b.cdpClient.Browser.GetVersion(b.ctx)
	if err != nil {
		k6ext.Panic(b.ctx, "getting browser user agent: %w", err)
	}

	return ua
}

// Version returns the controlled browser's version.
func (b *Browser) Version() string {
	_, product, _, _, _, err := b.cdpClient.Browser.GetVersion(b.ctx)
	if err != nil {
		k6ext.Panic(b.ctx, "getting browser version: %w", err)
	}

	i := strings.Index(product, "/")
	if i == -1 {
		return product
	}
	return product[i+1:]
}
