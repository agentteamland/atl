package publish

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/manifest"
)

func TestBranchName(t *testing.T) {
	if got := BranchName("agentteamland", "design-system-team"); got != "atl-publish/agentteamland-design-system-team" {
		t.Errorf("BranchName = %q", got)
	}
	// Non-alnum chars collapse to dashes and the result is trimmed.
	if got := BranchName("Me.Sut", "My Team!"); got != "atl-publish/me-sut-my-team" {
		t.Errorf("BranchName sanitize = %q", got)
	}
}

func TestStage(t *testing.T) {
	glob := t.TempDir()
	repo := t.TempDir()
	writeF(t, filepath.Join(glob, "agents/a/agent.md"), "BODY-A")
	writeF(t, filepath.Join(glob, "skills/s/SKILL.md"), "BODY-S")

	changes := []Change{
		{Rel: "agents/a/agent.md"},
		{Rel: "skills/s/SKILL.md", New: true},
	}
	paths, err := Stage(glob, repo, "teams/demo", changes)
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	// repo-relative paths carry the subpath prefix, slash-separated (for git add).
	want := []string{"teams/demo/agents/a/agent.md", "teams/demo/skills/s/SKILL.md"}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Errorf("paths = %v, want %v", paths, want)
	}
	// Files landed at repoRoot/subpath/rel with the right bytes — the subpath join.
	assertFileEq(t, filepath.Join(repo, "teams/demo/agents/a/agent.md"), "BODY-A")
	assertFileEq(t, filepath.Join(repo, "teams/demo/skills/s/SKILL.md"), "BODY-S")
}

func TestStageNoSubpath(t *testing.T) {
	glob := t.TempDir()
	repo := t.TempDir()
	writeF(t, filepath.Join(glob, "rules/r.md"), "R")
	paths, err := Stage(glob, repo, "", []Change{{Rel: "rules/r.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "rules/r.md" {
		t.Errorf("standalone paths = %v, want [rules/r.md]", paths)
	}
	assertFileEq(t, filepath.Join(repo, "rules/r.md"), "R")
}

// fakeGH records every command argv and returns canned stdout keyed by the
// command's first two tokens (e.g. "gh pr", "gh repo", "git add").
type fakeGH struct {
	calls [][]string
	out   map[string]string
}

func (f *fakeGH) runner(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	key := name
	if len(args) > 0 {
		key = name + " " + args[0]
	}
	return []byte(f.out[key]), nil
}

func TestProposeUpstreamArgv(t *testing.T) {
	glob := t.TempDir()
	writeF(t, filepath.Join(glob, "agents/a/agent.md"), "GAIN")

	f := &fakeGH{out: map[string]string{
		"gh api":  "octocat", // gh api user -q .login
		"gh repo": "main",   // serves repo view (DefaultBranch); fork/clone output unused
		"gh pr":   "https://github.com/agentteamland/atl/pull/123",
	}}
	m := &manifest.Manifest{
		Handle: "agentteamland", Name: "design-system-team",
		Source: manifest.Source{Repo: "agentteamland/atl", Subpath: "teams/design-system-team", Ref: "v2.0.0-alpha.1"},
	}
	bodyFile := filepath.Join(t.TempDir(), "body.md")
	writeF(t, bodyFile, "PR body")

	url, err := ProposeUpstream(GH{Run: f.runner}, m, glob, bodyFile, []Change{{Rel: "agents/a/agent.md"}})
	if err != nil {
		t.Fatalf("ProposeUpstream: %v", err)
	}
	if url != "https://github.com/agentteamland/atl/pull/123" {
		t.Errorf("url = %q", url)
	}

	// The cross-repo PR is the critical argv: base = upstream, head = login:branch.
	pr := findCall(f.calls, "gh", "pr")
	if pr == nil {
		t.Fatal("no gh pr create call recorded")
	}
	assertArg(t, pr, "--repo", "agentteamland/atl")
	assertArg(t, pr, "--base", "main")
	assertArg(t, pr, "--head", "octocat:atl-publish/agentteamland-design-system-team")
	assertArg(t, pr, "--body-file", bodyFile)

	// The gain was git-added with its subpath-prefixed repo-relative path.
	add := findGitCall(f.calls, "add")
	if add == nil || !containsArg(add, "teams/design-system-team/agents/a/agent.md") {
		t.Errorf("git add missing subpath-prefixed path: %v", add)
	}

	// The fork is created without cloning (idempotent), and the PR is opened.
	if findCall(f.calls, "gh", "repo") == nil {
		t.Error("no gh repo (fork/view/clone) calls recorded")
	}
}

// --- helpers ---

func assertFileEq(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(b) != want {
		t.Errorf("%s = %q, want %q", path, b, want)
	}
}

// findCall returns the first recorded call whose first two tokens match (gh
// subcommands: "gh pr", "gh repo").
func findCall(calls [][]string, name, sub string) []string {
	for _, c := range calls {
		if len(c) >= 2 && c[0] == name && c[1] == sub {
			return c
		}
	}
	return nil
}

// findGitCall returns the first `git -C <dir> <sub> ...` call (git subcommands
// sit at index 3 because every Git() call is prefixed with -C <dir>).
func findGitCall(calls [][]string, sub string) []string {
	for _, c := range calls {
		if len(c) >= 4 && c[0] == "git" && c[1] == "-C" && c[3] == sub {
			return c
		}
	}
	return nil
}

func containsArg(call []string, want string) bool {
	for _, a := range call {
		if a == want {
			return true
		}
	}
	return false
}

// assertArg checks that flag is present in call and immediately followed by want.
func assertArg(t *testing.T, call []string, flag, want string) {
	t.Helper()
	for i, a := range call {
		if a == flag {
			if i+1 < len(call) && call[i+1] == want {
				return
			}
			t.Errorf("%s = %q, want %q", flag, valueAfter(call, i), want)
			return
		}
	}
	t.Errorf("flag %s not found in %v", flag, call)
}

func valueAfter(call []string, i int) string {
	if i+1 < len(call) {
		return call[i+1]
	}
	return "<end>"
}
