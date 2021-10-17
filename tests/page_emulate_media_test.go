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
	_ "embed"
	"reflect"
	"testing"

	"github.com/dop251/goja"
	"github.com/grafana/xk6-browser/testutils/browsertest"
	"github.com/stretchr/testify/assert"
)

type emulateMediaOpts struct {
	Media         string `js:"media"`
	ColorScheme   string `js:"colorScheme"`
	ReducedMotion string `js:"reducedMotion"`
}

func TestPageEmulateMedia(t *testing.T) {
	bt := browsertest.NewBrowserTest(t)
	defer bt.Browser.Close()

	t.Run("Page.emulateMedia", func(t *testing.T) {
		t.Run("should work", func(t *testing.T) { testPageEmulateMedia(t, bt) })
	})
}

func testPageEmulateMedia(t *testing.T, bt *browsertest.BrowserTest) {
	p := bt.Browser.NewPage(nil)
	defer p.Close(nil)

	p.EmulateMedia(bt.Runtime.ToValue(emulateMediaOpts{
		Media:         "print",
		ColorScheme:   "dark",
		ReducedMotion: "reduce",
	}))

	result := p.Evaluate(bt.Runtime.ToValue("() => matchMedia('print').matches")).(goja.Value)
	switch result.ExportType().Kind() {
	case reflect.Bool:
		assert.True(t, result.ToBoolean(), "expected media 'print'")
	default:
		t.Fail()
	}

	result = p.Evaluate(bt.Runtime.ToValue("() => matchMedia('(prefers-color-scheme: dark)').matches")).(goja.Value)
	switch result.ExportType().Kind() {
	case reflect.Bool:
		assert.True(t, result.ToBoolean(), "expected color scheme 'dark'")
	default:
		t.Fail()
	}

	result = p.Evaluate(bt.Runtime.ToValue("() => matchMedia('(prefers-reduced-motion: reduce)').matches")).(goja.Value)
	switch result.ExportType().Kind() {
	case reflect.Bool:
		assert.True(t, result.ToBoolean(), "expected reduced motion setting to be 'reduce'")
	default:
		t.Fail()
	}
}
