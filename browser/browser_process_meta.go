package browser

import (
	"os"

	"github.com/grafana/xk6-browser/storage"
)

const (
	unknownProcessPid = -1
)

// processMeta handles the metadata associated with
// a browser process, especifically, the OS process handle
// and the associated browser data directory.
type processMeta interface {
	Pid() int
	Cleanup() error
}

// localProcessMeta holds the metadata for local
// browser process.
type localProcessMeta struct {
	process     *os.Process
	userDataDir *storage.Dir
}

// newLocalProcessMeta returns a new ProcessMeta
// for the given OS process and storage directory.
func newLocalProcessMeta(
	process *os.Process, userDataDir *storage.Dir,
) *localProcessMeta {
	return &localProcessMeta{
		process,
		userDataDir,
	}
}

// Pid returns the Pid for the local browser process.
func (l *localProcessMeta) Pid() int {
	return l.process.Pid
}

// Cleanup cleans the local user data directory associated
// with the local browser process.
func (l *localProcessMeta) Cleanup() error {
	return l.userDataDir.Cleanup() //nolint:wrapcheck
}

// remoteProcessMeta is a placeholder for a
// remote browser process metadata.
type remoteProcessMeta struct{}

// newRemoteProcessMeta returns a new ProcessMeta
// which acts as a placeholder for a remote browser process data.
func newRemoteProcessMeta() *remoteProcessMeta {
	return &remoteProcessMeta{}
}

// Pid returns -1 as the remote browser process is unknown.
func (r *remoteProcessMeta) Pid() int {
	return unknownProcessPid
}

// Cleanup does nothing and returns nil, as there is no
// access to the remote browser's user data directory.
func (r *remoteProcessMeta) Cleanup() error {
	// Nothing to do.
	return nil
}
