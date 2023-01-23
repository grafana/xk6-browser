package tests

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserContextAddInitScript(t *testing.T) {
	t.Parallel()

	t.Run("string_script_on_new_page", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withFileServer())
		bctx := tb.NewContext(nil)
		t.Cleanup(bctx.Close)

		rt := tb.vu.Runtime()
		body := "<h1>AddInitScript</h1>"
		script := fmt.Sprintf(
			`(function () { document.open(); document.write("%s"); document.close(); }());`,
			body,
		)
		_, err := rt.RunString(fmt.Sprintf("const s = '%s';", script))
		require.NoError(t, err)

		err = bctx.AddInitScript(rt.Get("s"), nil)
		require.NoError(t, err)
		p := bctx.NewPage()

		resp, err := p.Goto(tb.staticURL("empty.html"), nil)
		require.NotNil(t, resp)
		require.NoError(t, err)

		assert.Equal(t, wrapHTMLBody(body), p.Content())
	})

	t.Run("string_script_on_existing_page", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withFileServer())
		bctx := tb.NewContext(nil)
		t.Cleanup(bctx.Close)

		rt := tb.vu.Runtime()
		body := "<h1>AddInitScript</h1>"
		script := fmt.Sprintf(
			`(function () { document.open(); document.write("%s"); document.close(); }());`,
			body,
		)
		_, err := rt.RunString(fmt.Sprintf("const s = '%s';", script))
		require.NoError(t, err)

		p := bctx.NewPage()
		err = bctx.AddInitScript(rt.Get("s"), nil)
		require.NoError(t, err)

		resp, err := p.Goto(tb.staticURL("empty.html"), nil)
		require.NotNil(t, resp)
		require.NoError(t, err)

		assert.Equal(t, wrapHTMLBody(body), p.Content())
	})

	t.Run("function_script_without_args", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withFileServer())
		bctx := tb.NewContext(nil)
		t.Cleanup(bctx.Close)

		rt := tb.vu.Runtime()
		body := "<h1>AddInitScript</h1>"
		_, err := rt.RunString(fmt.Sprintf(`function f() {
			document.open();
			document.write("%s");
			document.close();
		}`, body))
		require.NoError(t, err)

		p := bctx.NewPage()
		err = bctx.AddInitScript(rt.Get("f"), nil)
		require.NoError(t, err)

		resp, err := p.Goto(tb.staticURL("empty.html"), nil)
		require.NotNil(t, resp)
		require.NoError(t, err)

		assert.Equal(t, wrapHTMLBody(body), p.Content())
	})

	t.Run("function_script_with_arg", func(t *testing.T) {
		t.Parallel()

		bodyContentScript := `function f(content) {
			document.open();
			document.write(content);
			document.close();
		}`
		bodyObjectScript := `function f(obj) {
			document.open();
			document.write(obj.content);
			document.close();
		}`

		tests := []struct {
			name      string
			script    string
			argScript string
			expBody   string
		}{
			{
				name:      "string_arg",
				script:    bodyContentScript,
				argScript: `const arg = '<h1>AddInitScript</h1>';`,
				expBody:   "<h1>AddInitScript</h1>",
			},
			{
				name:      "num_arg",
				script:    bodyContentScript,
				argScript: `const arg = 1;`,
				expBody:   "1",
			},
			{
				name:      "bool_arg",
				script:    bodyContentScript,
				argScript: `const arg = true;`,
				expBody:   "true",
			},
			{
				name:      "obj_arg",
				script:    bodyObjectScript,
				argScript: `const arg = {content: "obj arg"};`,
				expBody:   "obj arg",
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				tb := newTestBrowser(t, withFileServer())
				bctx := tb.NewContext(nil)
				t.Cleanup(bctx.Close)

				rt := tb.vu.Runtime()
				_, err := rt.RunString(tt.argScript)
				require.NoError(t, err)
				_, err = rt.RunString(tt.script)
				require.NoError(t, err)

				p := bctx.NewPage()
				err = bctx.AddInitScript(rt.Get("f"), rt.Get("arg"))
				require.NoError(t, err)

				resp, err := p.Goto(tb.staticURL("empty.html"), nil)
				require.NotNil(t, resp)
				require.NoError(t, err)

				assert.Equal(t, wrapHTMLBody(tt.expBody), p.Content())
			})
		}
	})

	t.Run("object_script_with_content", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withFileServer())
		bctx := tb.NewContext(nil)
		t.Cleanup(bctx.Close)

		rt := tb.vu.Runtime()
		body := "<h1>AddInitScript</h1>"
		script := fmt.Sprintf(
			`(function () { document.open(); document.write('%s'); document.close(); }());`,
			body,
		)
		_, err := rt.RunString(fmt.Sprintf(
			`const obj = {content: "%s"};`,
			script,
		))
		require.NoError(t, err)

		p := bctx.NewPage()
		err = bctx.AddInitScript(rt.Get("obj"), nil)
		require.NoError(t, err)

		resp, err := p.Goto(tb.staticURL("empty.html"), nil)
		require.NotNil(t, resp)
		require.NoError(t, err)

		assert.Equal(t, wrapHTMLBody(body), p.Content())
	})

	t.Run("object_script_with_path_rel", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withFileServer())
		bctx := tb.NewContext(nil)
		t.Cleanup(bctx.Close)

		rt := tb.vu.Runtime()
		path := "static/init_script.js"
		_, err := rt.RunString(fmt.Sprintf(
			`const obj = {path: "%s"};`,
			path,
		))
		require.NoError(t, err)

		p := bctx.NewPage()
		err = bctx.AddInitScript(rt.Get("obj"), nil)
		require.NoError(t, err)

		resp, err := p.Goto(tb.staticURL("empty.html"), nil)
		require.NotNil(t, resp)
		require.NoError(t, err)

		assert.Equal(t, wrapHTMLBody("<h1>AddInitScript</h1>"), p.Content())
	})

	t.Run("object_script_with_path_abs", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withFileServer())
		bctx := tb.NewContext(nil)
		t.Cleanup(bctx.Close)

		rt := tb.vu.Runtime()
		path, err := filepath.Abs("static/init_script.js")
		if err != nil {
			t.Fatal("error building test init script abosulte path")
		}
		_, err = rt.RunString(fmt.Sprintf(
			`const obj = {path: "%s"};`,
			path,
		))
		require.NoError(t, err)

		p := bctx.NewPage()
		err = bctx.AddInitScript(rt.Get("obj"), nil)
		require.NoError(t, err)

		resp, err := p.Goto(tb.staticURL("empty.html"), nil)
		require.NotNil(t, resp)
		require.NoError(t, err)

		assert.Equal(t, wrapHTMLBody("<h1>AddInitScript</h1>"), p.Content())
	})

	t.Run("error_invalid_script", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withFileServer())
		bctx := tb.NewContext(nil)
		t.Cleanup(bctx.Close)

		err := bctx.AddInitScript(nil, nil)
		require.ErrorContains(t, err, "invalid")
	})

	t.Run("error_script_not_callable_with_arg", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withFileServer())
		bctx := tb.NewContext(nil)
		t.Cleanup(bctx.Close)

		rt := tb.vu.Runtime()
		script := `(function () { document.open(); document.write("test"); document.close(); }());`
		_, err := rt.RunString(fmt.Sprintf("const s = '%s';", script))
		require.NoError(t, err)
		_, err = rt.RunString(`const arg = 1;`)
		require.NoError(t, err)

		err = bctx.AddInitScript(rt.Get("s"), rt.Get("arg"))
		require.ErrorContains(t, err, "script as function")
	})

	t.Run("error_script_object_no_content_nor_path", func(t *testing.T) {
		t.Parallel()

		tb := newTestBrowser(t, withFileServer())
		bctx := tb.NewContext(nil)
		t.Cleanup(bctx.Close)

		rt := tb.vu.Runtime()
		_, err := rt.RunString(`const obj = {prop: "test"};`)
		require.NoError(t, err)

		err = bctx.AddInitScript(rt.Get("obj"), nil)
		require.ErrorContains(t, err, "content or path")
	})
}

func wrapHTMLBody(body string) string {
	return fmt.Sprintf("<html><head></head><body>%s</body></html>", body)
}
