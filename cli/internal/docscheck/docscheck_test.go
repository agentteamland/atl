package docscheck

import (
	"os"
	"path/filepath"
	"testing"
)

// buildSite writes a minimal site tree under a temp dir and returns its path.
func buildSite(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	site := filepath.Join(root, "docs", "site")
	for rel, body := range files {
		p := filepath.Join(site, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// the resolver keys off .vitepress; make sure it exists
	if err := os.MkdirAll(filepath.Join(site, ".vitepress"), 0o755); err != nil {
		t.Fatal(err)
	}
	return site
}

func has(f []Finding, check, substr string) bool {
	for _, x := range f {
		if x.Check == check && (substr == "" || contains(x.Detail, substr)) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool { return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestCoverageMissingAndOrphan(t *testing.T) {
	site := buildSite(t, map[string]string{
		"cli/install.md":  "install page",
		"cli/overview.md": "intro (allowlisted)",
		"cli/ghost.md":    "describes nothing",
	})
	cmds := []Command{{Name: "install"}, {Name: "promote"}}
	f := Coverage(site, cmds)

	if !has(f, "coverage", "atl promote` has no docs page") {
		t.Error("want missing-page finding for promote")
	}
	if !has(f, "coverage", "no shipping command `atl ghost") {
		t.Error("want orphan-page finding for ghost")
	}
	if has(f, "coverage", "overview") {
		t.Error("overview is allowlisted; should not be flagged")
	}
	if has(f, "coverage", "atl install` has no docs page") {
		t.Error("install has a page; should not be flagged")
	}
}

func TestParityMissingMirror(t *testing.T) {
	site := buildSite(t, map[string]string{
		"cli/install.md":    "en",
		"cli/promote.md":    "en",
		"tr/cli/install.md": "tr",
		// tr/cli/promote.md intentionally missing
	})
	f := Parity(site)
	if !has(f, "parity", "tr/cli/promote.md") {
		t.Error("want parity finding for the missing promote mirror")
	}
	if has(f, "parity", "tr/cli/install.md") {
		t.Error("install has a mirror; should not be flagged")
	}
}

func TestTokensStaleInstruction(t *testing.T) {
	// The denylist is instructional, not bare-noun: a live install command for a
	// retired channel is flagged, but a bare historical-contrast mention is not.
	site := buildSite(t, map[string]string{
		"guide/install.md": "Run `brew install atl` to get started.",
		"guide/history.md":  "v1 fanned out to winget; it was retired in v2. v1's /save-learnings is now /drain.",
	})
	f := Tokens(site, DefaultDenylist)
	if !has(f, "tokens", "brew install") {
		t.Error("want a finding for the stale `brew install` instruction")
	}
	if has(f, "tokens", "winget") || has(f, "tokens", "save-learnings") {
		t.Error("bare historical-contrast mentions must NOT be flagged (the round-4 lesson)")
	}
}

func TestLinksBrokenInternal(t *testing.T) {
	site := buildSite(t, map[string]string{
		"cli/install.md": "[ok](./overview) [bad](./nope) [ext](https://example.com) [anchor](#x)",
		"cli/overview.md": "target",
	})
	f := Links(site)
	if !has(f, "links", "./nope") {
		t.Error("want a broken-link finding for ./nope")
	}
	if has(f, "links", "./overview") {
		t.Error("./overview resolves; should not be flagged")
	}
	if has(f, "links", "example.com") {
		t.Error("external links are out of scope for the internal-link check")
	}
}

func TestFlagsUndocumented(t *testing.T) {
	site := buildSite(t, map[string]string{
		"cli/install.md": "Use the --global flag to install globally.",
	})
	cmds := []Command{{Name: "install", Flags: []string{"global", "project"}}}
	f := Flags(site, cmds)
	if !has(f, "flags", "--project") {
		t.Error("want an undocumented-flag finding for --project")
	}
	if has(f, "flags", "--global") {
		t.Error("--global is documented; should not be flagged")
	}
}

func TestSkillCoverage(t *testing.T) {
	site := buildSite(t, map[string]string{
		"skills/drain.md": "drain page",
		"skills/ghost.md": "no such skill",
	})
	// build a core/skills tree as a sibling of docs/
	repo := filepath.Dir(filepath.Dir(site)) // <root>
	for _, s := range []string{"drain", "create-pr"} {
		if err := os.MkdirAll(filepath.Join(repo, "core", "skills", s), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	f := SkillCoverage(site, filepath.Join(repo, "core"))
	if !has(f, "coverage", "/create-pr` has no docs page") {
		t.Error("want missing-page finding for the create-pr skill")
	}
	if !has(f, "coverage", "no shipping skill `/ghost") {
		t.Error("want orphan-page finding for the ghost skill page")
	}
}
