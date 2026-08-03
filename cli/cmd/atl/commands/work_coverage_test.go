package commands

import (
	"fmt"
	"os"

	"github.com/agentteamland/atl/cli/internal/coverage"
	"path/filepath"
	"testing"
	"time"
)

func touch(t *testing.T, root, rel string) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// Discovery exists so a caller does not have to know which runner the project
// uses. Each family the packs actually name must be found, or the gate degrades
// into "pass --report by hand" — which is the same as not having it, since the
// path most likely to be skipped is the one nobody remembers.
func TestFindReportLocatesEachRunnersOutput(t *testing.T) {
	for _, rel := range []string{
		"coverage/lcov.info",              // c8 / vitest, and flutter --coverage
		"lcov.info",                       //
		"coverage/cobertura-coverage.xml", //
		"cover.out",                       // go test -coverprofile
		"coverage.out",                    //
	} {
		root := t.TempDir()
		want := touch(t, root, rel)
		got, err := findReport(root)
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		if got != want {
			t.Errorf("%s: found %q, want %q", rel, got, want)
		}
	}
}

// coverlet writes under a per-run GUID directory that cannot be named ahead of
// time, so this family is only reachable by glob. Without it the .NET stack —
// one of the two the first real project uses — has no discovery at all.
func TestFindReportGlobsCoverletsGuidDirectory(t *testing.T) {
	root := t.TempDir()
	want := touch(t, root, "TestResults/8f3c1e22-0000-4b7a-9d11-abcdef123456/coverage.cobertura.xml")

	got, err := findReport(root)
	if err != nil {
		t.Fatalf("coverlet output not discovered: %v", err)
	}
	if got != want {
		t.Errorf("found %q, want %q", got, want)
	}
}

// A directory named like a report must not satisfy discovery — the parse would
// fail later with a confusing error instead of the actionable one here.
func TestFindReportIgnoresADirectoryOfThatName(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cover.out"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := findReport(root); err == nil {
		t.Error("a directory named cover.out was accepted as a report")
	}
}

// The failure has to be actionable: "no report" is a state the developer fixes
// by running the pack's coverage command, so the error says that rather than
// leaving them to guess.
func TestFindReportFailsWithAnActionableMessage(t *testing.T) {
	_, err := findReport(t.TempDir())
	if err == nil {
		t.Fatal("an empty tree must not yield a report")
	}
	msg := err.Error()
	for _, want := range []string{"no coverage report", "production-unit.md", "--report"} {
		if !contains(msg, want) {
			t.Errorf("error should mention %q so the reader knows what to do; got: %s", want, msg)
		}
	}
}

// Ordering is a contract: the first candidate wins, so a project holding two
// reports gets a deterministic answer rather than a filesystem-order one.
func TestFindReportPrefersTheFirstCandidate(t *testing.T) {
	root := t.TempDir()
	first := touch(t, root, "coverage/lcov.info")
	touch(t, root, "cover.out")

	got, err := findReport(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != first {
		t.Errorf("found %q, want the earlier candidate %q", got, first)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- the decision logic, now that it is reachable ----------------------------

func fakeRun(status, diff string) coverage.Runner {
	return func(name string, args ...string) ([]byte, error) {
		for _, a := range args {
			if a == "--porcelain" {
				return []byte(status), nil
			}
		}
		return []byte(diff), nil
	}
}

const oneChangedLine = `--- a/src/app.ts
+++ b/src/app.ts
@@ -0,0 +1 @@
+const a = 1;
`

// The dirty-tree note is the difference between a real pass and a free one: run
// mid-edit, `base...HEAD` sees nothing and the measurement comes back 100%.
func TestMeasureWarnsWhenTheTreeIsDirty(t *testing.T) {
	root := t.TempDir()
	touch(t, root, "coverage/lcov.info")
	if err := os.WriteFile(filepath.Join(root, "coverage", "lcov.info"),
		[]byte("SF:src/app.ts\nDA:1,1\nend_of_record\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var warned []string
	_, err := measureCoverage(root, "main", "", 90, fakeRun(" M src/app.ts\n", oneChangedLine),
		func(m string) { warned = append(warned, m) })
	if err != nil {
		t.Fatal(err)
	}
	if len(warned) != 1 || !contains(warned[0], "COMMITTED work only") {
		t.Errorf("a dirty tree must be surfaced, got %v", warned)
	}

	warned = nil
	if _, err := measureCoverage(root, "main", "", 90, fakeRun("", oneChangedLine),
		func(m string) { warned = append(warned, m) }); err != nil {
		t.Fatal(err)
	}
	if len(warned) != 0 {
		t.Errorf("a clean tree must not warn, got %v", warned)
	}
}

// The result carries its own inputs, because a number whose base and report you
// cannot see is not reviewable — a stale report scores exactly as green.
func TestMeasureCarriesItsProvenance(t *testing.T) {
	root := t.TempDir()
	rp := filepath.Join(root, "cover.out")
	if err := os.WriteFile(rp, []byte("mode: set\nsrc/app.ts:1.1,1.9 1 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := measureCoverage(root, "origin/dev", "", 90, fakeRun("", oneChangedLine), func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if res.Base != "origin/dev" || res.Report != rp || res.Min != 90 {
		t.Errorf("provenance missing: base=%q report=%q min=%v", res.Base, res.Report, res.Min)
	}
}

// Without a config AND without --base there is no defensible ref to measure
// against, so it must refuse rather than guess one.
func TestMeasureRefusesWithNoBaseAndNoConfig(t *testing.T) {
	_, err := measureCoverage(t.TempDir(), "", "", 90, fakeRun("", ""), func(string) {})
	if err == nil {
		t.Fatal("no config and no --base must be an error, not a guessed default")
	}
	if !contains(err.Error(), "--base") {
		t.Errorf("the error should name the flag that fixes it: %v", err)
	}
}

func TestRenderNamesItsInputsAndTheUncoveredLines(t *testing.T) {
	res := coverage.Result{
		Files:   []coverage.FileResult{{Path: "src/app.ts", Covered: 1, Total: 3, Uncovered: []int{2, 3}}},
		Covered: 1, Total: 3, Percent: 33.3, Base: "origin/dev", Report: "coverage/lcov.info", Min: 90,
	}
	out := renderCoverage(res, false)
	for _, want := range []string{"33.3%", "1/3", "origin/dev", "coverage/lcov.info", "src/app.ts", "[2 3]"} {
		if !contains(out, want) {
			t.Errorf("rendered output should contain %q:\n%s", want, out)
		}
	}

	js := renderCoverage(res, true)
	for _, want := range []string{`"passed": false`, `"base": "origin/dev"`, `"uncovered"`} {
		if !contains(js, want) {
			t.Errorf("the JSON evidence form should contain %s:\n%s", want, js)
		}
	}
}

func TestRenderSaysSoWhenNothingMeasurableChanged(t *testing.T) {
	out := renderCoverage(coverage.Result{Percent: 100, Base: "main", Report: "r"}, false)
	if !contains(out, "no measurable source lines") {
		t.Errorf("a 0/0 result must say why it is 100%%, not just print it:\n%s", out)
	}
}

// Each of these is a path where the measurement could come back empty — and an
// empty measurement scores 100%. They are the branches most worth pinning.

func TestMeasurePropagatesAnUnparseableReport(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "cover.out"), []byte("mode: set\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A valid header with no records is parseable; a garbage file is not.
	bad := filepath.Join(root, "garbage.txt")
	if err := os.WriteFile(bad, []byte("not a coverage report"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := measureCoverage(root, "main", bad, 90, fakeRun("", oneChangedLine), func(string) {}); err == nil {
		t.Error("an unparseable report must error, not measure nothing and pass")
	}
}

func TestMeasurePropagatesAGitFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "cover.out"), []byte("mode: set\nsrc/a.ts:1.1,1.9 1 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	failing := func(name string, args ...string) ([]byte, error) {
		for _, a := range args {
			if a == "--porcelain" {
				return nil, nil // the status probe is best-effort
			}
		}
		return nil, errBadRev
	}
	if _, err := measureCoverage(root, "no-such-ref", "", 90, failing, func(string) {}); err == nil {
		t.Error("a failed git diff must error; an empty diff measures 0 lines and passes")
	}
}

func TestMeasurePropagatesAMissingReport(t *testing.T) {
	// No report anywhere and none passed: discovery must fail loudly rather than
	// let the caller proceed with nothing to intersect against.
	if _, err := measureCoverage(t.TempDir(), "main", "", 90, fakeRun("", oneChangedLine), func(string) {}); err == nil {
		t.Error("a missing report must error")
	}
}

func TestMeasureRefusesAMalformedDeliveryConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".delivery"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".delivery", "config.json"), []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	// With no --base the config is the only source of a ref; a broken one must
	// surface rather than silently fall through to a guessed default.
	if _, err := measureCoverage(root, "", "", 90, fakeRun("", oneChangedLine), func(string) {}); err == nil {
		t.Error("a malformed delivery config must error rather than yield a guessed base")
	}
}

// The base comes from the project's own config, never a hardcoded "dev" — a
// project that renamed its pair must not be measured against a ref that does
// not exist.
func TestMeasureDefaultsTheBaseToTheConfiguredDevBranch(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".delivery"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"owner":"o","repo":"r","backend":"github","branchPair":{"dev":"integration","release":"live"}}`
	if err := os.WriteFile(filepath.Join(root, ".delivery", "config.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cover.out"), []byte("mode: set\nsrc/app.ts:1.1,1.9 1 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := measureCoverage(root, "", "", 90, fakeRun("", oneChangedLine), func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if res.Base != "origin/integration" {
		t.Errorf("base = %q, want origin/integration — the renamed pair, not a hardcoded dev", res.Base)
	}
}

var errBadRev = fmt.Errorf("fatal: bad revision")

// --- stale profile ----------------------------------------------------------

// The failure this exists for, reproduced: `go test -coverprofile` writes a
// profile even when the test result is CACHED, and that profile carries the
// cached run's line numbers. Measured on a real change, a stale profile put
// three comment lines inside an "uncovered block" and reported a 100% change as
// 63.6%. The other direction is worse — a stale profile can mark newly-added
// lines as covered, which is a false green on the gate that decides done.
func TestMeasureWarnsWhenTheProfileIsOlderThanTheSource(t *testing.T) {
	root := t.TempDir()
	rp := filepath.Join(root, "cover.out")
	if err := os.WriteFile(rp, []byte("mode: set\nsrc/app.ts:1.1,1.9 1 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := touch(t, root, "src/app.ts")
	// The source is newer than the profile by a clear margin.
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(rp, old, old); err != nil {
		t.Fatal(err)
	}

	var warned []string
	if _, err := measureCoverage(root, "main", rp, 90,
		fakeRun("", "--- a/"+src+"\n+++ b/"+src+"\n@@ -0,0 +1 @@\n+const a = 1;\n"),
		func(m string) { warned = append(warned, m) }); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, w := range warned {
		if contains(w, "older than") && contains(w, "-count=1") {
			found = true
		}
	}
	if !found {
		t.Errorf("a profile older than its source must be surfaced, got %v", warned)
	}
}

// The converse, so the warning stays worth reading: a fresh profile is silent.
// A caution that fires on every run is how a real one stops being read — the
// same lesson this whole change set is about.
func TestMeasureIsQuietWhenTheProfileIsFresh(t *testing.T) {
	root := t.TempDir()
	src := touch(t, root, "src/app.ts")
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(src, old, old); err != nil {
		t.Fatal(err)
	}
	rp := filepath.Join(root, "cover.out")
	if err := os.WriteFile(rp, []byte("mode: set\nsrc/app.ts:1.1,1.9 1 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var warned []string
	if _, err := measureCoverage(root, "main", rp, 90,
		fakeRun("", "--- a/"+src+"\n+++ b/"+src+"\n@@ -0,0 +1 @@\n+const a = 1;\n"),
		func(m string) { warned = append(warned, m) }); err != nil {
		t.Fatal(err)
	}
	for _, w := range warned {
		if contains(w, "older than") {
			t.Errorf("a fresh profile must not warn, got %q", w)
		}
	}
}

// A source file the diff names but that is not on disk (deleted in the change,
// or outside the working tree) is not staleness. A staleness check that cries
// wolf on every deletion would be switched off within a week.
func TestStaleProfileIgnoresMissingSources(t *testing.T) {
	root := t.TempDir()
	rp := filepath.Join(root, "cover.out")
	if err := os.WriteFile(rp, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if stale, _ := staleProfile(rp, []string{filepath.Join(root, "gone.go")}); stale {
		t.Error("a missing source must not read as a stale profile")
	}
}

// And a missing report is not staleness either — findReport already errors on
// that path, and reporting it twice in different words helps nobody.
func TestStaleProfileIgnoresMissingReport(t *testing.T) {
	if stale, _ := staleProfile(filepath.Join(t.TempDir(), "nope.out"), []string{"x.go"}); stale {
		t.Error("a missing report must not read as stale")
	}
}
