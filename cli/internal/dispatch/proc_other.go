//go:build !unix

package dispatch

import (
	"os"
	"os/exec"
)

// configureProcAttr is a no-op off Unix: process groups are a POSIX concept, and
// the delivery engine's worker lifecycle runs on the maintainer's Unix host. The
// binary must still compile for every goreleaser target, hence this stub.
func configureProcAttr(cmd *exec.Cmd) {}

// signalProcess signals just the process off Unix (no process-group semantics).
func signalProcess(p *os.Process, sig os.Signal) error {
	return p.Signal(sig)
}
