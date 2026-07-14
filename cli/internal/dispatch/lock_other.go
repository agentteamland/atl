//go:build !unix

package dispatch

import "os"

// processAlive reports whether a process with the given pid is running. Off Unix
// there is no signal-0 probe, so this uses os.FindProcess, which on Windows opens
// a handle via OpenProcess and errors when the pid is gone; the handle is released
// immediately. The delivery engine's dispatch lock only runs on the maintainer's
// Unix host — this exists so the binary cross-compiles for every goreleaser
// target (mirrors proc_other.go).
func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	_ = p.Release()
	return true
}
