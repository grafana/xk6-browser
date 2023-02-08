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

	iID := GetIterationID(ctx)
	if _, ok := processRegister[iID]; !ok {
		processRegister[iID] = []int{}
	}
	processRegister[iID] = append(processRegister[iID], pid)
}

// ForceProcessShutdown should be called when
// xk6-browser is having to shutdown due to an
// internal error (and therefore a panic).
func ForceProcessShutdown(ctx context.Context) {
	processRegisterMu.Lock()
	defer processRegisterMu.Unlock()

	iID := GetIterationID(ctx)

	for _, pid := range processRegister[iID] {
		p, err := os.FindProcess(pid)
		if err != nil {
			// optimistically continue and don't kill the process
			continue
		}
		// no need to check the error for waiting the process to release
		// its resources or whether we could kill it as we're already
		// dying.
		_ = p.Release()
		_ = p.Kill()
	}
}
