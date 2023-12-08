// Package chromium is responsible for launching a Chrome browser process and managing its lifetime.
package chromium

import (
	"github.com/grafana/xk6-browser/browser"
)

// Browser is the public interface of a CDP browser.
type Browser struct {
	browser.Browser

	// TODO:
	// - add support for service workers
	// - add support for background pages
}
