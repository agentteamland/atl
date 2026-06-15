package publish

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/teampkg"
)

// BranchName is the deterministic branch a propose-upstream contribution uses.
func BranchName(handle, name string) string {
	return "atl-publish/" + sanitize(handle) + "-" + sanitize(name)
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// repoName returns the name segment of an "owner/name" repo.
func repoName(repo string) string {
	if _, name, ok := strings.Cut(repo, "/"); ok {
		return name
	}
	return repo
}

// Stage copies each change's bytes from the global .claude tree into a checked-
// out repo working tree at <subpath>/<rel>, creating parent dirs. It returns the
// repo-relative (slash-separated) paths written, for `git add`. Pure file I/O —
// no git, no network — so it's unit-testable with two temp dirs.
//
// The subpath join is mandatory: Change.Rel is .claude-relative (the repo's
// subpath was stripped at fetch time), but the repo tree needs the subpath
// prefix back. subpath "" (a standalone team) joins to just rel.
func Stage(globalClaude, repoRoot, subpath string, changes []Change) ([]string, error) {
	var written []string
	for _, c := range changes {
		src := filepath.Join(globalClaude, filepath.FromSlash(c.Rel))
		repoRel := path.Join(subpath, c.Rel) // slash-joined; "" subpath → c.Rel
		dst := filepath.Join(repoRoot, filepath.FromSlash(repoRel))
		if err := teampkg.CopyFile(src, dst); err != nil {
			return nil, fmt.Errorf("stage %s: %w", c.Rel, err)
		}
		written = append(written, repoRel)
	}
	return written, nil
}

// ProposeUpstream contributes the global-layer gains for a team you don't own:
// fork the source repo, branch off its default branch, stage the gains under the
// team's subpath, push to your fork, and open a PR against the source repo with
// bodyFile as its body. Returns the PR URL.
//
// The PR body is authored by the /publish skill (the CLI/Skill boundary); this
// only does the mechanics. The user's local + global gains never depend on the
// owner accepting the PR.
func ProposeUpstream(gh GH, m *manifest.Manifest, globalClaude, bodyFile string, changes []Change) (string, error) {
	login, err := gh.Login()
	if err != nil {
		return "", fmt.Errorf("resolve gh login: %w", err)
	}
	base, err := gh.DefaultBranch(m.Source.Repo)
	if err != nil {
		return "", fmt.Errorf("resolve default branch of %s: %w", m.Source.Repo, err)
	}
	if err := gh.Fork(m.Source.Repo); err != nil {
		return "", fmt.Errorf("fork %s: %w", m.Source.Repo, err)
	}
	fork := login + "/" + repoName(m.Source.Repo)

	dir, err := os.MkdirTemp("", "atl-publish-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	if err := gh.Clone(fork, dir); err != nil {
		return "", fmt.Errorf("clone fork %s: %w", fork, err)
	}
	// Branch off the upstream default so the diff is clean even if the fork is
	// stale (an old fork's default can lag the source).
	if _, err := gh.Git(dir, "remote", "add", "upstream", "https://github.com/"+m.Source.Repo+".git"); err != nil {
		return "", err
	}
	if _, err := gh.Git(dir, "fetch", "upstream", base); err != nil {
		return "", err
	}
	branch := BranchName(m.Handle, m.Name)
	if _, err := gh.Git(dir, "checkout", "-b", branch, "upstream/"+base); err != nil {
		return "", err
	}

	paths, err := Stage(globalClaude, dir, m.Source.Subpath, changes)
	if err != nil {
		return "", err
	}
	if _, err := gh.Git(dir, append([]string{"add", "--"}, paths...)...); err != nil {
		return "", err
	}
	subject := fmt.Sprintf("atl publish: contribute %d gain(s) to %s/%s", len(changes), m.Handle, m.Name)
	if _, err := gh.Git(dir, "commit", "-m", subject); err != nil {
		return "", err
	}
	if _, err := gh.Git(dir, "push", "-u", "origin", branch); err != nil {
		return "", err
	}
	title := fmt.Sprintf("Contribute usage gains to %s", m.Name)
	url, err := gh.PRCreate(m.Source.Repo, base, login+":"+branch, title, bodyFile)
	if err != nil {
		return "", fmt.Errorf("open PR: %w", err)
	}
	return url, nil
}
