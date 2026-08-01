package teampkg

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadManifest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "team.json"),
		`{"schemaVersion":1,"name":"x","version":"1.0.0","scope":"global","capabilities":{"review":"r"}}`)
	tm, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if tm.Name != "x" || tm.Scope != "global" || tm.Version != "1.0.0" {
		t.Errorf("got %+v", tm)
	}
}

// A team declares where it keeps durable state; the platform reads the
// declaration so it can version that state without knowing which team owns it.
func TestDeclaredStores(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "team.json"), `{
	  "schemaVersion": 1, "name": "x", "version": "1.0.0", "scope": "global",
	  "capabilities": {
	    "review": "a bare string — an older capability shape",
	    "profile": {"provider": true, "store": "~/.atl/profiles"},
	    "archive": {"store": "  "},
	    "notes":   {"store": "~/.atl/notes"}
	  }
	}`)
	tm, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	got := tm.DeclaredStores()
	// Sorted by capability name ("notes" < "profile"), so a re-install produces a
	// byte-identical manifest instead of churning on Go's map iteration order.
	want := []string{"~/.atl/notes", "~/.atl/profiles"}
	if len(got) != len(want) {
		t.Fatalf("DeclaredStores = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DeclaredStores = %v, want %v", got, want)
		}
	}
}

// The common case, and the one that must not error: a team with no durable store
// of its own.
func TestDeclaredStoresEmptyWhenNoneDeclared(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "team.json"),
		`{"schemaVersion":1,"name":"x","version":"1.0.0","scope":"global","capabilities":{"review":"r"}}`)
	tm, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if got := tm.DeclaredStores(); len(got) != 0 {
		t.Fatalf("DeclaredStores = %v, want none", got)
	}
}

func TestReadManifestMissingName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "team.json"), `{"version":"1.0.0"}`)
	if _, err := ReadManifest(dir); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestCopyAssets(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "team.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(src, "README.md"), "readme")
	writeFile(t, filepath.Join(src, "agents/api/agent.md"), "API")
	writeFile(t, filepath.Join(src, "skills/build/skill.md"), "BUILD")
	writeFile(t, filepath.Join(src, "rules/r.md"), "RULE")
	writeFile(t, filepath.Join(src, "knowledge/adapter.md"), "ADAPTER")
	writeFile(t, filepath.Join(src, "scripts/helper.sh"), "#!/usr/bin/env bash\necho hi\n")
	writeFile(t, filepath.Join(src, "packs/web/pack.md"), "PACK")                // M1 knowledge-pack seam — must reflect too
	writeFile(t, filepath.Join(src, "backends/github/adapter.md"), "GH-ADAPTER") // per-backend adapter contract — must reflect too

	claude := t.TempDir()
	files, err := CopyAssets(src, claude)
	if err != nil {
		t.Fatalf("CopyAssets: %v", err)
	}
	for _, rel := range []string{"agents/api/agent.md", "skills/build/skill.md", "rules/r.md", "knowledge/adapter.md", "scripts/helper.sh", "packs/web/pack.md", "backends/github/adapter.md"} {
		if _, err := os.Stat(filepath.Join(claude, rel)); err != nil {
			t.Errorf("missing copied %s: %v", rel, err)
		}
		if files[rel] == "" {
			t.Errorf("files map missing %s", rel)
		}
	}
	if _, err := os.Stat(filepath.Join(claude, "team.json")); err == nil {
		t.Error("team.json should not be copied")
	}
	if _, err := os.Stat(filepath.Join(claude, "README.md")); err == nil {
		t.Error("README.md should not be copied")
	}
	if files["team.json"] != "" {
		t.Error("team.json should not be in files map")
	}
}

func TestCopyAssetsNoAssets(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "team.json"), `{"name":"x"}`)
	if _, err := CopyAssets(src, t.TempDir()); err == nil {
		t.Error("expected error when team ships no assets")
	}
}

func TestCopyFilePreservesExecBit(t *testing.T) {
	src := filepath.Join(t.TempDir(), "helper.sh")
	if err := os.WriteFile(src, []byte("#!/usr/bin/env bash\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "helper.sh")
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	fi, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("exec bit not preserved: dst mode = %v, want 0755", fi.Mode().Perm())
	}
}

func TestReadManifestDependencies(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "team.json"),
		`{"name":"consumer","version":"1.0.0","dependencies":{"core":"^1.0.0","profile-team":"^1.1.0"}}`)
	tm, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if tm.Dependencies["profile-team"] != "^1.1.0" || tm.Dependencies["core"] != "^1.0.0" {
		t.Errorf("dependencies = %+v", tm.Dependencies)
	}
}

// A team declares the capture channels it owns; the platform reads the
// declaration so it can emit each channel's signal without knowing which team
// ships the drain skill and the rule that acts on it.
func TestDeclaredChannels(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "team.json"), `{
	  "schemaVersion": 1, "name": "x", "version": "1.0.0", "scope": "global",
	  "capabilities": {
	    "review": "a bare string — an older capability shape",
	    "profile": {"role": "provider", "store": "~/.atl/profiles", "channel": {
	      "name": "profile-fact", "drain": "/profile-drain",
	      "rule": "profile-capture", "describes": "durable entity facts"}},
	    "plain":   {"store": "~/.atl/notes"},
	    "blank":   {"channel": {"name": "  ", "drain": "/x", "rule": "y", "describes": "z"}},
	    "audit":   {"channel": {"name": "audit-note", "drain": "/audit-drain",
	      "rule": "audit-capture", "describes": "audit notes"}}
	  }
	}`)
	tm, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	got := tm.DeclaredChannels()
	// Sorted by capability name ("audit" < "profile"), so a re-install produces a
	// byte-identical manifest instead of churning on Go's map iteration order.
	// "review" (a bare string), "plain" (no channel key) and "blank" (no name) are
	// all skipped rather than erroring — the same tolerance DeclaredStores applies.
	if len(got) != 2 {
		t.Fatalf("DeclaredChannels = %+v, want 2", got)
	}
	if got[0].Name != "audit-note" || got[1].Name != "profile-fact" {
		t.Fatalf("DeclaredChannels order = %q,%q, want audit-note,profile-fact", got[0].Name, got[1].Name)
	}
	if got[1].Drain != "/profile-drain" || got[1].Rule != "profile-capture" || got[1].Describes != "durable entity facts" {
		t.Errorf("every field a signal sentence needs must survive: %+v", got[1])
	}
	if !got[1].Valid() {
		t.Errorf("a fully-specified declaration must be Valid: %+v", got[1])
	}
}

// A named declaration missing one of the other fields is RETURNED, deliberately:
// dropping it here would make "declared but broken" indistinguishable from "not
// declared" without re-fetching the team source, so nothing could ever report it.
// The read side refuses it; this side records it.
func TestDeclaredChannelsKeepsABrokenDeclarationForReporting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "team.json"), `{
	  "schemaVersion": 1, "name": "x", "version": "1.0.0",
	  "capabilities": {"profile": {"channel": {"name": "profile-fact", "drain": "/profile-drain"}}}
	}`)
	tm, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	got := tm.DeclaredChannels()
	if len(got) != 1 {
		t.Fatalf("DeclaredChannels = %+v, want the broken declaration kept", got)
	}
	if got[0].Valid() {
		t.Error("a declaration missing rule + describes must not be Valid")
	}
	if missing := got[0].MissingFields(); len(missing) != 2 || missing[0] != "rule" || missing[1] != "describes" {
		t.Errorf("MissingFields = %v, want [rule describes]", missing)
	}
}

// The common case, and the one that must not error: a team that owns no capture
// channel. Its install records none, so no signal can ever name it.
func TestDeclaredChannelsEmptyWhenNoneDeclared(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "team.json"),
		`{"schemaVersion":1,"name":"x","version":"1.0.0","capabilities":{"review":"r","profile":{"store":"~/.atl/p"}}}`)
	tm, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if got := tm.DeclaredChannels(); len(got) != 0 {
		t.Fatalf("DeclaredChannels = %+v, want none", got)
	}
}
