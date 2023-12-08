//go:build !linux
// +build !linux

package browser

import "os/exec"

// killAfterParent kills the child process when the parent process dies.
func killAfterParent(cmd *exec.Cmd) {
}
