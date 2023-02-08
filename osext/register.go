package osext

import (
	"context"
	"os"
	"sync"

	"github.com/grafana/xk6-browser/log"
)

var (
	processRegister   = map[string][]int{} //nolint:gochecknoglobals
	processRegisterMu = sync.Mutex{}       //nolint:gochecknoglobals
)

func register(ctx context.Context, logger *log.Logger, pid int) {
	processRegisterMu.Lock()
	defer processRegisterMu.Unlock()

	logger.Debugf("Process:register", "registered Process pid %d", pid)

	rID := GetRunID(ctx)
	if _, ok := processRegister[rID]; !ok {
		processRegister[rID] = []int{}
	}
	processRegister[rID] = append(processRegister[rID], pid)
}

// ForceProcessShutdown should be called when
// xk6-browser is having to shutdown due to an
// internal error (and therefore a panic).
func ForceProcessShutdown(ctx context.Context) {
	processRegisterMu.Lock()
	defer processRegisterMu.Unlock()

	rID := GetRunID(ctx)

	for _, pid := range processRegister[rID] {
		Kill(pid)
	}
}

// Kill will look for and kill the process with the
// given pid. This is only being exported to allow
// integration tests to override it so that in
// those tests the browser processes isn't killed
// which currently break many tests.
var Kill = func(pid int) { //nolint:gochecknoglobals
	p, err := os.FindProcess(pid)
	if err != nil {
		// optimistically continue and don't kill the process
		return
	}
	// no need to check the error since we're already dying.
	_ = p.Release()
	_ = p.Kill()
}
