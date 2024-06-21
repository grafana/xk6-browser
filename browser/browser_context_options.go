package browser

import (
	"context"
	"fmt"

	"github.com/grafana/sobek"

	"github.com/grafana/xk6-browser/common"
	"github.com/grafana/xk6-browser/k6ext"
)

// ParseBrowserContextOptions parses the browser context options.
func ParseBrowserContextOptions( //nolint:funlen,gocognit,cyclop
	ctx context.Context,
	opts sobek.Value,
) (*common.BrowserContextOptions, error) {
	popts := common.NewBrowserContextOptions()

	if !sobekValueExists(opts) {
		return popts, nil // return the default options
	}

	rt := k6ext.Runtime(ctx)
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
			geoloc, err := ParseGeolocation(ctx, o.Get(k).ToObject(rt))
			if err != nil {
				return nil, fmt.Errorf("parsing geolocation options: %w", err)
			}
			popts.Geolocation = geoloc
		case "hasTouch":
			popts.HasTouch = o.Get(k).ToBoolean()
		case "httpCredentials":
			credentials := common.NewCredentials()
			if err := credentials.Parse(ctx, o.Get(k).ToObject(rt)); err != nil {
				return nil, fmt.Errorf("parsing httpCredentials options: %w", err)
			}
			popts.HttpCredentials = credentials
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
			screen := &common.Screen{}
			if err := screen.Parse(ctx, o.Get(k).ToObject(rt)); err != nil {
				return nil, fmt.Errorf("parsing screen options: %w", err)
			}
			popts.Screen = screen
		case "timezoneID":
			popts.TimezoneID = o.Get(k).String()
		case "userAgent":
			popts.UserAgent = o.Get(k).String()
		case "viewport":
			viewport := &common.Viewport{}
			if err := viewport.Parse(ctx, o.Get(k).ToObject(rt)); err != nil {
				return nil, fmt.Errorf("parsing viewport options: %w", err)
			}
			popts.Viewport = viewport
		}
	}

	return popts, nil
}

// ParseGeolocation parses the geolocation.
func ParseGeolocation(ctx context.Context, opts sobek.Value) (*common.Geolocation, error) {
	var geoloc common.Geolocation

	if !sobekValueExists(opts) {
		return &geoloc, nil // return the default options
	}

	o := opts.ToObject(k6ext.Runtime(ctx))
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
