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

package api

import "github.com/dop251/goja"

// Browser is the public interface of a CDP browser.
type Browser interface {
	Close()
	Contexts() []BrowserContext
	IsConnected() bool
	NewContext(opts goja.Value) BrowserContext
	NewPage(opts goja.Value) Page
	On(string) *goja.Promise
	UserAgent() string
	Version() string
}
