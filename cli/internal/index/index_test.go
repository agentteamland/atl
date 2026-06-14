package index

import (
	"net/http"
	"net/http/httptest"
	"os"
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
	if len(ix.Teams) < 2 {
		t.Fatalf("teams = %d, want >= 2", len(ix.Teams))
	}
	for _, e := range ix.Teams {
		if e.Handle == "" || e.Name == "" || e.Source.Repo == "" || e.Source.Ref == "" {
			t.Errorf("seed entry %q has empty required field: %+v", e.Ref(), e)
		}
	}
}

func TestLookup(t *testing.T) {
	ix, err := Seed()
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	e, err := ix.Lookup("agentteamland", "software-project-team")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if e.Source.Repo != "agentteamland/software-project-team" {
		t.Errorf("source repo = %q", e.Source.Repo)
	}
	if e.Ref() != "agentteamland/software-project-team" {
		t.Errorf("Ref() = %q", e.Ref())
	}
	if _, err := ix.Lookup("nobody", "ghost"); err == nil {
		t.Error("expected not-found error for unknown team")
	}
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
	if len(ix.Teams) < 2 {
		t.Errorf("expected seed (>=2 teams), got %d", len(ix.Teams))
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

func mustCachePath(t *testing.T) string {
	t.Helper()
	p, err := CachePath()
	if err != nil {
		t.Fatal(err)
	}
	return p
}
