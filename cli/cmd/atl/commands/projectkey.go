package commands

import (
	"os"
	"path/filepath"
)

// projectKey resolves the directory that identifies "this project" for the
// global, path-keyed stores under ~/.atl — the learning queue and the sweep
// digest. Both partition by project, both live outside the project, and both
// are read back by a person or an agent standing in an arbitrary directory
// inside it. So the key has to be a property of the REPOSITORY, not of wherever
// the caller happened to be.
//
// Keying by the working directory failed differently in each store, and the two
// failures are worth keeping side by side because they are the same bug:
//
//   - The QUEUE lost data. `atl work dispatch` cuts one git worktree per unit
//     and deletes it on completion, so an autonomous worker's markers were
//     queued under a path that then ceased to exist — and since every read
//     surface is project-scoped, no drain could name them again. Measured
//     2026-08-08: 13 pending items in 6 vanished buckets, 7 of them real
//     learnings captured by delivery workers. The payload was intact the whole
//     time; only its address was gone, which is exactly why nothing ever
//     reported a problem.
//
//   - The DIGEST lost agreement. Its session-start signal is handed the root
//     resolved here while the `atl digest` command resolved its own with
//     os.Getwd(), so a session started in a subdirectory could be told "N
//     findings waiting" and then be shown none by the very command the signal
//     names. Nothing is destroyed, but a finding that cannot be reached is a
//     finding nobody decides on, which is the digest's whole purpose.
//
// Resolving through git's COMMON dir is what makes both work: in a worktree it
// points at the main repository's .git, so the worktree's items land in the
// parent's key and outlive the worktree. In an ordinary checkout — including a
// subdirectory of one — it collapses to the repository root, so every session
// in one repository shares one key instead of scattering by cwd.
//
// Fails toward the old behaviour: outside git, or if git cannot answer, the
// working directory is still a usable key. A store that partitions oddly is
// recoverable; one that refuses to open is not.
//
// The arms differ in one way that looks like an oversight and is left alone
// deliberately: the git arm evaluates symlinks, the fallback returns the working
// directory as os.Getwd gave it. Normalising the fallback too would be tidier
// and would RE-KEY every existing non-git project's bucket — the same silent
// loss this function was written to stop, arriving as a cleanup. A directory
// takes one arm or the other deterministically, so the asymmetry fragments
// nothing in practice; it is not worth a migration to remove.
func projectKey() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	common, err := gitRevParsePath(cwd, "--git-common-dir")
	if err != nil {
		return cwd, nil
	}
	root := filepath.Dir(common)
	if st, serr := os.Stat(root); serr != nil || !st.IsDir() {
		return cwd, nil
	}
	if resolved, rerr := filepath.EvalSymlinks(root); rerr == nil {
		root = resolved
	}
	return root, nil
}
