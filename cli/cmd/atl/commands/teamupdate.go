package commands

import (
	"os"
	"path/filepath"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/generation"
	"github.com/agentteamland/atl/cli/internal/index"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/semver"
	"github.com/agentteamland/atl/cli/internal/source"
	"github.com/agentteamland/atl/cli/internal/teampkg"
)

// updateTeams checks each installed team against the resolved index and, when a
// newer version is published, re-fetches it and reflects the change under
// fan-out discipline (unmodified files updated, user edits preserved). Returns
// how many teams advanced. The network half of `atl update` (the local fan-out
// is fanOut).
func updateTeams(projectRoot string) (int, error) {
	ix, err := index.Resolve()
	if err != nil {
		return 0, err
	}
	advanced := 0
	for _, s := range []scope.Scope{scope.Project, scope.Global} {
		layer, err := scope.LayerDir(s, projectRoot)
		if err != nil {
			return advanced, err
		}
		claude, err := scope.ClaudeDir(s, projectRoot)
		if err != nil {
			return advanced, err
		}
		manifests, err := manifest.List(layer)
		if err != nil {
			return advanced, err
		}
		scopeAdvanced := 0
		for _, m := range manifests {
			entry, lookErr := ix.Lookup(m.Handle, m.Name)
			if lookErr != nil {
				continue // not in the index (e.g. a local-only team) — nothing to pull
			}
			if !semver.Less(m.Version, entry.Version) {
				continue // already current
			}
			if err := upgradeTeam(m, entry, layer, claude); err != nil {
				return advanced, err
			}
			advanced++
			scopeAdvanced++
		}
		if s == scope.Global && scopeAdvanced > 0 {
			_ = generation.Bump() // global team upgraded → other projects fan out
		}
	}
	return advanced, nil
}

// upgradeTeam fetches entry's source and reflects it onto installed team m,
// preserving user-modified files, then rewrites the manifest at the new version.
func upgradeTeam(m *manifest.Manifest, entry *index.Entry, layer, claude string) error {
	srcDir, err := source.Fetch(entry.Source.Repo, entry.Source.Subpath, entry.Source.Ref)
	if err != nil {
		return err
	}
	defer os.RemoveAll(srcDir)

	files, err := reflectWithFanout(srcDir, claude, m.Files)
	if err != nil {
		return err
	}
	m.Version = entry.Version
	m.Source = manifest.Source{Repo: entry.Source.Repo, Subpath: entry.Source.Subpath, Ref: entry.Source.Ref}
	m.Files = files
	return m.Write(layer)
}

// reflectWithFanout reflects a freshly fetched team (srcDir) onto claudeDir under
// fan-out discipline against the install baseline: unmodified files refresh to
// the new upstream, user-modified files are preserved, and a brand-new file is
// added only when no divergent local file already occupies its path (a colliding
// local file — the user's or the learning loop's — is preserved, never clobbered).
// Returns the next baseline (the new files map).
func reflectWithFanout(srcDir, claudeDir string, baseline map[string]string) (map[string]string, error) {
	next := map[string]string{}
	for _, ad := range teampkg.AssetDirs {
		root := filepath.Join(srcDir, ad)
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		walkErr := filepath.WalkDir(root, func(p string, d os.DirEntry, werr error) error {
			if werr != nil {
				return werr
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(srcDir, p)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			upstreamBytes, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			upstream := fanout.Hash(upstreamBytes)
			dst := filepath.Join(claudeDir, rel)
			local, err := fanout.HashFile(dst)
			if err != nil {
				return err
			}
			base, known := baseline[rel]
			switch {
			case !known:
				// A path the install did not previously own but the new version now
				// ships. Take it only when there is no divergent local file there —
				// otherwise it is the user's own (or the learning loop's) file at a
				// colliding path, and copying over it would silently destroy local
				// work. Mirror the promote fan-out's Decide("", local, upstream)
				// handling: an empty baseline + a divergent local resolves to Preserve.
				switch fanout.Decide("", local, upstream) {
				case fanout.Refresh: // no local file → add the new upstream file
					if err := teampkg.CopyFile(p, dst); err != nil {
						return err
					}
					next[rel] = upstream
				case fanout.UpToDate: // local already equals upstream → safe to own
					next[rel] = upstream
				}
				// Preserve → keep the user's file, and do not claim ownership of it.
			case fanout.Decide(base, local, upstream) == fanout.Refresh:
				if err := teampkg.CopyFile(p, dst); err != nil {
					return err
				}
				next[rel] = upstream
			case fanout.Decide(base, local, upstream) == fanout.UpToDate:
				next[rel] = upstream // already current → advance baseline
			default: // Preserve — user-modified; keep the original install baseline
				next[rel] = base
			}
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	// Files in the old baseline but absent from the new version are dropped from
	// the manifest (and left on disk — conservative; a future prune can remove).
	return next, nil
}
