package browserprocess

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/grafana/xk6-browser/log"
)

type processState struct {
	pid int
}

var (
	browserProcessRegister   = map[string]*processState{} //nolint:gochecknoglobals
	browserProcessRegisterMu = sync.Mutex{}               //nolint:gochecknoglobals
)

func register(ctx context.Context, logger *log.Logger, pid int) {
	browserProcessRegisterMu.Lock()
	defer browserProcessRegisterMu.Unlock()

	iID := GetIterationID(ctx)
	key := strconv.FormatInt(int64(pid), 10) + iID

	logger.Debugf("BrowserProcess:register", "registered BrowserProcess pid %d", pid)

	browserProcessRegister[key] = &processState{pid: pid}
}

// ForceProcessShutdown should be called when
// xk6-browser is having to shutdown due to an
// internal error (and therefore a panic).
func ForceProcessShutdown(ctx context.Context) {
	browserProcessRegisterMu.Lock()
	defer browserProcessRegisterMu.Unlock()

	iID := GetIterationID(ctx)

	for k, v := range browserProcessRegister {
		if iID != "" && !strings.Contains(k, iID) {
			continue
		}

		p, err := os.FindProcess(v.pid)
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
