package api

import (
	"github.com/dop251/goja"

	k6modules "go.k6.io/k6/js/modules"
)

// BrowserType is the public interface of a CDP browser client.
type BrowserType interface {
	Connect(vu k6modules.VU, wsEndpoint string, opts goja.Value) Browser
	ExecutablePath() string
	Launch(vu k6modules.VU, opts goja.Value) (_ Browser, browserProcessID int)
	LaunchPersistentContext(vu k6modules.VU, userDataDir string, opts goja.Value) Browser
	Name() string
}
