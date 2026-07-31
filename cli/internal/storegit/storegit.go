// Package storegit keeps a team's declared durable store under local version
// control, so a write that overwrites a value never destroys the previous one.
//
// The problem it solves is narrow and mechanical. A team may keep state outside
// the reflected `.claude` tree — profile-team's `~/.atl/profiles` is the first —
// and the write policy for such a store is usually last-write-wins. Without
// version control an overwrite is unrecoverable: the old value is simply gone,
// with no record that it ever existed. Measured on a real store: a field went
// from an inferred placeholder to a confirmed value and the previous text now
// exists nowhere.
//
// Two design constraints shape everything here:
//
//   - **Core stays team-agnostic.** This package is handed a path that a team
//     DECLARED (`capabilities.<name>.store`, recorded into the install manifest).
//     It never learns a team name, and it has no opinion about what the store
//     contains. A future team that declares a store gets the same treatment for
//     free.
//   - **Local only, never a remote.** A store holds the user's most sensitive
//     data. This package creates no remote, adds no remote, and pushes nothing.
//     Carrying a copy off the machine is a separate, explicitly-consented act.
//
// Everything fails silently. This runs from session-start, where a failure must
// never block the user, and where a missing `git` binary is a perfectly ordinary
// state rather than an error worth reporting.
package storegit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// commitMessage is the message every automatic snapshot commit carries. It is
// fixed and recognisable so the log reads as a machine record rather than an
// attempt to describe what changed — describing the change is the knowledge
// layer's job, not this one's.
const commitMessage = "chore(store): snapshot"

// Ensure brings one declared store under local version control and commits
// whatever is currently uncommitted.
//
// It is a no-op — not an error — when the directory does not exist. A store is
// created by its owning team on first use, so "absent" means the feature simply
// is not in use on this machine, and creating an empty versioned directory there
// would be litter that also misreports the feature as active.
//
// Returns true only when a commit was actually made, so the caller can stay
// silent on the (overwhelmingly common) no-change path.
func Ensure(dir string) bool {
	dir = expand(dir)
	if dir == "" {
		return false
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return false // not in use on this machine
	}
	if !hasGit() {
		return false
	}
	if !isRepoRoot(dir) {
		// A store nested INSIDE some other repo is left alone: initialising a
		// repo there would shadow the outer one, and committing would be writing
		// into a repo this package does not own.
		if insideOtherRepo(dir) {
			return false
		}
		if !run(dir, "init") {
			return false
		}
	}
	if clean(dir) {
		return false
	}
	if !run(dir, "add", "-A") {
		return false
	}
	return run(dir, "commit", "-m", commitMessage+" "+time.Now().Format("2006-01-02T15:04:05"))
}

// EnsureAll runs Ensure over every declared store, de-duplicated, and reports
// how many produced a commit. Two teams declaring the same path is legitimate
// (a provider and a consumer naming one store), and must not commit twice.
func EnsureAll(dirs []string) int {
	seen := map[string]bool{}
	n := 0
	for _, d := range dirs {
		key := expand(d)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if Ensure(key) {
			n++
		}
	}
	return n
}

// expand resolves a leading `~` against the user's home. Store paths are
// recorded verbatim from team.json, which writes them in tilde form.
func expand(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
	}
	return p
}

func hasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// isRepoRoot reports whether dir is itself the root of a git repo — not merely
// inside one.
func isRepoRoot(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && (fi.IsDir() || fi.Mode().IsRegular())
}

// insideOtherRepo reports whether dir sits within a git repo whose root is
// somewhere above it.
func insideOtherRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	top := strings.TrimSpace(string(out))
	if top == "" {
		return false
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	topAbs, err := filepath.Abs(top)
	if err != nil {
		return false
	}
	return topAbs != abs
}

func clean(dir string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return true // unreadable: treat as nothing to do rather than guess
	}
	return strings.TrimSpace(string(out)) == ""
}

// run executes a git command in dir, silently. Identity is supplied per-command
// so the commit works on a machine with no global git identity configured —
// without writing anything into the user's own git config.
func run(dir string, args ...string) bool {
	full := append([]string{
		"-c", "user.name=atl",
		"-c", "user.email=atl@localhost",
		"-c", "commit.gpgsign=false",
	}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	return cmd.Run() == nil
}
