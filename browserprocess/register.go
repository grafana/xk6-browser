package browserprocess

import (
	"os"
	"sync"

	"github.com/grafana/xk6-browser/log"
)

var (
	browserProcessRegister   = map[int]interface{}{}
	browserProcessRegisterMu = sync.Mutex{}
)

func register(logger *log.Logger, pid int) {
	browserProcessRegisterMu.Lock()
	defer browserProcessRegisterMu.Unlock()

	logger.Debugf("BrowserProcess:register", "registered BrowserProcess pid %d", pid)

	browserProcessRegister[pid] = nil
}

// ForceProcessShutdown should be called when
// xk6-browser is having to shutdown due to an
// internal error (and therefore a panic).
func ForceProcessShutdown() {
	browserProcessRegisterMu.Lock()
	defer browserProcessRegisterMu.Unlock()

	for pid := range browserProcessRegister {
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
