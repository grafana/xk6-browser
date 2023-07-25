package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/xk6-browser/api"
	"github.com/grafana/xk6-browser/common"
)

func TestURLSkipRequest(t *testing.T) {
	t.Parallel()

	tb := newTestBrowser(t, withLogCache())
	p := tb.NewPage(nil)

	_, err := p.Goto("data:text/html,hello", nil)
	require.NoError(t, err)
	tb.logCache.assertContains(t, "skipping request handling of data URL")

	_, err = p.Goto("blob:something", nil)
	require.NoError(t, err)
	tb.logCache.assertContains(t, "skipping request handling of blob URL")
}

func TestBasicAuth(t *testing.T) {
	const (
		validUser     = "validuser"
		validPassword = "validpass"
	)

	auth := func(tb testing.TB, user, pass string) api.Response {
		tb.Helper()

		browser := newTestBrowser(t, withHTTPServer())
		bc, err := browser.NewContext(
			browser.toGojaValue(struct {
				HttpCredentials *common.Credentials `js:"httpCredentials"` //nolint:revive
			}{
				HttpCredentials: &common.Credentials{
					Username: user,
					Password: pass,
				},
			}))
		require.NoError(t, err)
		p, err := bc.NewPage()
		require.NoError(t, err)

		opts := browser.toGojaValue(struct {
			WaitUntil string `js:"waitUntil"`
		}{
			WaitUntil: "load",
		})
		url := browser.url(fmt.Sprintf("/basic-auth/%s/%s", validUser, validPassword))
		res, err := p.Goto(url, opts)
		require.NoError(t, err)

		return res
	}

	t.Run("valid", func(t *testing.T) {
		resp := auth(t, validUser, validPassword)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, int(resp.Status()))
	})
	t.Run("invalid", func(t *testing.T) {
		resp := auth(t, "invalidUser", "invalidPassword")
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusUnauthorized, int(resp.Status()))
	})
}
