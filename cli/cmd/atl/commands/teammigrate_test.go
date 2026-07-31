package commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/manifest"
)

// teamSource writes a minimal team.json into a temp dir and returns a fetch that
// serves it, plus a counter of how many times it was called.
func teamSource(t *testing.T, teamJSON string) (fetchFunc, *int) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "team.json"), []byte(teamJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	return func(repo, subpath, ref string) (string, error) {
		calls++
		// The migration must re-fetch the version this install actually resolved
		// from — the manifest's own pin — not whatever the index points at now.
		if repo != "acme/demo" || ref != "v1.0.0" {
			t.Fatalf("fetched %s@%s, want acme/demo@v1.0.0 (the manifest's pin)", repo, ref)
		}
		return dir, nil
	}, &calls
}

func preV2Manifest() *manifest.Manifest {
	return &manifest.Manifest{
		SchemaVersion: 1,
		Handle:        "acme",
		Name:          "demo",
		Version:       "1.0.0",
		Scope:         "global",
		Source:        manifest.Source{Repo: "acme/demo", Ref: "v1.0.0"},
		Files:         map[string]string{"agents/api/agent.md": "hash"},
	}
}

// This is the only path by which an install that predates the `stores` field
// ever gains it — i.e. every user who installed a store-declaring team before
// this shipped. It must fill the field, stamp the schema so it never repeats,
// and touch nothing else.
func TestMigrateTeamManifestBackfillsStores(t *testing.T) {
	layer := t.TempDir()
	fetch, calls := teamSource(t, `{"name":"demo","version":"1.0.0","capabilities":{"profile":{"store":"~/.atl/profiles"}}}`)

	m := preV2Manifest()
	if err := migrateTeamManifest(m, layer, fetch); err != nil {
		t.Fatalf("migrateTeamManifest: %v", err)
	}

	got, err := manifest.Read(layer, "acme", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Stores) != 1 || got.Stores[0] != "~/.atl/profiles" {
		t.Fatalf("stores = %v, want [~/.atl/profiles]", got.Stores)
	}
	if got.SchemaVersion != manifest.SchemaVersion {
		t.Fatalf("schemaVersion = %d, want %d", got.SchemaVersion, manifest.SchemaVersion)
	}
	if got.Version != "1.0.0" || got.Files["agents/api/agent.md"] != "hash" {
		t.Fatalf("the migration disturbed the record: %+v", got)
	}
	if *calls != 1 {
		t.Fatalf("fetch called %d times, want 1", *calls)
	}
}

// The migration is gated on the schema version, so stamping it is what makes the
// run happen exactly once rather than on every update forever.
func TestMigrateTeamManifestDoesNotRepeat(t *testing.T) {
	layer := t.TempDir()
	fetch, _ := teamSource(t, `{"name":"demo","version":"1.0.0","capabilities":{"profile":{"store":"~/.atl/profiles"}}}`)

	m := preV2Manifest()
	if err := migrateTeamManifest(m, layer, fetch); err != nil {
		t.Fatal(err)
	}
	got, err := manifest.Read(layer, "acme", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion < manifest.SchemaVersion {
		t.Fatal("the migration would run again on the next update")
	}
}

// A failed fetch must leave the manifest at the OLD schema, so the next update
// retries. Stamping on failure would strand the install: it would claim a shape
// it never got, and the backfill would never run again.
func TestMigrateTeamManifestLeavesTheSchemaAloneOnFailure(t *testing.T) {
	layer := t.TempDir()
	m := preV2Manifest()
	if err := m.Write(layer); err != nil {
		t.Fatal(err)
	}

	failing := func(repo, subpath, ref string) (string, error) {
		return "", errors.New("network is down")
	}
	if err := migrateTeamManifest(m, layer, failing); err == nil {
		t.Fatal("expected an error from a failing fetch")
	}

	got, err := manifest.Read(layer, "acme", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1 — a failed migration claimed the new schema", got.SchemaVersion)
	}
	if len(got.Stores) != 0 {
		t.Fatalf("stores = %v, want none", got.Stores)
	}
}

// A team that declares no store still migrates: it has nothing to record, but it
// must stop being re-examined on every update.
func TestMigrateTeamManifestHandlesNoDeclaredStore(t *testing.T) {
	layer := t.TempDir()
	fetch, _ := teamSource(t, `{"name":"demo","version":"1.0.0"}`)

	m := preV2Manifest()
	if err := migrateTeamManifest(m, layer, fetch); err != nil {
		t.Fatal(err)
	}
	got, err := manifest.Read(layer, "acme", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Stores) != 0 {
		t.Fatalf("stores = %v, want none", got.Stores)
	}
	if got.SchemaVersion != manifest.SchemaVersion {
		t.Fatalf("schemaVersion = %d, want %d", got.SchemaVersion, manifest.SchemaVersion)
	}
}
