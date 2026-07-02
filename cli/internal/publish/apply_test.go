package publish

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/manifest"
)

func TestBranchName(t *testing.T) {
	if got := BranchName("acme", "example-team"); got != "atl-publish/acme-example-team" {
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
	// Simulate `gh repo clone <repo> <dir>`: seed the dir with a team.json so
	// RePublish's BumpVersion has a real file to bump.
	if name == "gh" && len(args) >= 4 && args[0] == "repo" && args[1] == "clone" {
		dir := args[3]
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "team.json"), []byte(`{"name":"x","version":"0.1.0"}`+"\n"), 0o644)
	}
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
		Handle: "acme", Name: "example-team",
		Source: manifest.Source{Repo: "agentteamland/atl", Subpath: "teams/example-team", Ref: "v2.0.0-alpha.1"},
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
	assertArg(t, pr, "--head", "octocat:atl-publish/acme-example-team")
	assertArg(t, pr, "--body-file", bodyFile)

	// The gain was git-added with its subpath-prefixed repo-relative path.
	add := findGitCall(f.calls, "add")
	if add == nil || !containsArg(add, "teams/example-team/agents/a/agent.md") {
		t.Errorf("git add missing subpath-prefixed path: %v", add)
	}

	// The fork is created without cloning (idempotent), and the PR is opened.
	if findCall(f.calls, "gh", "repo") == nil {
		t.Error("no gh repo (fork/view/clone) calls recorded")
	}
}

func TestBumpVersion(t *testing.T) {
	repo := t.TempDir()
	writeF(t, filepath.Join(repo, "team.json"), "{\n  \"name\": \"x\",\n  \"version\": \"1.2.9\",\n  \"scope\": \"global\"\n}\n")
	old, nw, err := BumpVersion(repo, "")
	if err != nil {
		t.Fatalf("BumpVersion: %v", err)
	}
	if old != "1.2.9" || nw != "1.2.10" {
		t.Errorf("bump = (%q, %q), want (1.2.9, 1.2.10)", old, nw)
	}
	b, _ := os.ReadFile(filepath.Join(repo, "team.json"))
	if !strings.Contains(string(b), `"version": "1.2.10"`) {
		t.Errorf("version not bumped in file:\n%s", b)
	}
	if !strings.Contains(string(b), `"name": "x"`) || !strings.Contains(string(b), `"scope": "global"`) {
		t.Errorf("other fields/format not preserved:\n%s", b)
	}
}

func TestRePublishArgv(t *testing.T) {
	glob := t.TempDir()
	writeF(t, filepath.Join(glob, "agents/a/agent.md"), "GAIN")

	f := &fakeGH{out: map[string]string{"gh repo": "main"}} // DefaultBranch -> main
	m := &manifest.Manifest{
		Handle: "octocat", Name: "my-team",
		Source: manifest.Source{Repo: "octocat/my-team", Subpath: "", Ref: "v0.1.0"},
	}
	msgFile := filepath.Join(t.TempDir(), "msg.txt")
	writeF(t, msgFile, "chore: re-publish e2e gains")

	tag, err := RePublish(GH{Run: f.runner}, m, glob, msgFile, []Change{{Rel: "agents/a/agent.md"}})
	if err != nil {
		t.Fatalf("RePublish: %v", err)
	}
	if tag != "v0.1.1" {
		t.Errorf("tag = %q, want v0.1.1 (patch bump of the cloned 0.1.0)", tag)
	}
	// Staged both the gain and the version-bumped team.json.
	add := findGitCall(f.calls, "add")
	if add == nil || !containsArg(add, "agents/a/agent.md") || !containsArg(add, "team.json") {
		t.Errorf("git add missing gain or team.json: %v", add)
	}
	// Committed from the skill's message file, tagged, pushed branch + tag.
	commit := findGitCall(f.calls, "commit")
	if commit == nil || !containsArg(commit, msgFile) {
		t.Errorf("git commit not from --body-file: %v", commit)
	}
	tagCall := findGitCall(f.calls, "tag")
	if tagCall == nil || !containsArg(tagCall, "v0.1.1") {
		t.Errorf("git tag call: %v", tagCall)
	}
	if findGitCall(f.calls, "push") == nil {
		t.Error("no git push call")
	}
	// Topic ensured (gh repo edit) — index CI's discovery key.
	if findCall(f.calls, "gh", "repo") == nil {
		t.Error("no gh repo (view/clone/edit) calls recorded")
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
