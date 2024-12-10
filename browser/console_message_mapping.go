package browser

import (
	"github.com/grafana/xk6-browser/common"
)

// mapConsoleMessage to the JS module.
func mapConsoleMessage(vu moduleVU, event common.PageOnEvent) mapping {
	cm := event.ConsoleMessage

	return mapping{
		"args": func() []mapping {
			pauseOnBreakpoint(vu.breakpointRegistry, vu.Runtime())

			var (
				margs []mapping
				args  = cm.Args
			)
			for _, arg := range args {
				a := mapJSHandle(vu, arg)
				margs = append(margs, a)
			}

			return margs
		},
		// page(), text() and type() are defined as
		// functions in order to match Playwright's API
		"page": func() mapping {
			pauseOnBreakpoint(vu.breakpointRegistry, vu.Runtime())

			return mapPage(vu, cm.Page)
		},
		"text": func() string {
			pauseOnBreakpoint(vu.breakpointRegistry, vu.Runtime())

			return cm.Text
		},
		"type": func() string {
			pauseOnBreakpoint(vu.breakpointRegistry, vu.Runtime())

			return cm.Type
		},
	}
}
