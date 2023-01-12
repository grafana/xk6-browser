// Package browser provides an entry point to the browser extension.
package browser

import (
	"context"
	"errors"
	"os"

	"github.com/dop251/goja"

	"github.com/grafana/xk6-browser/common"
	"github.com/grafana/xk6-browser/k6ext"

	k6common "go.k6.io/k6/js/common"
	k6modules "go.k6.io/k6/js/modules"
)

const version = "0.7.0"

type (
	// RootModule is the global module instance that will create module
	// instances for each VU.
	RootModule struct{}

	// JSModule exposes the properties available to the JS script.
	JSModule struct {
		Chromium *goja.Object
		Devices  map[string]common.Device
		Version  string
	}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		mod *JSModule
	}
)

// moduleVU carries module specific VU information.
//
// Currently, it is used to carry the VU object to the
// inner objects and promises.
type moduleVU struct {
	k6modules.VU
}

func (vu moduleVU) Context() context.Context {
	// promises and inner objects need the VU object to be
	// able to use k6-core specific functionality.
	return k6ext.WithVU(vu.VU.Context(), vu.VU)
}

var (
	_ k6modules.Module   = &RootModule{}
	_ k6modules.Instance = &ModuleInstance{}
)

// New returns a pointer to a new RootModule instance.
func New() *RootModule {
	return &RootModule{}
}

// NewModuleInstance implements the k6modules.Module interface to return
// a new instance for each VU.
func (*RootModule) NewModuleInstance(vu k6modules.VU) k6modules.Instance {
	if _, ok := os.LookupEnv("K6_BROWSER_DISABLE_RUN"); ok {
		msg := "Disable run flag enabled, browser test run aborted. Please contact support."
		if m, ok := os.LookupEnv("K6_BROWSER_DISABLE_RUN_MSG"); ok {
			msg = m
		}

		k6common.Throw(vu.Runtime(), errors.New(msg))
	}

	// using our custom VU so that we can be ready for the future
	// changes to the k6-core VU code. And we can have a fine-grained
	// control over it, now and in the future.
	//
	// this puts the VU object in the context so that it can be
	// used by inner objects such as k6ext.Promise.
	vu = moduleVU{vu}

	return &ModuleInstance{
		mod: &JSModule{
			Chromium: mapBrowserToGoja(vu.Context(), vu),
			Devices:  common.GetDevices(),
			Version:  version,
		},
	}
}

// Exports returns the exports of the JS module so that it can be used in test
// scripts.
func (mi *ModuleInstance) Exports() k6modules.Exports {
	return k6modules.Exports{Default: mi.mod}
}
