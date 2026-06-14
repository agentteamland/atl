// Package publish computes what a team's published version is missing relative
// to the user's global layer — the deterministic core of `atl publish` (ring
// 2→3 of gain circulation).
//
// publish is the consent-proposal verb: it never runs automatically (it crosses
// the author boundary). This package only *decides* — ownership and the set of
// publishable gains. Applying them (own team → re-publish; not-owned →
// propose-upstream via a gh fork + PR, with the PR body authored by the
// /publish skill) is the command layer's job and is tested live, not here.
package publish

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/agentteamland/atl/cli/internal/fanout"
)

// Change is one file whose global-layer copy differs from the published version.
type Change struct {
	Rel string // path relative to the .claude dir (slash-separated)
	New bool   // absent in the published version (a brand-new file to contribute)
}

// Owns reports whether ghLogin owns repo ("owner/name") — i.e. the repo owner
// segment equals the GitHub login. This is the handle-namespace ownership check
// (mesut/my-team): a personal team you own re-publishes; anything else proposes
// upstream. (Org-maintained repos where you have push but aren't the owner are a
// later refinement — they'd need a push-permission check, not just the owner.)
func Owns(repo, ghLogin string) bool {
	owner, _, ok := strings.Cut(repo, "/")
	return ok && owner != "" && ghLogin != "" && owner == ghLogin
}

// Plan diffs the team's global-layer files against the freshly-fetched published
// version: every file that differs (or is absent upstream) is a publishable
// gain. Whole-file (hash) comparison, consistent with promote/fanout — no
// content merge. candidates are the .claude-relative paths to consider (the
// install manifest's file set, plus any grown files the caller discovered); the
// function stays pure by taking them as input.
func Plan(globalClaude, publishedDir string, candidates []string) ([]Change, error) {
	var changes []Change
	for _, rel := range candidates {
		g, err := fanout.HashFile(filepath.Join(globalClaude, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		if g == "" {
			continue // not present in the global layer → nothing to publish
		}
		p, err := fanout.HashFile(filepath.Join(publishedDir, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		if g == p {
			continue // identical to what's already published
		}
		changes = append(changes, Change{Rel: rel, New: p == ""})
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Rel < changes[j].Rel })
	return changes, nil
}
