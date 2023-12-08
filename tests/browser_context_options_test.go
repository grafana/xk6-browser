package tests

import (
	_ "embed"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/xk6-browser/browser"
)

func TestContextOptionsDefaultValues(t *testing.T) {
	t.Parallel()

	opts := browser.NewContextOptions()
	assert.False(t, opts.AcceptDownloads)
	assert.False(t, opts.BypassCSP)
	assert.Equal(t, browser.ColorSchemeLight, opts.ColorScheme)
	assert.Equal(t, 1.0, opts.DeviceScaleFactor)
	assert.Empty(t, opts.ExtraHTTPHeaders)
	assert.Nil(t, opts.Geolocation)
	assert.False(t, opts.HasTouch)
	assert.Nil(t, opts.HTTPCredentials)
	assert.False(t, opts.IgnoreHTTPSErrors)
	assert.False(t, opts.IsMobile)
	assert.True(t, opts.JavaScriptEnabled)
	assert.Equal(t, browser.DefaultLocale, opts.Locale)
	assert.False(t, opts.Offline)
	assert.Empty(t, opts.Permissions)
	assert.Equal(t, browser.ReducedMotionNoPreference, opts.ReducedMotion)
	assert.Equal(t, &browser.Screen{Width: browser.DefaultScreenWidth, Height: browser.DefaultScreenHeight}, opts.Screen)
	assert.Equal(t, "", opts.TimezoneID)
	assert.Equal(t, "", opts.UserAgent)
	assert.Equal(t,
		&browser.Viewport{Width: browser.DefaultScreenWidth, Height: browser.DefaultScreenHeight},
		opts.Viewport,
	)
}

func TestContextOptionsDefaultViewport(t *testing.T) {
	t.Parallel()

	p := newTestBrowser(t).NewPage(nil)

	viewportSize := p.ViewportSize()
	assert.Equal(t, float64(browser.DefaultScreenWidth), viewportSize["width"])
	assert.Equal(t, float64(browser.DefaultScreenHeight), viewportSize["height"])
}

func TestContextOptionsSetViewport(t *testing.T) {
	t.Parallel()

	tb := newTestBrowser(t)
	bctx, err := tb.NewContext(tb.toGojaValue(struct {
		Viewport browser.Viewport `js:"viewport"`
	}{
		Viewport: browser.Viewport{
			Width:  800,
			Height: 600,
		},
	}))
	require.NoError(t, err)
	t.Cleanup(bctx.Close)
	p, err := bctx.NewPage()
	require.NoError(t, err)

	viewportSize := p.ViewportSize()
	assert.Equal(t, float64(800), viewportSize["width"])
	assert.Equal(t, float64(600), viewportSize["height"])
}

func TestContextOptionsExtraHTTPHeaders(t *testing.T) {
	t.Parallel()

	tb := newTestBrowser(t, withHTTPServer())
	bctx, err := tb.NewContext(tb.toGojaValue(struct {
		ExtraHTTPHeaders map[string]string `js:"extraHTTPHeaders"`
	}{
		ExtraHTTPHeaders: map[string]string{
			"Some-Header": "Some-Value",
		},
	}))
	require.NoError(t, err)
	t.Cleanup(bctx.Close)
	p, err := bctx.NewPage()
	require.NoError(t, err)

	err = tb.awaitWithTimeout(time.Second*5, func() error {
		resp, err := p.Goto(tb.url("/get"), nil)
		if err != nil {
			return err
		}
		require.NotNil(t, resp)
		var body struct{ Headers map[string][]string }
		require.NoError(t, json.Unmarshal(resp.Body().Bytes(), &body))
		h := body.Headers["Some-Header"]
		require.NotEmpty(t, h)
		assert.Equal(t, "Some-Value", h[0])
		return nil
	})
	require.NoError(t, err)
}
