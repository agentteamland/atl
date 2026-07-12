package index

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSeedLoads(t *testing.T) {
	ix, err := Seed()
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if ix.SchemaVersion != 1 {
		t.Errorf("schemaVersion = %d, want 1", ix.SchemaVersion)
	}
	// The seed may legitimately be empty (the v1-era first-party teams were
	// retired 2026-07; the catalog refills as teams are rebuilt/published) —
	// but any entry it does carry must be complete.
	for _, e := range ix.Teams {
		if e.Handle == "" || e.Name == "" || e.Source.Repo == "" || e.Source.Ref == "" {
			t.Errorf("seed entry %q has empty required field: %+v", e.Ref(), e)
		}
	}
}

// fixtureIndex is a synthetic two-team catalog for Lookup/Search tests, so they
// don't depend on what the embedded seed happens to contain.
func fixtureIndex(t *testing.T) *Index {
	t.Helper()
	ix, err := Load([]byte(`{"schemaVersion":1,"teams":[
		{"handle":"acme","name":"example-team","version":"1.0.0",
		 "description":"An example team for mobile apps built with Flutter.",
		 "keywords":["flutter","mobile"],
		 "source":{"repo":"agentteamland/atl","subpath":"teams/example-team","ref":"v1"}},
		{"handle":"acme","name":"proto-team","version":"0.1.0",
		 "description":"UI prototypes and design tokens.",
		 "keywords":["design"],
		 "source":{"repo":"acme/proto-team","subpath":"","ref":"v2"}}]}`))
	if err != nil {
		t.Fatalf("fixture Load: %v", err)
	}
	return ix
}

func TestLookup(t *testing.T) {
	ix := fixtureIndex(t)
	e, err := ix.Lookup("acme", "example-team")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if e.Source.Repo != "agentteamland/atl" {
		t.Errorf("source repo = %q", e.Source.Repo)
	}
	if e.Source.Subpath != "teams/example-team" {
		t.Errorf("source subpath = %q", e.Source.Subpath)
	}
	if e.Ref() != "acme/example-team" {
		t.Errorf("Ref() = %q", e.Ref())
	}
	if _, err := ix.Lookup("nobody", "ghost"); err == nil {
		t.Error("expected not-found error for unknown team")
	}
}

func TestSearch(t *testing.T) {
	ix := fixtureIndex(t)
	// Blank query browses the whole catalog.
	if got := ix.Search(""); len(got) != len(ix.Teams) {
		t.Errorf("Search(\"\") = %d, want all %d teams", len(got), len(ix.Teams))
	}
	// A keyword unique to one team matches exactly it, case-insensitively.
	for _, q := range []string{"flutter", "FLUTTER"} {
		if hits := ix.Search(q); len(hits) != 1 || hits[0].Ref() != "acme/example-team" {
			t.Errorf("Search(%q) = %v, want [acme/example-team]", q, refs(hits))
		}
	}
	// Description text matches too (only proto-team mentions prototypes).
	if hits := ix.Search("prototyp"); len(hits) != 1 || hits[0].Ref() != "acme/proto-team" {
		t.Errorf("Search(prototyp) = %v, want [acme/proto-team]", refs(hits))
	}
	// A miss returns nothing.
	if hits := ix.Search("no-such-team-xyz"); len(hits) != 0 {
		t.Errorf("Search(miss) = %v, want none", refs(hits))
	}
}

func refs(es []Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Ref()
	}
	return out
}

func TestLoadInvalid(t *testing.T) {
	if _, err := Load([]byte("{not json")); err == nil {
		t.Error("expected parse error on malformed JSON")
	}
}

func TestParseRef(t *testing.T) {
	h, n, err := ParseRef("mesut/my-team")
	if err != nil || h != "mesut" || n != "my-team" {
		t.Errorf("ParseRef(mesut/my-team) = %q,%q,%v", h, n, err)
	}
	for _, bad := range []string{"", "noslash", "/empty", "empty/", "a/b/c"} {
		if _, _, err := ParseRef(bad); err == nil {
			t.Errorf("ParseRef(%q) expected error, got nil", bad)
		}
	}
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"schemaVersion":1,"teams":[{"handle":"h","name":"n","source":{"repo":"h/n","ref":"v1"}}]}`))
	}))
	defer srv.Close()
	ix, err := Fetch(srv.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(ix.Teams) != 1 || ix.Teams[0].Ref() != "h/n" {
		t.Errorf("fetched index = %+v", ix.Teams)
	}
}

func TestFetchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := Fetch(srv.URL); err == nil {
		t.Error("expected error on HTTP 404")
	}
}

func TestResolveFallsBackToSeed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	// no cache present → embedded seed
	ix, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	seed, err := Seed()
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if ix.SchemaVersion != seed.SchemaVersion || len(ix.Teams) != len(seed.Teams) {
		t.Errorf("Resolve without a cache should return the seed: got %d teams, seed has %d", len(ix.Teams), len(seed.Teams))
	}
}

func TestRefreshCacheThenResolveUsesCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"schemaVersion":1,"teams":[{"handle":"only","name":"cached","source":{"repo":"only/cached","ref":"v9"}}]}`))
	}))
	defer srv.Close()

	if err := RefreshCache(srv.URL); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}
	if _, err := os.Stat(mustCachePath(t)); err != nil {
		t.Fatalf("cache not written: %v", err)
	}
	ix, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(ix.Teams) != 1 || ix.Teams[0].Ref() != "only/cached" {
		t.Errorf("Resolve should use the cache, got %+v", ix.Teams)
	}
}

// TestResolvePrefersNewerGeneratedAt: a cache older than the embedded seed must
// not mask a freshly-upgraded binary's newer catalog; a newer cache still wins.
func TestResolvePrefersNewerGeneratedAt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	writeCache := func(generatedAt string) {
		body := `{"schemaVersion":1,"generatedAt":"` + generatedAt + `","teams":[{"handle":"only","name":"stale","source":{"repo":"only/stale","ref":"v1"}}]}`
		p := mustCachePath(t)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Cache older than the seed (2026-07-10) → the seed wins (the unique cache team absent).
	writeCache("2000-01-01T00:00:00Z")
	ix, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if len(ix.Teams) == 1 && ix.Teams[0].Ref() == "only/stale" {
		t.Error("a cache older than the seed must not be preferred over the seed")
	}

	// Cache newer than the seed → the cache wins.
	writeCache("2099-01-01T00:00:00Z")
	ix, err = Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if len(ix.Teams) != 1 || ix.Teams[0].Ref() != "only/stale" {
		t.Errorf("a cache newer than the seed should be preferred, got %+v", ix.Teams)
	}
}

func mustCachePath(t *testing.T) string {
	t.Helper()
	p, err := CachePath()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLookupByName(t *testing.T) {
	ix := &Index{Teams: []Entry{
		{Handle: "user1", Name: "shared"},
		{Handle: "agentteamland", Name: "shared", Verified: true},
		{Handle: "acme", Name: "solo"},
	}}
	if e, err := ix.LookupByName("shared"); err != nil || e.Handle != "agentteamland" {
		t.Errorf("LookupByName(shared) = %+v, %v; verified should win", e, err)
	}
	if e, err := ix.LookupByName("solo"); err != nil || e.Handle != "acme" {
		t.Errorf("LookupByName(solo) = %+v, %v", e, err)
	}
	if _, err := ix.LookupByName("nope"); err == nil {
		t.Error("LookupByName(nope) should error")
	}
}
