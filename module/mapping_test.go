package module

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"

	"github.com/grafana/xk6-browser/browser"

	k6common "go.k6.io/k6/js/common"
	k6modulestest "go.k6.io/k6/js/modulestest"
	k6lib "go.k6.io/k6/lib"
	k6metrics "go.k6.io/k6/metrics"
)

// customMappings is a list of custom mappings for our module API.
// Some of them are wildcards, such as query to $ mapping; and
// others are for publicly accessible fields, such as mapping
// of page.keyboard to Page.getKeyboard.
func customMappings() map[string]string {
	return map[string]string{
		// wildcards
		"pageAPI.query":             "$",
		"pageAPI.queryAll":          "$$",
		"frameAPI.query":            "$",
		"frameAPI.queryAll":         "$$",
		"elementHandleAPI.query":    "$",
		"elementHandleAPI.queryAll": "$$",
		// getters
		"pageAPI.getKeyboard":    "keyboard",
		"pageAPI.getMouse":       "mouse",
		"pageAPI.getTouchscreen": "touchscreen",
		// internal methods
		"elementHandleAPI.objectID":    "",
		"frameAPI.id":                  "",
		"frameAPI.loaderID":            "",
		"JSHandleAPI.objectID":         "",
		"browserAPI.close":             "",
		"frameAPI.evaluateWithContext": "",
		// TODO: browser.on method is unexposed until more event
		// types other than 'disconnect' are supported.
		// See: https://github.com/grafana/xk6-browser/issues/913
		"browserAPI.on": "",
	}
}

// TestMappings tests that all the methods of the API (api/) are
// to the module. This is to ensure that we don't forget to map
// a new method to the module.
func TestMappings(t *testing.T) {
	t.Parallel()

	type test struct {
		apiInterface any
		mapp         func() mapping
	}

	var (
		vu = &k6modulestest.VU{
			RuntimeField: goja.New(),
			InitEnvField: &k6common.InitEnvironment{
				TestPreInitState: &k6lib.TestPreInitState{
					Registry: k6metrics.NewRegistry(),
				},
			},
		}
		customMappings = customMappings()
	)

	// testMapping tests that all the methods of an API are mapped
	// to the module. And wildcards are mapped correctly and their
	// methods are not mapped.
	testMapping := func(t *testing.T, tt test) {
		t.Helper()

		var (
			typ    = reflect.TypeOf(tt.apiInterface).Elem()
			mapped = tt.mapp()
			tested = make(map[string]bool)
		)
		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i)
			require.NotNil(t, method)

			// goja uses methods that starts with lowercase.
			// so we need to convert the first letter to lowercase.
			m := toFirstLetterLower(method.Name)

			cm, cmok := isCustomMapping(customMappings, typ.Name(), m)
			// if the method is a custom mapping, it should not be
			// mapped to the module. so we should not find it in
			// the mapped methods.
			if _, ok := mapped[m]; cmok && ok {
				t.Errorf("method %q should not be mapped", m)
			}
			// a custom mapping with an empty string means that
			// the method should not exist on the API.
			if cmok && cm == "" {
				continue
			}
			// change the method name if it is mapped to a custom
			// method. these custom methods are not exist on our
			// API. so we need to use the mapped method instead.
			if cmok {
				m = cm
			}
			if _, ok := mapped[m]; !ok {
				t.Errorf("method %q not found", m)
			}
			// to detect if a method is redundantly mapped.
			tested[m] = true
		}
		// detect redundant mappings.
		for m := range mapped {
			if !tested[m] {
				t.Errorf("method %q is redundant", m)
			}
		}
	}

	for name, tt := range map[string]test{
		"browser": {
			apiInterface: (*browserAPI)(nil),
			mapp: func() mapping {
				return mapBrowser(moduleVU{VU: vu})
			},
		},
		"browserContext": {
			apiInterface: (*browserContextAPI)(nil),
			mapp: func() mapping {
				return mapBrowserContext(moduleVU{VU: vu}, &browser.BrowserContext{})
			},
		},
		"page": {
			apiInterface: (*pageAPI)(nil),
			mapp: func() mapping {
				return mapPage(moduleVU{VU: vu}, &browser.Page{
					Keyboard:    &browser.Keyboard{},
					Mouse:       &browser.Mouse{},
					Touchscreen: &browser.Touchscreen{},
				})
			},
		},
		"elementHandle": {
			apiInterface: (*elementHandleAPI)(nil),
			mapp: func() mapping {
				return mapElementHandle(moduleVU{VU: vu}, &browser.ElementHandle{})
			},
		},
		"jsHandle": {
			apiInterface: (*browser.JSHandleAPI)(nil),
			mapp: func() mapping {
				return mapJSHandle(moduleVU{VU: vu}, &browser.BaseJSHandle{})
			},
		},
		"frame": {
			apiInterface: (*frameAPI)(nil),
			mapp: func() mapping {
				return mapFrame(moduleVU{VU: vu}, &browser.Frame{})
			},
		},
		"mapRequest": {
			apiInterface: (*requestAPI)(nil),
			mapp: func() mapping {
				return mapRequest(moduleVU{VU: vu}, &browser.Request{})
			},
		},
		"mapResponse": {
			apiInterface: (*responseAPI)(nil),
			mapp: func() mapping {
				return mapResponse(moduleVU{VU: vu}, &browser.Response{})
			},
		},
		"mapWorker": {
			apiInterface: (*workerAPI)(nil),
			mapp: func() mapping {
				return mapWorker(moduleVU{VU: vu}, &browser.Worker{})
			},
		},
		"mapLocator": {
			apiInterface: (*locatorAPI)(nil),
			mapp: func() mapping {
				return mapLocator(moduleVU{VU: vu}, &browser.Locator{})
			},
		},
		"mapConsoleMessage": {
			apiInterface: (*consoleMessageAPI)(nil),
			mapp: func() mapping {
				return mapConsoleMessage(moduleVU{VU: vu}, &browser.ConsoleMessage{})
			},
		},
	} {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			testMapping(t, tt)
		})
	}
}

// toFirstLetterLower converts the first letter of the string to lower case.
func toFirstLetterLower(s string) string {
	// Special cases.
	// Instead of loading up an acronyms list, just do this.
	// Good enough for our purposes.
	special := map[string]string{
		"ID":        "id",
		"JSON":      "json",
		"JSONValue": "jsonValue",
		"URL":       "url",
	}
	if v, ok := special[s]; ok {
		return v
	}
	if s == "" {
		return ""
	}

	return strings.ToLower(s[:1]) + s[1:]
}

// isCustomMapping returns true if the method is a custom mapping
// and returns the name of the method to be called instead of the
// original one.
func isCustomMapping(customMappings map[string]string, typ, method string) (string, bool) {
	name := typ + "." + method

	if s, ok := customMappings[name]; ok {
		return s, ok
	}

	return "", false
}

// ----------------------------------------------------------------------------
// JavaScript API definitions.
// ----------------------------------------------------------------------------

// browserAPI is the public interface of a CDP browser.
type browserAPI interface {
	Close()
	Context() *browser.BrowserContext
	CloseContext()
	IsConnected() bool
	NewContext(opts goja.Value) (*browser.BrowserContext, error)
	NewPage(opts goja.Value) (*browser.Page, error)
	On(string) (bool, error)
	UserAgent() string
	Version() string
}

// browserContextAPI is the public interface of a CDP browser context.
type browserContextAPI interface {
	AddCookies(cookies []*browser.Cookie) error
	AddInitScript(script goja.Value, arg goja.Value) error
	Browser() *browser.Browser
	ClearCookies() error
	ClearPermissions()
	Close()
	Cookies(urls ...string) ([]*browser.Cookie, error)
	ExposeBinding(name string, callback goja.Callable, opts goja.Value)
	ExposeFunction(name string, callback goja.Callable)
	GrantPermissions(permissions []string, opts goja.Value)
	NewCDPSession() any
	NewPage() (*browser.Page, error)
	Pages() []*browser.Page
	Route(url goja.Value, handler goja.Callable)
	SetDefaultNavigationTimeout(timeout int64)
	SetDefaultTimeout(timeout int64)
	SetExtraHTTPHeaders(headers map[string]string) error
	SetGeolocation(geolocation goja.Value)
	SetHTTPCredentials(httpCredentials goja.Value)
	SetOffline(offline bool)
	StorageState(opts goja.Value)
	Unroute(url goja.Value, handler goja.Callable)
	WaitForEvent(event string, optsOrPredicate goja.Value) (any, error)
}

// pageAPI is the interface of a single browser tab.
type pageAPI interface {
	AddInitScript(script goja.Value, arg goja.Value)
	AddScriptTag(opts goja.Value)
	AddStyleTag(opts goja.Value)
	BringToFront()
	Check(selector string, opts goja.Value)
	Click(selector string, opts goja.Value) error
	Close(opts goja.Value) error
	Content() string
	Context() *browser.BrowserContext
	Dblclick(selector string, opts goja.Value)
	DispatchEvent(selector string, typ string, eventInit goja.Value, opts goja.Value)
	DragAndDrop(source string, target string, opts goja.Value)
	EmulateMedia(opts goja.Value)
	EmulateVisionDeficiency(typ string)
	Evaluate(pageFunc goja.Value, arg ...goja.Value) any
	EvaluateHandle(pageFunc goja.Value, arg ...goja.Value) (browser.JSHandleAPI, error)
	ExposeBinding(name string, callback goja.Callable, opts goja.Value)
	ExposeFunction(name string, callback goja.Callable)
	Fill(selector string, value string, opts goja.Value)
	Focus(selector string, opts goja.Value)
	Frame(frameSelector goja.Value) *browser.Frame
	Frames() []*browser.Frame
	GetAttribute(selector string, name string, opts goja.Value) goja.Value
	GetKeyboard() *browser.Keyboard
	GetMouse() *browser.Mouse
	GetTouchscreen() *browser.Touchscreen
	GoBack(opts goja.Value) *browser.Response
	GoForward(opts goja.Value) *browser.Response
	Goto(url string, opts goja.Value) (*browser.Response, error)
	Hover(selector string, opts goja.Value)
	InnerHTML(selector string, opts goja.Value) string
	InnerText(selector string, opts goja.Value) string
	InputValue(selector string, opts goja.Value) string
	IsChecked(selector string, opts goja.Value) bool
	IsClosed() bool
	IsDisabled(selector string, opts goja.Value) bool
	IsEditable(selector string, opts goja.Value) bool
	IsEnabled(selector string, opts goja.Value) bool
	IsHidden(selector string, opts goja.Value) bool
	IsVisible(selector string, opts goja.Value) bool
	Locator(selector string, opts goja.Value) *browser.Locator
	MainFrame() *browser.Frame
	On(event string, handler func(*browser.ConsoleMessage) error) error
	Opener() pageAPI
	Pause()
	Pdf(opts goja.Value) []byte
	Press(selector string, key string, opts goja.Value)
	Query(selector string) (*browser.ElementHandle, error)
	QueryAll(selector string) ([]*browser.ElementHandle, error)
	Reload(opts goja.Value) *browser.Response
	Route(url goja.Value, handler goja.Callable)
	Screenshot(opts goja.Value) goja.ArrayBuffer
	SelectOption(selector string, values goja.Value, opts goja.Value) []string
	SetContent(html string, opts goja.Value)
	SetDefaultNavigationTimeout(timeout int64)
	SetDefaultTimeout(timeout int64)
	SetExtraHTTPHeaders(headers map[string]string)
	SetInputFiles(selector string, files goja.Value, opts goja.Value)
	SetViewportSize(viewportSize goja.Value)
	Tap(selector string, opts goja.Value)
	TextContent(selector string, opts goja.Value) string
	ThrottleCPU(browser.CPUProfile) error
	ThrottleNetwork(browser.NetworkProfile) error
	Title() string
	Type(selector string, text string, opts goja.Value)
	Uncheck(selector string, opts goja.Value)
	Unroute(url goja.Value, handler goja.Callable)
	URL() string
	Video() any
	ViewportSize() map[string]float64
	WaitForEvent(event string, optsOrPredicate goja.Value) any
	WaitForFunction(fn, opts goja.Value, args ...goja.Value) (any, error)
	WaitForLoadState(state string, opts goja.Value)
	WaitForNavigation(opts goja.Value) (*browser.Response, error)
	WaitForRequest(urlOrPredicate, opts goja.Value) *browser.Request
	WaitForResponse(urlOrPredicate, opts goja.Value) *browser.Response
	WaitForSelector(selector string, opts goja.Value) (*browser.ElementHandle, error)
	WaitForTimeout(timeout int64)
	Workers() []*browser.Worker
}

// consoleMessageAPI is the interface of a console message.
type consoleMessageAPI interface {
	Args() []browser.JSHandleAPI
	Page() *browser.Page
	Text() string
	Type() string
}

// frameAPI is the interface of a CDP target frame.
type frameAPI interface {
	AddScriptTag(opts goja.Value)
	AddStyleTag(opts goja.Value)
	Check(selector string, opts goja.Value)
	ChildFrames() []*browser.Frame
	Click(selector string, opts goja.Value) error
	Content() string
	Dblclick(selector string, opts goja.Value)
	DispatchEvent(selector string, typ string, eventInit goja.Value, opts goja.Value)
	// EvaluateWithContext for internal use only
	EvaluateWithContext(ctx context.Context, pageFunc goja.Value, args ...goja.Value) (any, error)
	Evaluate(pageFunc goja.Value, args ...goja.Value) any
	EvaluateHandle(pageFunc goja.Value, args ...goja.Value) (browser.JSHandleAPI, error)
	Fill(selector string, value string, opts goja.Value)
	Focus(selector string, opts goja.Value)
	FrameElement() (*browser.ElementHandle, error)
	GetAttribute(selector string, name string, opts goja.Value) goja.Value
	Goto(url string, opts goja.Value) (*browser.Response, error)
	Hover(selector string, opts goja.Value)
	InnerHTML(selector string, opts goja.Value) string
	InnerText(selector string, opts goja.Value) string
	InputValue(selector string, opts goja.Value) string
	IsChecked(selector string, opts goja.Value) bool
	IsDetached() bool
	IsDisabled(selector string, opts goja.Value) bool
	IsEditable(selector string, opts goja.Value) bool
	IsEnabled(selector string, opts goja.Value) bool
	IsHidden(selector string, opts goja.Value) bool
	IsVisible(selector string, opts goja.Value) bool
	ID() string
	LoaderID() string
	Locator(selector string, opts goja.Value) *browser.Locator
	Name() string
	Query(selector string) (*browser.ElementHandle, error)
	QueryAll(selector string) ([]*browser.ElementHandle, error)
	Page() *browser.Page
	ParentFrame() *browser.Frame
	Press(selector string, key string, opts goja.Value)
	SelectOption(selector string, values goja.Value, opts goja.Value) []string
	SetContent(html string, opts goja.Value)
	SetInputFiles(selector string, files goja.Value, opts goja.Value)
	Tap(selector string, opts goja.Value)
	TextContent(selector string, opts goja.Value) string
	Title() string
	Type(selector string, text string, opts goja.Value)
	Uncheck(selector string, opts goja.Value)
	URL() string
	WaitForFunction(pageFunc, opts goja.Value, args ...goja.Value) (any, error)
	WaitForLoadState(state string, opts goja.Value)
	WaitForNavigation(opts goja.Value) (*browser.Response, error)
	WaitForSelector(selector string, opts goja.Value) (*browser.ElementHandle, error)
	WaitForTimeout(timeout int64)
}

// elementHandleAPI is the interface of an in-page DOM element.
type elementHandleAPI interface {
	browser.JSHandleAPI

	BoundingBox() *browser.Rect
	Check(opts goja.Value)
	Click(opts goja.Value) error
	ContentFrame() (*browser.Frame, error)
	Dblclick(opts goja.Value)
	DispatchEvent(typ string, props goja.Value)
	Fill(value string, opts goja.Value)
	Focus()
	GetAttribute(name string) goja.Value
	Hover(opts goja.Value)
	InnerHTML() string
	InnerText() string
	InputValue(opts goja.Value) string
	IsChecked() bool
	IsDisabled() bool
	IsEditable() bool
	IsEnabled() bool
	IsHidden() bool
	IsVisible() bool
	OwnerFrame() (*browser.Frame, error)
	Press(key string, opts goja.Value)
	Query(selector string) (*browser.ElementHandle, error)
	QueryAll(selector string) ([]*browser.ElementHandle, error)
	Screenshot(opts goja.Value) goja.ArrayBuffer
	ScrollIntoViewIfNeeded(opts goja.Value)
	SelectOption(values goja.Value, opts goja.Value) []string
	SelectText(opts goja.Value)
	SetInputFiles(files goja.Value, opts goja.Value)
	Tap(opts goja.Value)
	TextContent() string
	Type(text string, opts goja.Value)
	Uncheck(opts goja.Value)
	WaitForElementState(state string, opts goja.Value)
	WaitForSelector(selector string, opts goja.Value) (*browser.ElementHandle, error)
}

// requestAPI is the interface of an HTTP request.
type requestAPI interface {
	AllHeaders() map[string]string
	Failure() goja.Value
	Frame() *browser.Frame
	HeaderValue(string) goja.Value
	Headers() map[string]string
	HeadersArray() []browser.HTTPHeader
	IsNavigationRequest() bool
	Method() string
	PostData() string
	PostDataBuffer() goja.ArrayBuffer
	PostDataJSON() string
	RedirectedFrom() requestAPI
	RedirectedTo() requestAPI
	ResourceType() string
	Response() *browser.Response
	Size() browser.HTTPMessageSize
	Timing() goja.Value
	URL() string
}

// responseAPI is the interface of an HTTP response.
type responseAPI interface {
	AllHeaders() map[string]string
	Body() goja.ArrayBuffer
	Finished() bool
	Frame() *browser.Frame
	HeaderValue(string) goja.Value
	HeaderValues(string) []string
	Headers() map[string]string
	HeadersArray() []browser.HTTPHeader
	JSON() goja.Value
	Ok() bool
	Request() *browser.Request
	SecurityDetails() goja.Value
	ServerAddr() goja.Value
	Size() browser.HTTPMessageSize
	Status() int64
	StatusText() string
	URL() string
}

// locatorAPI represents a way to find element(s) on a page at any moment.
type locatorAPI interface {
	Click(opts goja.Value) error
	Dblclick(opts goja.Value)
	Check(opts goja.Value)
	Uncheck(opts goja.Value)
	IsChecked(opts goja.Value) bool
	IsEditable(opts goja.Value) bool
	IsEnabled(opts goja.Value) bool
	IsDisabled(opts goja.Value) bool
	IsVisible(opts goja.Value) bool
	IsHidden(opts goja.Value) bool
	Fill(value string, opts goja.Value)
	Focus(opts goja.Value)
	GetAttribute(name string, opts goja.Value) goja.Value
	InnerHTML(opts goja.Value) string
	InnerText(opts goja.Value) string
	TextContent(opts goja.Value) string
	InputValue(opts goja.Value) string
	SelectOption(values goja.Value, opts goja.Value) []string
	Press(key string, opts goja.Value)
	Type(text string, opts goja.Value)
	Hover(opts goja.Value)
	Tap(opts goja.Value)
	DispatchEvent(typ string, eventInit, opts goja.Value)
	WaitFor(opts goja.Value)
}

// keyboardAPI is the interface of a keyboard input device.
// TODO: map this to page.GetKeyboard(). Currently, the common.Keyboard type
// mapping is not tested using this interface. We use the concrete type
// without testing its exported methods.
type keyboardAPI interface { //nolint: unused
	Down(key string)
	InsertText(char string)
	Press(key string, opts goja.Value)
	Type(text string, opts goja.Value)
	Up(key string)
}

// touchscreenAPI is the interface of a touchscreen.
// TODO: map this to page.GetTouchscreen(). Currently, the common.TouchscreenAPI type
// mapping is not tested using this interface. We use the concrete type
// without testing its exported methods.
type touchscreenAPI interface { //nolint: unused
	Tap(x float64, y float64)
}

// mouseAPI is the interface of a mouse input device.
// TODO: map this to page.GetMouse(). Currently, the common.Mouse type
// mapping is not tested using this interface. We use the concrete type
// without testing its exported methods.
type mouseAPI interface { //nolint: unused
	Click(x float64, y float64, opts goja.Value)
	DblClick(x float64, y float64, opts goja.Value)
	Down(x float64, y float64, opts goja.Value)
	Move(x float64, y float64, opts goja.Value)
	Up(x float64, y float64, opts goja.Value)
	// Wheel(opts goja.Value)
}

// workerAPI is the interface of a web worker.
type workerAPI interface {
	Evaluate(pageFunc goja.Value, args ...goja.Value) any
	EvaluateHandle(pageFunc goja.Value, args ...goja.Value) (browser.JSHandleAPI, error)
	URL() string
}
