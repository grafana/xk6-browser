package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBrowserContextAddCookies(t *testing.T) {
	b := newTestBrowser(t)

	t.Run("happy_path", func(t *testing.T) {
		cookies, err := b.vu.Runtime().RunString(`
			[
				{
					name: "test_cookie_name",
					value: "test_cookie_value",
					url: "https://test.go"
				}
			];
		`)
		require.NoError(t, err)

		bc := b.NewContext(b.toGojaValue(nil))
		bc.AddCookies(cookies)

		// TODO: assert that the cookies are added once Cookies() is implemented
	})

	t.Run("nil_cookies", func(t *testing.T) {
		bc := b.NewContext(b.toGojaValue(nil))
		bc.AddCookies(nil)

		// TODO: assert that no cookies are added once Cookies() is implemented
	})

	t.Run("goja_null_cookies", func(t *testing.T) {
		cookies, err := b.vu.Runtime().RunString(`
			null;
		`)
		require.NoError(t, err)

		bc := b.NewContext(b.toGojaValue(nil))
		bc.AddCookies(cookies)

		// TODO: assert that no cookies are added once Cookies() is implemented
	})

	t.Run("goja_undefined_cookies", func(t *testing.T) {
		cookies, err := b.vu.Runtime().RunString(`
			undefined;
		`)
		require.NoError(t, err)

		bc := b.NewContext(b.toGojaValue(nil))
		bc.AddCookies(cookies)

		// TODO: assert that no cookies are added once Cookies() is implemented
	})
}
