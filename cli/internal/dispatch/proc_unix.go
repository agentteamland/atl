//go:build unix

package dispatch

import (
	"os"
	"os/exec"
	"syscall"
)

// configureProcAttr makes the worker its own process-group leader so the recovery
// ladder can signal the WHOLE tree (the worker AND any child it spawned — e.g. the
// Azure MCP server) as one group. Without this, a SIGKILL reaches only the worker
// PID and a surviving grandchild keeps the inherited stderr pipe open, which hangs
// the reaper's Wait() and (with WaitDelay as the backstop) leaks the grandchild.
func configureProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// signalProcess signals the worker's whole process group (pgid == the worker pid,
// set by configureProcAttr), so a SIGTERM/SIGKILL reaches its grandchildren too.
// It falls back to signalling just the process if the group signal fails (e.g. the
// group was never set) so behaviour degrades gracefully rather than dropping the
// signal.
func signalProcess(p *os.Process, sig os.Signal) error {
	s, ok := sig.(syscall.Signal)
	if !ok {
		return p.Signal(sig)
	}
	if err := syscall.Kill(-p.Pid, s); err != nil {
		return p.Signal(sig)
	}
	return nil
}
