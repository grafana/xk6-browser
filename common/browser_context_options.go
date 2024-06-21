package common

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grafana/sobek"

	"github.com/grafana/xk6-browser/k6ext"
)

// BrowserContextOptions stores browser context options.
type BrowserContextOptions struct {
	AcceptDownloads   bool              `js:"acceptDownloads"`
	BypassCSP         bool              `js:"bypassCSP"`
	ColorScheme       ColorScheme       `js:"colorScheme"`
	DeviceScaleFactor float64           `js:"deviceScaleFactor"`
	ExtraHTTPHeaders  map[string]string `js:"extraHTTPHeaders"`
	Geolocation       *Geolocation      `js:"geolocation"`
	HasTouch          bool              `js:"hasTouch"`
	HttpCredentials   *Credentials      `js:"httpCredentials"`
	IgnoreHTTPSErrors bool              `js:"ignoreHTTPSErrors"`
	IsMobile          bool              `js:"isMobile"`
	JavaScriptEnabled bool              `js:"javaScriptEnabled"`
	Locale            string            `js:"locale"`
	Offline           bool              `js:"offline"`
	Permissions       []string          `js:"permissions"`
	ReducedMotion     ReducedMotion     `js:"reducedMotion"`
	Screen            *Screen           `js:"screen"`
	TimezoneID        string            `js:"timezoneID"`
	UserAgent         string            `js:"userAgent"`
	VideosPath        string            `js:"videosPath"`
	Viewport          *Viewport         `js:"viewport"`
}

// NewBrowserContextOptions creates a default set of browser context options.
func NewBrowserContextOptions() *BrowserContextOptions {
	return &BrowserContextOptions{
		ColorScheme:       ColorSchemeLight,
		DeviceScaleFactor: 1.0,
		ExtraHTTPHeaders:  make(map[string]string),
		JavaScriptEnabled: true,
		Locale:            DefaultLocale,
		Permissions:       []string{},
		ReducedMotion:     ReducedMotionNoPreference,
		Screen:            &Screen{Width: DefaultScreenWidth, Height: DefaultScreenHeight},
		Viewport:          &Viewport{Width: DefaultScreenWidth, Height: DefaultScreenHeight},
	}
}

// Validate validates the browser context options.
func (b *BrowserContextOptions) Validate() error {
	if err := b.Geolocation.Validate(); err != nil {
		return fmt.Errorf("validating geolocation option: %w", err)
	}

	return nil
}

// Geolocation represents a geolocation.
type Geolocation struct {
	Latitude  float64 `js:"latitude"`
	Longitude float64 `js:"longitude"`
	Accurracy float64 `js:"accurracy"`
}

// Validate validates the geolocation.
func (g *Geolocation) Validate() error {
	if g == nil {
		return nil // nothing to validate
	}

	if g.Accurracy < 0 {
		return fmt.Errorf(`invalid accuracy "%.2f": precondition 0 <= ACCURACY failed`, g.Accurracy)
	}
	if g.Latitude < -90 || g.Latitude > 90 {
		return fmt.Errorf(`invalid latitude "%.2f": precondition -90 <= LATITUDE <= 90 failed`, g.Latitude)
	}
	if g.Longitude < -180 || g.Longitude > 180 {
		return fmt.Errorf(`invalid longitude "%.2f": precondition -180 <= LONGITUDE <= 180 failed`, g.Longitude)
	}

	return nil
}

// WaitForEventOptions are the options used by the browserContext.waitForEvent API.
type WaitForEventOptions struct {
	Timeout     time.Duration
	PredicateFn sobek.Callable
}

// NewWaitForEventOptions created a new instance of WaitForEventOptions with a
// default timeout.
func NewWaitForEventOptions(defaultTimeout time.Duration) *WaitForEventOptions {
	return &WaitForEventOptions{
		Timeout: defaultTimeout,
	}
}

// Parse will parse the options or a callable predicate function. It can parse
// only a callable predicate function or an object which contains a callable
// predicate function and a timeout.
func (w *WaitForEventOptions) Parse(ctx context.Context, optsOrPredicate sobek.Value) error {
	if !sobekValueExists(optsOrPredicate) {
		return nil
	}

	var (
		isCallable bool
		rt         = k6ext.Runtime(ctx)
	)

	w.PredicateFn, isCallable = sobek.AssertFunction(optsOrPredicate)
	if isCallable {
		return nil
	}

	opts := optsOrPredicate.ToObject(rt)
	for _, k := range opts.Keys() {
		switch k {
		case "predicate":
			w.PredicateFn, isCallable = sobek.AssertFunction(opts.Get(k))
			if !isCallable {
				return errors.New("predicate function is not callable")
			}
		case "timeout": //nolint:goconst
			w.Timeout = time.Duration(opts.Get(k).ToInteger()) * time.Millisecond
		}
	}

	return nil
}

// GrantPermissionsOptions is used by BrowserContext.GrantPermissions.
type GrantPermissionsOptions struct {
	Origin string
}

// NewGrantPermissionsOptions returns a new GrantPermissionsOptions.
func NewGrantPermissionsOptions() *GrantPermissionsOptions {
	return &GrantPermissionsOptions{}
}

// Parse parses the options from opts if opts exists in the sobek runtime.
func (g *GrantPermissionsOptions) Parse(ctx context.Context, opts sobek.Value) {
	rt := k6ext.Runtime(ctx)

	if sobekValueExists(opts) {
		opts := opts.ToObject(rt)
		for _, k := range opts.Keys() {
			if k == "origin" {
				g.Origin = opts.Get(k).String()
				break
			}
		}
	}
}
