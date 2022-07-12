/*
 *
 * xk6-browser - a browser automation extension for k6
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package tests

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserNewPage(t *testing.T) {
	b := newTestBrowser(t)
	p := b.NewPage(nil)
	l := len(b.Contexts())
	assert.Equal(t, 1, l, "expected there to be 1 browser context, but found %d", l)

	p2 := b.NewPage(nil)
	l = len(b.Contexts())
	assert.Equal(t, 2, l, "expected there to be 2 browser context, but found %d", l)

	p.Close(nil)
	l = len(b.Contexts())
	assert.Equal(t, 1, l, "expected there to be 1 browser context after first page close, but found %d", l)
	p2.Close(nil)
	l = len(b.Contexts())
	assert.Equal(t, 0, l, "expected there to be 0 browser context after second page close, but found %d", l)
}

func TestTmpDirCleanup(t *testing.T) {
	tmpDirPath := "./"

	err := os.Setenv("TMPDIR", tmpDirPath)
	assert.NoError(t, err)
	defer func() {
		err = os.Unsetenv("TMPDIR")
		assert.NoError(t, err)
	}()

	b := newTestBrowser(t, withSkipClose())
	p := b.NewPage(nil)
	p.Close(nil)

	matches, err := filepath.Glob(tmpDirPath + "xk6-browser-data-*")
	assert.NoError(t, err)
	assert.NotEmpty(t, matches, "a dir should exist that matches the pattern `xk6-browser-data-*`")

	b.Close()

	matches, err = filepath.Glob(tmpDirPath + "xk6-browser-data-*")
	assert.NoError(t, err)
	assert.Empty(t, matches, "a dir shouldn't exist which matches the pattern `xk6-browser-data-*`")
}

func TestBrowserOn(t *testing.T) {
	t.Parallel()

	script := `b.on('%s').then(log).catch(log);`

	t.Run("err_wrong_event", func(t *testing.T) {
		t.Parallel()

		b := newTestBrowser(t)
		require.NoError(t, b.vu.Runtime().Set("b", b.Browser))

		err := b.vu.RunLoop(func() error {
			_, err := b.runString(script, "wrongevent")
			return err
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, `unknown browser event: "wrongevent", must be "disconnected"`)
	})

	t.Run("ok_promise_resolved", func(t *testing.T) {
		t.Parallel()

		var (
			b   = newTestBrowser(t, withSkipClose())
			rt  = b.vu.Runtime()
			log []string
		)

		require.NoError(t, rt.Set("b", b.Browser))
		require.NoError(t, rt.Set("log", func(s string) { log = append(log, s) }))

		err := b.vu.RunLoop(func() error {
			time.AfterFunc(100*time.Millisecond, b.Browser.Close)
			_, err := b.runString(script, "disconnected")
			return err
		})
		require.NoError(t, err)
		assert.Contains(t, log, "true")
	})

	t.Run("ok_promise_rejected", func(t *testing.T) {
		t.Parallel()

		var (
			ctx, cancel = context.WithCancel(context.Background())
			b           = newTestBrowser(t, ctx)
			rt          = b.vu.Runtime()
			log         []string
		)

		require.NoError(t, rt.Set("b", b.Browser))
		require.NoError(t, rt.Set("log", func(s string) { log = append(log, s) }))

		err := b.vu.RunLoop(func() error {
			time.AfterFunc(100*time.Millisecond, cancel)
			_, err := b.runString(script, "disconnected")
			return err
		})
		require.NoError(t, err)
		assert.Contains(t, log, "browser.on promise rejected: context canceled")
	})
}

// This only works for Chrome!
func TestBrowserVersion(t *testing.T) {
	const re = `^\d+\.\d+\.\d+\.\d+$`
	r, _ := regexp.Compile(re)
	ver := newTestBrowser(t).Version()
	assert.Regexp(t, r, ver, "expected browser version to match regex %q, but found %q", re, ver)
}

// This only works for Chrome!
// TODO: Improve this test, see:
// https://github.com/grafana/xk6-browser/pull/51#discussion_r742696736
func TestBrowserUserAgent(t *testing.T) {
	b := newTestBrowser(t)

	// testBrowserVersion() tests the version already
	// just look for "Headless" in UserAgent
	ua := b.UserAgent()
	if prefix := "Mozilla/5.0"; !strings.HasPrefix(ua, prefix) {
		t.Errorf("UserAgent should start with %q, but got: %q", prefix, ua)
	}
	assert.Contains(t, ua, "Headless")
}
