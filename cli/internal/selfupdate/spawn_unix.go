//go:build unix

package selfupdate

import (
	"os"
	"os/exec"
	"syscall"
)

// spawnDetachedUpgrade forks `atl upgrade` as a detached process — its own
// process group (Setpgid), stdio on the null device (nil fds), and no Wait — so
// it outlives the short-lived session-start process that spawned it and its
// download+swap runs independently. Unix only (the Setpgid split mirrors the
// package's own cross-platform discipline; a running Windows .exe can't
// self-replace, so AutoApply notifies there instead of calling this).
func spawnDetachedUpgrade() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "upgrade")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release() // fully detach — never Wait
}
