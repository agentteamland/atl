package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSlugForPath(t *testing.T) {
	cases := []struct{ path, want string }{
		// dot-free path — leading slash + separators become hyphens
		{"/Users/foo/projects/myapp", "-Users-foo-projects-myapp"},
		// a dot in a path component (.claude) must slug to a hyphen too — this is the
		// worktree case that silently broke capture. Verified against a real on-disk
		// Claude Code slug: /Users/x/p/.claude/worktrees/y -> ...p--claude-worktrees-y
		{
			"/Users/mesutkurak/projects/beekod/BeeCommerce/.claude/worktrees/dazzling-morse-b2e4ae",
			"-Users-mesutkurak-projects-beekod-BeeCommerce--claude-worktrees-dazzling-morse-b2e4ae",
		},
		// every other non-alphanumeric also folds to a hyphen
		{"/a/b.c/d_e", "-a-b-c-d-e"},
	}
	for _, c := range cases {
		if got := SlugForPath(c.path); got != c.want {
			t.Errorf("SlugForPath(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestExtractText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.jsonl")
	lines := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Decided. <!-- learning: A -->"},{"type":"tool_use","name":"x"}]}}
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<!-- learning: from-user-ignored -->"}]}}
{"type":"assistant","message":{"role":"assistant","content":"string shape <!-- profile-fact: B -->"}}
not even json
`
	if err := os.WriteFile(path, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
	text, err := ExtractText(path)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !strings.Contains(text, "<!-- learning: A -->") {
		t.Error("missing assistant array-content marker")
	}
	if !strings.Contains(text, "<!-- profile-fact: B -->") {
		t.Error("missing assistant string-content marker")
	}
	if strings.Contains(text, "from-user-ignored") {
		t.Error("user-message text must be ignored")
	}
}

func TestExtractFlow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.jsonl")
	lines := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"do the auth thing"}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"on it"},{"type":"tool_use","name":"Edit"}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}
{"type":"user","message":{"role":"user","content":"no, use refresh tokens not sessions"}}
{"type":"assistant","message":{"role":"assistant","content":"fixed"}}
not even json
`
	if err := os.WriteFile(path, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
	turns, err := ExtractFlow(path)
	if err != nil {
		t.Fatalf("flow: %v", err)
	}
	// Expected: user "do the auth thing", assistant "on it", user "no, use…",
	// assistant "fixed". The tool_result-only user message yields no text → dropped.
	if len(turns) != 4 {
		t.Fatalf("want 4 prose turns, got %d: %+v", len(turns), turns)
	}
	if turns[0].Role != "user" || turns[0].Text != "do the auth thing" {
		t.Errorf("turn0: %+v", turns[0])
	}
	if turns[1].Role != "assistant" || turns[1].Text != "on it" {
		t.Errorf("turn1 (tool_use must be dropped): %+v", turns[1])
	}
	if turns[2].Role != "user" || !strings.Contains(turns[2].Text, "refresh tokens") {
		t.Errorf("turn2 (the correction): %+v", turns[2])
	}
	for _, tn := range turns {
		if strings.Contains(tn.Text, "ok") && tn.Text == "ok" {
			t.Error("tool_result content must not become a turn")
		}
	}
}

func TestFindModTimeFilterAndOrder(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.jsonl")
	recent := filepath.Join(dir, "recent.jsonl")
	_ = os.WriteFile(old, []byte("{}"), 0o600)
	_ = os.WriteFile(recent, []byte("{}"), 0o600)
	t0 := time.Now().Add(-2 * time.Hour)
	t1 := time.Now().Add(-1 * time.Minute)
	_ = os.Chtimes(old, t0, t0)
	_ = os.Chtimes(recent, t1, t1)

	// since 1h ago → only recent
	files, err := Find(dir, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0].Path) != "recent.jsonl" {
		t.Fatalf("modtime filter: got %+v", files)
	}

	// zero since → both, oldest first
	all, _ := Find(dir, time.Time{})
	if len(all) != 2 || filepath.Base(all[0].Path) != "old.jsonl" {
		t.Fatalf("ordering: got %+v", all)
	}
}

func TestFindMissingDir(t *testing.T) {
	files, err := Find(filepath.Join(t.TempDir(), "nope"), time.Time{})
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if files != nil {
		t.Fatalf("want nil, got %+v", files)
	}
}

func TestFindIgnoresNonJsonl(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("{}"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0o600)
	files, _ := Find(dir, time.Time{})
	if len(files) != 1 {
		t.Fatalf("want 1 jsonl, got %d", len(files))
	}
}
