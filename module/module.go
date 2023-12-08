// Package module provides an entry point to the browser module.
package module

import (
	"log"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	"sync"

	"github.com/dop251/goja"

	"github.com/grafana/xk6-browser/common"
	"github.com/grafana/xk6-browser/env"
	"github.com/grafana/xk6-browser/k6ext"

	k6modules "go.k6.io/k6/js/modules"
)

type (
	// Root is the global module instance that will create module
	// instances for each VU.
	Root struct {
		PidRegistry    *pidRegistry
		remoteRegistry *remoteRegistry
		initOnce       *sync.Once
	}

	// JS exposes the properties available to the JS script.
	JS struct {
		Browser         *goja.Object
		Devices         map[string]common.Device
		NetworkProfiles map[string]common.NetworkProfile `js:"networkProfiles"`
	}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		mod *JS
	}
)

var (
	_ k6modules.Module   = &Root{}
	_ k6modules.Instance = &ModuleInstance{}
)

// New returns a pointer to a new RootModule instance.
func New() *Root {
	return &Root{
		PidRegistry: &pidRegistry{},
		initOnce:    &sync.Once{},
	}
}

// NewModuleInstance implements the k6modules.Module interface to return
// a new instance for each VU.
func (m *Root) NewModuleInstance(vu k6modules.VU) k6modules.Instance {
	// initialization should be done once per module as it initializes
	// globally used values across the whole test run and not just the
	// current VU. Since initialization can fail with an error,
	// we've had to place it here so that if an error occurs a
	// panic can be initiated and safely handled by k6.
	m.initOnce.Do(func() {
		m.initialize(vu)
	})
	return &ModuleInstance{
		mod: &JS{
			Browser: mapBrowserToGoja(moduleVU{
				VU:                vu,
				pidRegistry:       m.PidRegistry,
				browserRegistry:   newBrowserRegistry(vu, m.remoteRegistry, m.PidRegistry),
				taskQueueRegistry: newTaskQueueRegistry(vu),
			}),
			Devices:         common.GetDevices(),
			NetworkProfiles: common.GetNetworkProfiles(),
		},
	}
}

// Exports returns the exports of the JS module so that it can be used in test
// scripts.
func (mi *ModuleInstance) Exports() k6modules.Exports {
	return k6modules.Exports{Default: mi.mod}
}

// initialize initializes the module instance with a new remote registry
// and debug server, etc.
func (m *Root) initialize(vu k6modules.VU) {
	var (
		err     error
		initEnv = vu.InitEnv()
	)
	m.remoteRegistry, err = newRemoteRegistry(initEnv.LookupEnv)
	if err != nil {
		k6ext.Abort(vu.Context(), "failed to create remote registry: %v", err)
	}
	if _, ok := initEnv.LookupEnv(env.EnableProfiling); ok {
		go startDebugServer()
	}
}

func startDebugServer() {
	log.Println("Starting http debug server", env.ProfilingServerAddr)
	log.Println(http.ListenAndServe(env.ProfilingServerAddr, nil)) //nolint:gosec
	// no linted because we don't need to set timeouts for the debug server.
}
