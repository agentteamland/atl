//go:build unix

package dispatch

import "syscall"

// processAlive reports whether the process with the given pid is running.
// Signal 0 probes existence without delivering a signal: nil (or EPERM — the
// process exists but we can't signal it) means alive; ESRCH means gone.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
