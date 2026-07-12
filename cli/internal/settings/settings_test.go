package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

var v2Hooks = []Hook{
	{Event: "SessionStart", Command: "atl session-start"},
	{Event: "UserPromptSubmit", Command: "atl tick --throttle=10m"},
}

func readHooks(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatal(err)
	}
	hm, _ := obj["hooks"].(map[string]any)
	return hm
}

func TestInstallHooksIdempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	path, err := InstallHooks(v2Hooks)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InstallHooks(v2Hooks); err != nil { // run again
		t.Fatal(err)
	}

	hm := readHooks(t, path)
	for _, event := range []string{"SessionStart", "UserPromptSubmit"} {
		groups, _ := hm[event].([]any)
		if len(groups) != 1 {
			t.Fatalf("%s: want 1 group after two installs, got %d", event, len(groups))
		}
	}
}

func TestInstallHooksPreservesUserHooks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// seed a user-owned SessionStart hook
	if err := os.MkdirAll(home+"/.claude", 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"my-own-thing"}]}]}}`
	if err := os.WriteFile(home+"/.claude/settings.json", []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := InstallHooks(v2Hooks)
	if err != nil {
		t.Fatal(err)
	}

	groups, _ := readHooks(t, path)["SessionStart"].([]any)
	if len(groups) != 2 {
		t.Fatalf("want 2 SessionStart groups (user + atl), got %d", len(groups))
	}
	// the user's command must still be present
	found := false
	for _, g := range groups {
		hs := g.(map[string]any)["hooks"].([]any)
		for _, h := range hs {
			if h.(map[string]any)["command"] == "my-own-thing" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("user's own hook was dropped")
	}
}

func TestInstallHooksMatcher(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path, err := InstallHooks([]Hook{
		{Event: "PreToolUse", Matcher: "Bash|Edit|Write", Command: "atl guard"},
		{Event: "SessionStart", Command: "atl session-start"},
	})
	if err != nil {
		t.Fatal(err)
	}
	hm := readHooks(t, path)

	// The PreToolUse group carries the tool matcher.
	pre, _ := hm["PreToolUse"].([]any)
	if len(pre) != 1 {
		t.Fatalf("want 1 PreToolUse group, got %d", len(pre))
	}
	if m := pre[0].(map[string]any)["matcher"]; m != "Bash|Edit|Write" {
		t.Fatalf("PreToolUse matcher: got %v, want Bash|Edit|Write", m)
	}

	// A matcher-less event emits no matcher key (unchanged shape).
	ss, _ := hm["SessionStart"].([]any)
	if _, has := ss[0].(map[string]any)["matcher"]; has {
		t.Fatal("SessionStart group should not carry a matcher key")
	}
}

func TestInstallHooksCommandShape(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path, err := InstallHooks(v2Hooks)
	if err != nil {
		t.Fatal(err)
	}
	groups, _ := readHooks(t, path)["UserPromptSubmit"].([]any)
	cmd := groups[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)["command"]
	if cmd != "atl tick --throttle=10m" {
		t.Fatalf("command shape: got %q", cmd)
	}
}

// TestInstallHooksRefusesUnparseable: a non-empty settings.json that won't parse
// must NOT be silently overwritten (which would wipe every non-hook key).
func TestInstallHooksRefusesUnparseable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	corrupt := `{"permissions": {"allow": [ // trailing comma + comment = invalid`
	if err := os.WriteFile(path, []byte(corrupt), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallHooks(v2Hooks); err == nil {
		t.Fatal("InstallHooks should refuse to overwrite an unparseable settings.json")
	}
	// The user's file must be left byte-for-byte intact.
	if b, _ := os.ReadFile(path); string(b) != corrupt {
		t.Errorf("the corrupt file was modified; got %q", b)
	}
}
