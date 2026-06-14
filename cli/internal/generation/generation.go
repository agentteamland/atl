// Package generation is the global-layer change counter behind the every-prompt
// fan-out (decision doc 5.6, three-speed cadence).
//
// A write to the user-global layer (a global install, a global team upgrade, a
// promote) bumps a counter at ~/.atl/generation. Each project records the last
// counter it fanned out from. The per-prompt tick fans out only when the two
// differ — so an unchanged global layer makes the per-prompt hook ~free (one
// small file read, no work), which is what keeps every-prompt affordable.
package generation

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func globalPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".atl", "generation"), nil
}

func seenPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".atl", "seen-generation")
}

func read(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	return n
}

func write(path string, n int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strconv.Itoa(n)+"\n"), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Current returns the global generation counter (0 if never bumped).
func Current() (int, error) {
	p, err := globalPath()
	if err != nil {
		return 0, err
	}
	return read(p), nil
}

// Bump increments the global generation counter — call after any write to the
// user-global layer.
func Bump() error {
	p, err := globalPath()
	if err != nil {
		return err
	}
	return write(p, read(p)+1)
}

// Changed reports whether the global layer has advanced past what projectRoot
// last fanned out from, and returns the current global generation.
func Changed(projectRoot string) (changed bool, current int, err error) {
	cur, err := Current()
	if err != nil {
		return false, 0, err
	}
	return cur != read(seenPath(projectRoot)), cur, nil
}

// MarkSeen records that projectRoot has fanned out from generation n.
func MarkSeen(projectRoot string, n int) error {
	return write(seenPath(projectRoot), n)
}
