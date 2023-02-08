package tests

import (
	"fmt"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserContextAddCookies(t *testing.T) {
	opts := defaultLaunchOpts()
	tb := newTestBrowser(t, withFileServer(), opts)

	t.Run("happy_path", func(t *testing.T) {
		testCookieName := "test_cookie_name"
		testCookieValue := "test_cookie_value"

		bc := tb.NewContext(nil)
		cmd := fmt.Sprintf(`
			[
				{
					name: "%v",
					value: "%v",
					url: "%v"
				}
			];
		`, testCookieName, testCookieValue, tb.URL(""))
		cookies, err := tb.vu.Runtime().RunString(cmd)
		require.NoError(t, err)

		bc.AddCookies(cookies)

		p := bc.NewPage()
		_, err = p.Goto(
			tb.staticURL("add_cookies.html"),
			tb.toGojaValue(struct {
				WaitUntil string `js:"waitUntil"`
			}{
				WaitUntil: "load",
			}),
		)
		require.NoError(t, err)

		result := p.TextContent("#cookies", nil)
		assert.EqualValues(t, fmt.Sprintf("%v=%v", testCookieName, testCookieValue), result)
	})

	type errorTestCase struct {
		description string
		cookiesCmd  string
	}

	for _, testCase := range []errorTestCase{
		{
			description: "nil_cookies",
			cookiesCmd:  "",
		},
		{
			description: "goja_null_cookies",
			cookiesCmd:  "null;",
		},
		{
			description: "goja_undefined_cookies",
			cookiesCmd:  "undefined;",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			var cookies goja.Value
			if testCase.cookiesCmd != "" {
				var err error
				cookies, err = tb.vu.Runtime().RunString(testCase.cookiesCmd)
				require.NoError(t, err)
			}

			bc := tb.NewContext(nil)
			assert.Panics(t, func() { bc.AddCookies(cookies) })
		})
	}
}
