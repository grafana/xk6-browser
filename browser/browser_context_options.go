package browser

import (
	"fmt"

	"github.com/grafana/sobek"

	"github.com/grafana/xk6-browser/common"
)

const (
	optionWidth  = "width"
	optionHeight = "height"
)

// ParseBrowserContextOptions parses the browser context options.
func ParseBrowserContextOptions( //nolint:funlen,gocognit,cyclop
	rt *sobek.Runtime,
	opts sobek.Value,
) (*common.BrowserContextOptions, error) {
	popts := common.NewBrowserContextOptions()

	if !sobekValueExists(opts) {
		return popts, nil // return the default options
	}

	o := opts.ToObject(rt)
	for _, k := range o.Keys() {
		switch k {
		case "acceptDownloads":
			popts.AcceptDownloads = o.Get(k).ToBoolean()
		case "bypassCSP":
			popts.BypassCSP = o.Get(k).ToBoolean()
		case "colorScheme":
			switch common.ColorScheme(o.Get(k).String()) { //nolint:exhaustive
			case "light":
				popts.ColorScheme = common.ColorSchemeLight
			case "dark":
				popts.ColorScheme = common.ColorSchemeDark
			default:
				popts.ColorScheme = common.ColorSchemeNoPreference
			}
		case "deviceScaleFactor":
			popts.DeviceScaleFactor = o.Get(k).ToFloat()
		case "extraHTTPHeaders":
			headers := o.Get(k).ToObject(rt)
			for _, k := range headers.Keys() {
				popts.ExtraHTTPHeaders[k] = headers.Get(k).String()
			}
		case "geolocation":
			geoloc, err := ParseGeolocation(rt, o.Get(k).ToObject(rt))
			if err != nil {
				return nil, fmt.Errorf("parsing geolocation options: %w", err)
			}
			popts.Geolocation = geoloc
		case "hasTouch":
			popts.HasTouch = o.Get(k).ToBoolean()
		case "httpCredentials":
			creds, err := ParseCredentials(rt, o.Get(k).ToObject(rt))
			if err != nil {
				return nil, fmt.Errorf("parsing httpCredentials options: %w", err)
			}
			popts.HttpCredentials = creds
		case "ignoreHTTPSErrors":
			popts.IgnoreHTTPSErrors = o.Get(k).ToBoolean()
		case "isMobile":
			popts.IsMobile = o.Get(k).ToBoolean()
		case "javaScriptEnabled":
			popts.JavaScriptEnabled = o.Get(k).ToBoolean()
		case "locale":
			popts.Locale = o.Get(k).String()
		case "offline":
			popts.Offline = o.Get(k).ToBoolean()
		case "permissions":
			if ps, ok := o.Get(k).Export().([]any); ok {
				for _, p := range ps {
					popts.Permissions = append(popts.Permissions, fmt.Sprintf("%v", p))
				}
			}
		case "reducedMotion":
			switch common.ReducedMotion(o.Get(k).String()) { //nolint:exhaustive
			case "reduce":
				popts.ReducedMotion = common.ReducedMotionReduce
			default:
				popts.ReducedMotion = common.ReducedMotionNoPreference
			}
		case "screen":
			screen, err := ParseScreen(rt, o.Get(k).ToObject(rt))
			if err != nil {
				return nil, fmt.Errorf("parsing screen options: %w", err)
			}
			popts.Screen = screen
		case "timezoneID":
			popts.TimezoneID = o.Get(k).String()
		case "userAgent":
			popts.UserAgent = o.Get(k).String()
		case "viewport":
			vp, err := ParseViewport(rt, o.Get(k).ToObject(rt))
			if err != nil {
				return nil, fmt.Errorf("parsing viewport options: %w", err)
			}
			popts.Viewport = vp
		}
	}

	return popts, nil
}

// ParseGeolocation parses the geolocation.
func ParseGeolocation(rt *sobek.Runtime, opts sobek.Value) (*common.Geolocation, error) {
	var geoloc common.Geolocation

	if !sobekValueExists(opts) {
		return &geoloc, nil // return the default options
	}

	o := opts.ToObject(rt)
	for _, k := range o.Keys() {
		switch k {
		case "accuracy":
			geoloc.Accurracy = o.Get(k).ToFloat()
		case "latitude":
			geoloc.Latitude = o.Get(k).ToFloat()
		case "longitude":
			geoloc.Longitude = o.Get(k).ToFloat()
		}
	}

	return &geoloc, nil
}

// ParseCredentials parses the credentials.
func ParseCredentials(rt *sobek.Runtime, opts sobek.Value) (*common.Credentials, error) {
	var creds common.Credentials

	if !sobekValueExists(opts) {
		return &creds, nil // return the default options
	}

	o := opts.ToObject(rt)
	for _, k := range o.Keys() {
		switch k {
		case "username":
			creds.Username = o.Get(k).String()
		case "password":
			creds.Password = o.Get(k).String()
		}
	}

	return &creds, nil
}

// ParseScreen parses the screen options.
func ParseScreen(rt *sobek.Runtime, opts sobek.Value) (*common.Screen, error) {
	var screen common.Screen

	if !sobekValueExists(opts) {
		return &screen, nil // return the default options
	}

	o := opts.ToObject(rt)
	for _, k := range o.Keys() {
		switch k {
		case optionWidth:
			screen.Width = o.Get(k).ToInteger()
		case optionHeight:
			screen.Height = o.Get(k).ToInteger()
		}
	}

	return &screen, nil
}

// ParseViewport parses the viewport options.
func ParseViewport(rt *sobek.Runtime, opts sobek.Value) (*common.Viewport, error) {
	var viewport common.Viewport

	if !sobekValueExists(opts) {
		return &viewport, nil // return the default options
	}

	o := opts.ToObject(rt)
	for _, k := range o.Keys() {
		switch k {
		case optionWidth:
			viewport.Width = o.Get(k).ToInteger()
		case optionHeight:
			viewport.Height = o.Get(k).ToInteger()
		}
	}

	return &viewport, nil
}
