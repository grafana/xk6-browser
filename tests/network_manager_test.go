package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/xk6-browser/api"
	"github.com/grafana/xk6-browser/common"

	k6lib "go.k6.io/k6/lib"
	k6types "go.k6.io/k6/lib/types"
)

func TestURLSkipRequest(t *testing.T) {
	t.Parallel()

	tb := newTestBrowser(t, withLogCache())
	p := tb.NewPage(nil)

	require.NoError(t, tb.await(func() error {
		tb.promiseThen(p.Goto("data:text/html,hello", nil),
			func() *goja.Promise {
				assert.True(t, tb.logCache.contains("skipping request handling of data URL"))
				return p.Goto("blob:something", nil)
			},
		).then(func() {
			assert.True(t, tb.logCache.contains("skipping request handling of blob URL"))
		})

		return nil
	}))
}

func TestBlockHostnames(t *testing.T) {
	tb := newTestBrowser(t, withHTTPServer(), withLogCache())

	blocked, err := k6types.NewNullHostnameTrie([]string{"*.test"})
	require.NoError(t, err)
	tb.vu.State().Options.BlockedHostnames = blocked

	p := tb.NewPage(nil)

	require.NoError(t, tb.await(func() error {
		tb.promiseThen(
			p.Goto("http://host.test/", nil),
			func(res api.Response) *goja.Promise {
				require.Nil(t, res)
				require.True(t, tb.logCache.contains("was interrupted: hostname host.test is in a blocked pattern"))
				return p.Goto(tb.URL("/get"), nil)
			},
		).then(func(res api.Response) {
			assert.NotNil(t, res)
		})

		return nil
	}))
}

func TestBlockIPs(t *testing.T) {
	tb := newTestBrowser(t, withHTTPServer(), withLogCache())

	ipnet, err := k6lib.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)
	tb.vu.State().Options.BlacklistIPs = []*k6lib.IPNet{ipnet}

	p := tb.NewPage(nil)
	require.NoError(t, tb.await(func() error {
		tb.promiseThen(
			p.Goto("http://10.0.0.1:8000/", nil),
			func(res api.Response) *goja.Promise {
				require.Nil(t, res)
				assert.True(t, tb.logCache.contains(
					`was interrupted: IP 10.0.0.1 is in a blacklisted range "10.0.0.0/8"`))
				return p.Goto(tb.URL("/get"), nil)
			},
		).then(func(res api.Response) {
			// Ensure other requests go through
			assert.NotNil(t, res)
		})

		return nil
	}))
}

func TestBasicAuth(t *testing.T) {
	const (
		validUser     = "validuser"
		validPassword = "validpass"
	)

	browser := newTestBrowser(t, withHTTPServer())

	auth := func(tb testing.TB, user, pass string) api.Response {
		tb.Helper()

		p := browser.NewContext(
			browser.toGojaValue(struct {
				HttpCredentials *common.Credentials `js:"httpCredentials"` //nolint:revive
			}{
				HttpCredentials: &common.Credentials{
					Username: user,
					Password: pass,
				},
			})).
			NewPage()

		var res api.Response
		require.NoError(t, browser.await(func() error {
			browser.promiseThen(
				p.Goto(
					browser.URL(fmt.Sprintf("/basic-auth/%s/%s", validUser, validPassword)),
					browser.toGojaValue(struct {
						WaitUntil string `js:"waitUntil"`
					}{
						WaitUntil: "load",
					}),
				),
				func(resp api.Response) {
					res = resp
				})

			return nil
		}))

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
