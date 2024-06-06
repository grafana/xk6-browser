package browser

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/grafana/sobek"
	"github.com/stretchr/testify/require"

	"github.com/grafana/xk6-browser/common"

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
			RuntimeField: sobek.New(),
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
				return mapBrowserContext(moduleVU{VU: vu}, &common.BrowserContext{})
			},
		},
		"page": {
			apiInterface: (*pageAPI)(nil),
			mapp: func() mapping {
				return mapPage(moduleVU{VU: vu}, &common.Page{
					Keyboard:    &common.Keyboard{},
					Mouse:       &common.Mouse{},
					Touchscreen: &common.Touchscreen{},
				})
			},
		},
		"elementHandle": {
			apiInterface: (*elementHandleAPI)(nil),
			mapp: func() mapping {
				return mapElementHandle(moduleVU{VU: vu}, &common.ElementHandle{})
			},
		},
		"jsHandle": {
			apiInterface: (*common.JSHandleAPI)(nil),
			mapp: func() mapping {
				return mapJSHandle(moduleVU{VU: vu}, &common.BaseJSHandle{})
			},
		},
		"frame": {
			apiInterface: (*frameAPI)(nil),
			mapp: func() mapping {
				return mapFrame(moduleVU{VU: vu}, &common.Frame{})
			},
		},
		"mapRequest": {
			apiInterface: (*requestAPI)(nil),
			mapp: func() mapping {
				return mapRequest(moduleVU{VU: vu}, &common.Request{})
			},
		},
		"mapResponse": {
			apiInterface: (*responseAPI)(nil),
			mapp: func() mapping {
				return mapResponse(moduleVU{VU: vu}, &common.Response{})
			},
		},
		"mapWorker": {
			apiInterface: (*workerAPI)(nil),
			mapp: func() mapping {
				return mapWorker(moduleVU{VU: vu}, &common.Worker{})
			},
		},
		"mapLocator": {
			apiInterface: (*locatorAPI)(nil),
			mapp: func() mapping {
				return mapLocator(moduleVU{VU: vu}, &common.Locator{})
			},
		},
		"mapConsoleMessage": {
			apiInterface: (*consoleMessageAPI)(nil),
			mapp: func() mapping {
				return mapConsoleMessage(moduleVU{VU: vu}, &common.ConsoleMessage{})
			},
		},
		"mapTouchscreen": {
			apiInterface: (*touchscreenAPI)(nil),
			mapp: func() mapping {
				return mapTouchscreen(moduleVU{VU: vu}, &common.Touchscreen{})
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
	Context() *common.BrowserContext
	CloseContext()
	IsConnected() bool
	NewContext(opts sobek.Value) (*common.BrowserContext, error)
	NewPage(opts sobek.Value) (*common.Page, error)
	On(string) (bool, error)
	UserAgent() string
	Version() string
}

// browserContextAPI is the public interface of a CDP browser context.
type browserContextAPI interface {
	AddCookies(cookies []*common.Cookie) error
	AddInitScript(script sobek.Value, arg sobek.Value) error
	Browser() *common.Browser
	ClearCookies() error
	ClearPermissions()
	Close()
	Cookies(urls ...string) ([]*common.Cookie, error)
	GrantPermissions(permissions []string, opts sobek.Value)
	NewPage() (*common.Page, error)
	Pages() []*common.Page
	SetDefaultNavigationTimeout(timeout int64)
	SetDefaultTimeout(timeout int64)
	SetGeolocation(geolocation sobek.Value)
	SetHTTPCredentials(httpCredentials sobek.Value)
	SetOffline(offline bool)
	WaitForEvent(event string, optsOrPredicate sobek.Value) (any, error)
}

// pageAPI is the interface of a single browser tab.
type pageAPI interface {
	BringToFront()
	Check(selector string, opts sobek.Value)
	Click(selector string, opts sobek.Value) error
	Close(opts sobek.Value) error
	Content() string
	Context() *common.BrowserContext
	Dblclick(selector string, opts sobek.Value)
	DispatchEvent(selector string, typ string, eventInit sobek.Value, opts sobek.Value)
	EmulateMedia(opts sobek.Value)
	EmulateVisionDeficiency(typ string)
	Evaluate(pageFunc sobek.Value, arg ...sobek.Value) any
	EvaluateHandle(pageFunc sobek.Value, arg ...sobek.Value) (common.JSHandleAPI, error)
	Fill(selector string, value string, opts sobek.Value)
	Focus(selector string, opts sobek.Value)
	Frames() []*common.Frame
	GetAttribute(selector string, name string, opts sobek.Value) sobek.Value
	GetKeyboard() *common.Keyboard
	GetMouse() *common.Mouse
	GetTouchscreen() *common.Touchscreen
	Goto(url string, opts sobek.Value) (*common.Response, error)
	Hover(selector string, opts sobek.Value)
	InnerHTML(selector string, opts sobek.Value) string
	InnerText(selector string, opts sobek.Value) string
	InputValue(selector string, opts sobek.Value) string
	IsChecked(selector string, opts sobek.Value) bool
	IsClosed() bool
	IsDisabled(selector string, opts sobek.Value) bool
	IsEditable(selector string, opts sobek.Value) bool
	IsEnabled(selector string, opts sobek.Value) bool
	IsHidden(selector string, opts sobek.Value) bool
	IsVisible(selector string, opts sobek.Value) bool
	Locator(selector string, opts sobek.Value) *common.Locator
	MainFrame() *common.Frame
	On(event string, handler func(*common.ConsoleMessage) error) error
	Opener() pageAPI
	Press(selector string, key string, opts sobek.Value)
	Query(selector string) (*common.ElementHandle, error)
	QueryAll(selector string) ([]*common.ElementHandle, error)
	Reload(opts sobek.Value) *common.Response
	Screenshot(opts sobek.Value) sobek.ArrayBuffer
	SelectOption(selector string, values sobek.Value, opts sobek.Value) []string
	SetContent(html string, opts sobek.Value)
	SetDefaultNavigationTimeout(timeout int64)
	SetDefaultTimeout(timeout int64)
	SetExtraHTTPHeaders(headers map[string]string)
	SetInputFiles(selector string, files sobek.Value, opts sobek.Value)
	SetViewportSize(viewportSize sobek.Value)
	Tap(selector string, opts sobek.Value) (*sobek.Promise, error)
	TextContent(selector string, opts sobek.Value) string
	ThrottleCPU(common.CPUProfile) error
	ThrottleNetwork(common.NetworkProfile) error
	Title() string
	Type(selector string, text string, opts sobek.Value)
	Uncheck(selector string, opts sobek.Value)
	URL() string
	ViewportSize() map[string]float64
	WaitForFunction(fn, opts sobek.Value, args ...sobek.Value) (any, error)
	WaitForLoadState(state string, opts sobek.Value)
	WaitForNavigation(opts sobek.Value) (*common.Response, error)
	WaitForSelector(selector string, opts sobek.Value) (*common.ElementHandle, error)
	WaitForTimeout(timeout int64)
	Workers() []*common.Worker
}

// consoleMessageAPI is the interface of a console message.
type consoleMessageAPI interface {
	Args() []common.JSHandleAPI
	Page() *common.Page
	Text() string
	Type() string
}

// frameAPI is the interface of a CDP target frame.
type frameAPI interface {
	Check(selector string, opts sobek.Value)
	ChildFrames() []*common.Frame
	Click(selector string, opts sobek.Value) error
	Content() string
	Dblclick(selector string, opts sobek.Value)
	DispatchEvent(selector string, typ string, eventInit sobek.Value, opts sobek.Value)
	// EvaluateWithContext for internal use only
	EvaluateWithContext(ctx context.Context, pageFunc sobek.Value, args ...sobek.Value) (any, error)
	Evaluate(pageFunc sobek.Value, args ...sobek.Value) any
	EvaluateHandle(pageFunc sobek.Value, args ...sobek.Value) (common.JSHandleAPI, error)
	Fill(selector string, value string, opts sobek.Value)
	Focus(selector string, opts sobek.Value)
	FrameElement() (*common.ElementHandle, error)
	GetAttribute(selector string, name string, opts sobek.Value) sobek.Value
	Goto(url string, opts sobek.Value) (*common.Response, error)
	Hover(selector string, opts sobek.Value)
	InnerHTML(selector string, opts sobek.Value) string
	InnerText(selector string, opts sobek.Value) string
	InputValue(selector string, opts sobek.Value) string
	IsChecked(selector string, opts sobek.Value) bool
	IsDetached() bool
	IsDisabled(selector string, opts sobek.Value) bool
	IsEditable(selector string, opts sobek.Value) bool
	IsEnabled(selector string, opts sobek.Value) bool
	IsHidden(selector string, opts sobek.Value) bool
	IsVisible(selector string, opts sobek.Value) bool
	ID() string
	LoaderID() string
	Locator(selector string, opts sobek.Value) *common.Locator
	Name() string
	Query(selector string) (*common.ElementHandle, error)
	QueryAll(selector string) ([]*common.ElementHandle, error)
	Page() *common.Page
	ParentFrame() *common.Frame
	Press(selector string, key string, opts sobek.Value)
	SelectOption(selector string, values sobek.Value, opts sobek.Value) []string
	SetContent(html string, opts sobek.Value)
	SetInputFiles(selector string, files sobek.Value, opts sobek.Value)
	Tap(selector string, opts sobek.Value) (*sobek.Promise, error)
	TextContent(selector string, opts sobek.Value) string
	Title() string
	Type(selector string, text string, opts sobek.Value)
	Uncheck(selector string, opts sobek.Value)
	URL() string
	WaitForFunction(pageFunc, opts sobek.Value, args ...sobek.Value) (any, error)
	WaitForLoadState(state string, opts sobek.Value)
	WaitForNavigation(opts sobek.Value) (*common.Response, error)
	WaitForSelector(selector string, opts sobek.Value) (*common.ElementHandle, error)
	WaitForTimeout(timeout int64)
}

// elementHandleAPI is the interface of an in-page DOM element.
type elementHandleAPI interface {
	common.JSHandleAPI

	BoundingBox() *common.Rect
	Check(opts sobek.Value)
	Click(opts sobek.Value) error
	ContentFrame() (*common.Frame, error)
	Dblclick(opts sobek.Value)
	DispatchEvent(typ string, props sobek.Value)
	Fill(value string, opts sobek.Value)
	Focus()
	GetAttribute(name string) sobek.Value
	Hover(opts sobek.Value)
	InnerHTML() string
	InnerText() string
	InputValue(opts sobek.Value) string
	IsChecked() bool
	IsDisabled() bool
	IsEditable() bool
	IsEnabled() bool
	IsHidden() bool
	IsVisible() bool
	OwnerFrame() (*common.Frame, error)
	Press(key string, opts sobek.Value)
	Query(selector string) (*common.ElementHandle, error)
	QueryAll(selector string) ([]*common.ElementHandle, error)
	Screenshot(opts sobek.Value) sobek.ArrayBuffer
	ScrollIntoViewIfNeeded(opts sobek.Value)
	SelectOption(values sobek.Value, opts sobek.Value) []string
	SelectText(opts sobek.Value)
	SetInputFiles(files sobek.Value, opts sobek.Value)
	Tap(opts sobek.Value) (*sobek.Promise, error)
	TextContent() string
	Type(text string, opts sobek.Value)
	Uncheck(opts sobek.Value)
	WaitForElementState(state string, opts sobek.Value)
	WaitForSelector(selector string, opts sobek.Value) (*common.ElementHandle, error)
}

// requestAPI is the interface of an HTTP request.
type requestAPI interface {
	AllHeaders() map[string]string
	Frame() *common.Frame
	HeaderValue(string) sobek.Value
	Headers() map[string]string
	HeadersArray() []common.HTTPHeader
	IsNavigationRequest() bool
	Method() string
	PostData() string
	PostDataBuffer() sobek.ArrayBuffer
	ResourceType() string
	Response() *common.Response
	Size() common.HTTPMessageSize
	Timing() sobek.Value
	URL() string
}

// responseAPI is the interface of an HTTP response.
type responseAPI interface {
	AllHeaders() map[string]string
	Body() sobek.ArrayBuffer
	Frame() *common.Frame
	HeaderValue(string) sobek.Value
	HeaderValues(string) []string
	Headers() map[string]string
	HeadersArray() []common.HTTPHeader
	JSON() sobek.Value
	Ok() bool
	Request() *common.Request
	SecurityDetails() sobek.Value
	ServerAddr() sobek.Value
	Size() common.HTTPMessageSize
	Status() int64
	StatusText() string
	URL() string
}

// locatorAPI represents a way to find element(s) on a page at any moment.
type locatorAPI interface {
	Clear(opts *common.FrameFillOptions) error
	Click(opts sobek.Value) error
	Dblclick(opts sobek.Value)
	Check(opts sobek.Value)
	Uncheck(opts sobek.Value)
	IsChecked(opts sobek.Value) bool
	IsEditable(opts sobek.Value) bool
	IsEnabled(opts sobek.Value) bool
	IsDisabled(opts sobek.Value) bool
	IsVisible(opts sobek.Value) bool
	IsHidden(opts sobek.Value) bool
	Fill(value string, opts sobek.Value)
	Focus(opts sobek.Value)
	GetAttribute(name string, opts sobek.Value) sobek.Value
	InnerHTML(opts sobek.Value) string
	InnerText(opts sobek.Value) string
	TextContent(opts sobek.Value) string
	InputValue(opts sobek.Value) string
	SelectOption(values sobek.Value, opts sobek.Value) []string
	Press(key string, opts sobek.Value)
	Type(text string, opts sobek.Value)
	Hover(opts sobek.Value)
	Tap(opts sobek.Value) (*sobek.Promise, error)
	DispatchEvent(typ string, eventInit, opts sobek.Value)
	WaitFor(opts sobek.Value)
}

// keyboardAPI is the interface of a keyboard input device.
// TODO: map this to page.GetKeyboard(). Currently, the common.Keyboard type
// mapping is not tested using this interface. We use the concrete type
// without testing its exported methods.
type keyboardAPI interface { //nolint: unused
	Down(key string)
	InsertText(char string)
	Press(key string, opts sobek.Value)
	Type(text string, opts sobek.Value)
	Up(key string)
}

// touchscreenAPI is the interface of a touchscreen.
type touchscreenAPI interface {
	Tap(x float64, y float64) *sobek.Promise
}

// mouseAPI is the interface of a mouse input device.
// TODO: map this to page.GetMouse(). Currently, the common.Mouse type
// mapping is not tested using this interface. We use the concrete type
// without testing its exported methods.
type mouseAPI interface { //nolint: unused
	Click(x float64, y float64, opts sobek.Value)
	DblClick(x float64, y float64, opts sobek.Value)
	Down(x float64, y float64, opts sobek.Value)
	Move(x float64, y float64, opts sobek.Value)
	Up(x float64, y float64, opts sobek.Value)
	// Wheel(opts sobek.Value)
}

// workerAPI is the interface of a web worker.
type workerAPI interface {
	URL() string
}
