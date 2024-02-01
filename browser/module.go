// Package browser is the browser module's entry point, and
// initializer of various global types, and a translation layer
// between Goja and the internal business logic.
//
// It initializes and drives the downstream components by passing
// the necessary concrete dependencies.
package browser

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	"sync"

	"github.com/dop251/goja"

	"github.com/grafana/xk6-browser/common"
	"github.com/grafana/xk6-browser/env"
	"github.com/grafana/xk6-browser/k6ext"
	"github.com/grafana/xk6-browser/storage"

	k6modules "go.k6.io/k6/js/modules"
)

type (
	// RootModule is the global module instance that will create module
	// instances for each VU.
	RootModule struct {
		PidRegistry    *pidRegistry
		remoteRegistry *remoteRegistry
		initOnce       *sync.Once
		tracesMetadata map[string]string
		filePersister  storage.FilePersister
	}

	// JSModule exposes the properties available to the JS script.
	JSModule struct {
		Browser         *goja.Object
		Devices         map[string]common.Device
		NetworkProfiles map[string]common.NetworkProfile `js:"networkProfiles"`
	}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		mod *JSModule
	}
)

var (
	_ k6modules.Module   = &RootModule{}
	_ k6modules.Instance = &ModuleInstance{}
)

// New returns a pointer to a new RootModule instance.
func New() *RootModule {
	return &RootModule{
		PidRegistry: &pidRegistry{},
		initOnce:    &sync.Once{},
	}
}

// NewModuleInstance implements the k6modules.Module interface to return
// a new instance for each VU.
func (m *RootModule) NewModuleInstance(vu k6modules.VU) k6modules.Instance {
	// initialization should be done once per module as it initializes
	// globally used values across the whole test run and not just the
	// current VU. Since initialization can fail with an error,
	// we've had to place it here so that if an error occurs a
	// panic can be initiated and safely handled by k6.
	m.initOnce.Do(func() {
		m.initialize(vu)
	})
	return &ModuleInstance{
		mod: &JSModule{
			Browser: mapBrowserToGoja(moduleVU{
				VU:          vu,
				pidRegistry: m.PidRegistry,
				browserRegistry: newBrowserRegistry(
					context.Background(),
					vu,
					m.remoteRegistry,
					m.PidRegistry,
					m.tracesMetadata,
				),
				taskQueueRegistry:  newTaskQueueRegistry(vu),
				LocalFilePersister: &storage.LocalFilePersister{},
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
func (m *RootModule) initialize(vu k6modules.VU) {
	var (
		err     error
		initEnv = vu.InitEnv()
	)
	m.remoteRegistry, err = newRemoteRegistry(initEnv.LookupEnv)
	if err != nil {
		k6ext.Abort(vu.Context(), "failed to create remote registry: %v", err)
	}
	m.tracesMetadata, err = parseTracesMetadata(initEnv.LookupEnv)
	if err != nil {
		k6ext.Abort(vu.Context(), "parsing browser traces metadata: %v", err)
	}
	if _, ok := initEnv.LookupEnv(env.EnableProfiling); ok {
		go startDebugServer()
	}
	m.filePersister, err = storage.NewFilePersister(initEnv.LookupEnv)
	if err != nil {
		k6ext.Abort(vu.Context(), "failed to create file persister: %v", err)
	}
}

func startDebugServer() {
	log.Println("Starting http debug server", env.ProfilingServerAddr)
	log.Println(http.ListenAndServe(env.ProfilingServerAddr, nil)) //nolint:gosec
	// no linted because we don't need to set timeouts for the debug server.
}
