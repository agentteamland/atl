package commands

import (
	"fmt"
	"os"
	"path/filepath"
)

// repoRootEnv names the repository root explicitly. It exists for the layout an
// upward walk cannot reach: a hub checkout that clones the monorepo as a CHILD
// (<hub>/repos/atl), where the marker sits BELOW the working directory. That is
// the maintainer's documented session directory, and it is why `atl skills
// check`, `atl rules scan` and `atl docs check` all reported "nothing to check"
// and exited 0 there for 18 days while their cursors stood still (#483).
const repoRootEnv = "ATL_REPO_ROOT"

// findRepoRoot returns the directory holding marker (a relative path such as
// "core/skills"): $ATL_REPO_ROOT if set, else the nearest ancestor of the
// working directory that holds it.
//
// A downward search was implemented and REMOVED, and the reason is worth
// keeping because it is not obvious: in the hub layout the marker matches more
// than one directory, and the extra matches are ARCHIVED v1 clones carrying the
// same shape. `core/skills` resolves to both repos/atl and repos/core; the docs
// marker resolves under repos/docs — whose site is the retired v1 build. Which
// one a probe returns is decided by readdir order, so the gates would have run
// confidently against a dead repo and reported drift that is real about content
// nobody ships. Measured: from the hub, a probing build failed `atl docs check`
// on "command `atl decided` has no docs page" while decided.md exists and the
// same check is green inside the monorepo.
//
// That is strictly worse than the silent skip it was meant to fix — a wrong
// answer instead of no answer — so the root is named, never guessed.
func findRepoRoot(marker string) (string, error) {
	if v := os.Getenv(repoRootEnv); v != "" {
		if isDir(filepath.Join(v, marker)) {
			return v, nil
		}
		return "", fmt.Errorf("%s=%s does not hold %s", repoRootEnv, v, marker)
	}

	start, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := start; ; {
		if isDir(filepath.Join(dir, marker)) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found from %s upward (set %s to name the repository root)",
				marker, start, repoRootEnv)
		}
		dir = parent
	}
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
