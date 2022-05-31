package k6ext

import (
	"context"
	"fmt"
	"os"

	k6common "go.k6.io/k6/js/common"
)

// Panic will cause a panic with the given error which will shut
// the application down. Before panicking, it will find the
// browser process from the context and kill it if it still exists.
// TODO: test.
func Panic(ctx context.Context, format string, a ...interface{}) {
	rt := Runtime(ctx)
	if rt == nil {
		// this should never happen unless a programmer error
		panic("cannot get k6 runtime")
	}
	defer k6common.Throw(rt, fmt.Errorf(format, a...))

	pid := GetProcessID(ctx)
	if pid == 0 {
		// this should never happen unless a programmer error
		panic("cannot find process id")
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		// optimistically return and don't kill the process
		return
	}
	// no need to check the error for waiting the process to release
	// its resources or whether we could kill it as we're already
	// dying.
	_ = p.Release()
	_ = p.Kill()
}
