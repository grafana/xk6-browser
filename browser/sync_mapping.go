package browser

import (
	"fmt"

	"github.com/dop251/goja"

	k6common "go.k6.io/k6/js/common"
)

// syncMapBrowserToGoja maps the browser API to the JS module as a
// synchronous version.
func syncMapBrowserToGoja(vu moduleVU) *goja.Object {
	var (
		rt  = vu.Runtime()
		obj = rt.NewObject()
	)
	for k, v := range syncMapBrowser(vu) {
		err := obj.Set(k, rt.ToValue(v))
		if err != nil {
			k6common.Throw(rt, fmt.Errorf("mapping: %w", err))
		}
	}

	return obj
}
