package storegit

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// BackupState is one declared store's off-machine status: whether its history
// has anywhere to go, and how much of it has not gone there yet.
type BackupState struct {
	// Dir is the vetted absolute path of the store.
	Dir string
	// HasRemote reports whether the store's repo has any remote configured at
	// all. False means its history exists nowhere but this disk.
	HasRemote bool
	// Unpushed counts commits reachable from HEAD but from no remote-tracking
	// branch. It is the store's own count, so it is meaningful only alongside
	// HasRemote: with no remote every commit is unpushed, which is true but not
	// the useful way to say it.
	Unpushed int
}

// Backups reports the off-machine status of every declared store this package
// owns.
//
// It answers the question local versioning structurally cannot. EnsureAll
// guarantees that an overwritten VALUE stays recoverable; it guarantees nothing
// about the disk surviving. Those are different jobs, and the second one has had
// no detector — which is how a store can accumulate months of irreplaceable
// content while every automatic mechanism reports success.
//
// Read-only and never fails. A path that does not vet, a store this package did
// not create, an absent directory, or any git error simply drops out of the
// result — the caller decides what to say about what came back, and says nothing
// about what did not.
func Backups(dirs []string) []BackupState {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	home = resolve(home)
	if home == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()

	seen := map[string]bool{}
	var out []BackupState
	for _, d := range dirs {
		path := resolve(expand(d, home))
		// cwd is deliberately passed empty. acceptable's "never an ancestor of the
		// working directory" clause exists to stop this package WRITING into a repo
		// the user is working in; reading is not that hazard, and applying the write
		// rule here would silently drop the report for a user who happens to be
		// standing inside their own store — a miss in the one direction that costs
		// something, since the whole point of this pass is to notice an absence.
		if path == "" || seen[path] || !acceptable(path, home, "") {
			continue
		}
		seen[path] = true

		if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
			continue
		}
		// Only stores this package created are reported on. A repo the user made
		// themselves is theirs, and its remotes are none of our business.
		if !owned(path) {
			continue
		}

		st := BackupState{Dir: path}
		remotes, ok := git(ctx, path, nil, "remote")
		if !ok {
			continue // not a usable repo — nothing honest to report
		}
		st.HasRemote = remotes != ""

		// Commits on HEAD that no remote-tracking branch contains. This form is
		// correct in both arrangements a store can be in: with no remote it counts
		// the whole history, and with a remote whose branch was never fetched it
		// still counts honestly, where `@{u}..HEAD` would simply error out.
		n, ok := git(ctx, path, nil, "rev-list", "--count", "HEAD", "--not", "--remotes")
		if !ok {
			continue // an empty repo with no commits yet — nothing to have lost
		}
		if v, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			st.Unpushed = v
		}
		out = append(out, st)
	}
	return out
}

// Display renders a store path for a user-facing message, abbreviating the home
// directory to "~".
//
// Never print the raw path: it carries the account name, which must not reach a
// terminal transcript, a screenshot, or a pasted bug report. The abbreviation is
// also what the repo's own personal-path scanner expects to see in any string a
// human might copy out.
func Display(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	// Both the raw and the resolved pair are tried, and neither alone is enough.
	// resolve() goes through EvalSymlinks, which fails on a path that does not
	// exist — so resolving first would hand back the raw path, leaking the account
	// name in exactly the case where nothing on disk can be inspected to catch it.
	// Resolving is still needed for the real one, where /var and a relocated home
	// are symlinks and the unresolved forms do not share a prefix.
	for _, pair := range [2][2]string{{home, path}, {resolve(home), resolve(path)}} {
		h, p := pair[0], pair[1]
		if h == "" || p == "" {
			continue
		}
		rel, err := filepath.Rel(h, p)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}
		return "~" + string(filepath.Separator) + rel
	}
	return path
}
