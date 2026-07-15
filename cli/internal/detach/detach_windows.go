//go:build windows

package detach

import (
	"os"
	"os/exec"
	"syscall"
)

// detachedNoWindow = DETACHED_PROCESS | CREATE_NO_WINDOW — stable Win32 creation
// flags for a background process with no console. Using literals avoids a
// golang.org/x/sys dependency for two constants.
const detachedNoWindow = 0x00000008 | 0x08000000

// Spawn forks `atl <args...>` as a detached, window-less background process, so
// the short-lived parent (the session-start hook) can exit while the child keeps
// running. Safe on Windows because the only caller here is `atl update`, which
// merely copies team files — it is not a running-.exe self-replacement like
// `atl upgrade` (which stays notify-only on Windows, guarded in selfupdate).
func Spawn(args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedNoWindow}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release() // fully detach — never Wait
}
