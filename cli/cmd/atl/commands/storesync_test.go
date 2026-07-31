package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/manifest"
)

// versionDeclaredStores is the seam between "what teams declared" and "what gets
// versioned". Its job is to read the declaration out of the install manifests at
// both layers — a store declared by a globally-installed team must be found from
// inside any project, which is the case that matters, since profile-team is
// global-only.
func TestVersionDeclaredStoresReadsBothLayers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := t.TempDir()

	globalStore := filepath.Join(home, ".atl", "profiles")
	projectStore := filepath.Join(home, ".atl", "notes")
	for _, d := range []string{globalStore, projectStore} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "a.md"), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	must := func(m *manifest.Manifest, layer string) {
		t.Helper()
		if err := m.Write(layer); err != nil {
			t.Fatal(err)
		}
	}
	must(&manifest.Manifest{Handle: "acme", Name: "global-team", Stores: []string{"~/.atl/profiles"}}, filepath.Join(home, ".atl"))
	must(&manifest.Manifest{Handle: "acme", Name: "project-team", Stores: []string{"~/.atl/notes"}}, filepath.Join(project, ".atl"))

	if n := versionDeclaredStores(project); n != 2 {
		t.Fatalf("versionDeclaredStores = %d, want 2 (one per layer)", n)
	}
	for _, d := range []string{globalStore, projectStore} {
		if _, err := os.Stat(filepath.Join(d, ".git")); err != nil {
			t.Fatalf("%s was not versioned: %v", d, err)
		}
	}
}

// A team that declares no store must not cause any work — the common case.
func TestVersionDeclaredStoresQuietWithNoDeclarations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := t.TempDir()

	m := &manifest.Manifest{Handle: "acme", Name: "plain", Files: map[string]string{}}
	if err := m.Write(filepath.Join(project, ".atl")); err != nil {
		t.Fatal(err)
	}
	if n := versionDeclaredStores(project); n != 0 {
		t.Fatalf("versionDeclaredStores = %d, want 0", n)
	}
}

// With no project root resolved, the project layer must be skipped entirely:
// LayerDir would otherwise build a RELATIVE ".atl" and read whatever happens to
// sit under the current working directory.
func TestVersionDeclaredStoresSkipsAnUnresolvedProject(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cwd := t.TempDir()
	m := &manifest.Manifest{Handle: "acme", Name: "stray", Stores: []string{"~/.atl/stray"}}
	if err := m.Write(filepath.Join(cwd, ".atl")); err != nil {
		t.Fatal(err)
	}
	stray := filepath.Join(home, ".atl", "stray")
	if err := os.MkdirAll(stray, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stray, "a.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	restore, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(restore)

	if n := versionDeclaredStores(""); n != 0 {
		t.Fatalf("versionDeclaredStores = %d, want 0 — it read a relative project layer", n)
	}
}
