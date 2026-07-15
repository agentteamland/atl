//go:build unix

// Package detach forks a subcommand of the running atl binary as a fully
// detached background process — its own process group, no wait, stdio on the
// null device — so it outlives the short-lived hook process that spawned it and
// runs independently. The session-start auto-update paths (the binary
// self-update and the team asset-update) use it to do slow network work without
// blocking the SessionStart hook, which must never block or fail.
package detach

import (
	"os"
	"os/exec"
	"syscall"
)

// Spawn forks `atl <args...>` detached from the caller: a new process group
// (Setpgid) plus Release (never Wait), so a short-lived parent can exit while
// the child keeps running. It inherits the caller's cwd + env; nil stdio leaves
// the child's fds on the null device. Windows has its own impl in
// detach_windows.go; only genuinely exotic targets fall through to the stub.
func Spawn(args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release() // fully detach — never Wait
}
