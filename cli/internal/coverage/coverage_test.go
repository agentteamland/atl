package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// fakeGit returns a canned `git diff` and records the argv, so the hunk parsing
// is asserted without building a repo.
func fakeGit(out string) (Runner, *[]string) {
	var argv []string
	return func(name string, args ...string) ([]byte, error) {
		argv = append([]string{name}, args...)
		return []byte(out), nil
	}, &argv
}

// --- diff parsing ------------------------------------------------------------

func TestDiffLinesReadsTheNewSideOfEachHunk(t *testing.T) {
	run, argv := fakeGit(`diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -3,0 +4,2 @@
+const a = 1;
+const b = 2;
@@ -10 +12 @@
+changed
`)
	got, err := DiffLines(run, "origin/dev")
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]bool{4: true, 5: true, 12: true}
	if len(got["app.js"]) != len(want) {
		t.Fatalf("app.js lines = %v, want %v", got["app.js"], want)
	}
	for l := range want {
		if !got["app.js"][l] {
			t.Errorf("line %d missing from %v", l, got["app.js"])
		}
	}
	// --unified=0 is load-bearing: with context lines a change would ride on the
	// tests covering its neighbours.
	joined := fmt.Sprint(*argv)
	for _, flag := range []string{"--unified=0", "--diff-filter=d", "origin/dev...HEAD"} {
		if !contains(joined, flag) {
			t.Errorf("git diff must pass %s; argv was %v", flag, *argv)
		}
	}
}

// A pure deletion cannot be covered, and counting it would make removing dead
// code fail a coverage gate.
func TestDiffLinesIgnoresDeletions(t *testing.T) {
	run, _ := fakeGit(`--- a/old.js
+++ b/old.js
@@ -5,3 +4,0 @@
-gone
-gone
-gone
`)
	got, err := DiffLines(run, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(got["old.js"]) != 0 {
		t.Errorf("a 0-count hunk contributed lines: %v", got["old.js"])
	}
}

// --- the three formats -------------------------------------------------------

func TestCoveredParsesGoProfile(t *testing.T) {
	p := write(t, "cover.out", `mode: set
github.com/x/y/app.go:10.20,12.3 2 1
github.com/x/y/app.go:20.2,21.4 1 0
`)
	meas, got, err := Covered(p)
	if err != nil {
		t.Fatal(err)
	}
	if !meas["github.com/x/y/app.go"][20] {
		t.Error("a count-0 block is MEASURABLE (the tool saw it) even though it is not covered")
	}
	m := got["github.com/x/y/app.go"]
	for _, l := range []int{10, 11, 12} {
		if !m[l] {
			t.Errorf("executed block line %d missing: %v", l, m)
		}
	}
	if m[20] || m[21] {
		t.Error("a block with count 0 was never executed and must not count as covered")
	}
}

func TestCoveredParsesLCOV(t *testing.T) {
	p := write(t, "lcov.info", `SF:src/app.ts
DA:1,3
DA:2,0
end_of_record
SF:src/other.ts
DA:9,1
end_of_record
`)
	meas, got, err := Covered(p)
	if err != nil {
		t.Fatal(err)
	}
	if !meas["src/app.ts"][2] {
		t.Error("a hits=0 line is MEASURABLE — the tool reported it, it just never ran")
	}
	if !got["src/app.ts"][1] || got["src/app.ts"][2] {
		t.Errorf("hits>0 covered, hits==0 not: %v", got["src/app.ts"])
	}
	// end_of_record must reset the current file, or one record's lines leak into
	// the next file's set.
	if got["src/app.ts"][9] {
		t.Error("a DA line leaked across end_of_record into the previous file")
	}
	if !got["src/other.ts"][9] {
		t.Errorf("second record not parsed: %v", got)
	}
}

func TestCoveredParsesCobertura(t *testing.T) {
	p := write(t, "cobertura.xml", `<?xml version="1.0"?>
<coverage><packages><package><classes>
<class filename="src/Api/Fee.cs"><lines>
<line number="7" hits="4"/><line number="8" hits="0"/>
</lines></class>
</classes></package></packages></coverage>`)
	meas, got, err := Covered(p)
	if err != nil {
		t.Fatal(err)
	}
	if !meas["src/Api/Fee.cs"][8] {
		t.Error("a hits=0 line is MEASURABLE — the tool reported it, it just never ran")
	}
	if !got["src/Api/Fee.cs"][7] || got["src/Api/Fee.cs"][8] {
		t.Errorf("hits>0 covered, hits==0 not: %v", got["src/Api/Fee.cs"])
	}
}

func TestCoveredRejectsAnUnknownFormat(t *testing.T) {
	if _, _, err := Covered(write(t, "junk.txt", "not a coverage report\n")); err == nil {
		t.Error("an unrecognized report must error, not silently measure nothing")
	}
}

// --- the two rules the gate's honesty rests on -------------------------------

// THE load-bearing case. A brand-new source file with no test at all is ABSENT
// from coverage output. Skipping absent files would score it 100% — the gate
// would be greenest exactly where the code is least tested.
func TestAnUntestedNewFileCountsAsZeroNotSkipped(t *testing.T) {
	diff := Lines{"src/brand-new.ts": {1: true, 2: true, 3: true}}
	covered := Lines{"src/existing.ts": {1: true}} // the report knows .ts, not this file

	got := Measure(diff, covered, covered)
	if got.Total != 3 || got.Covered != 0 {
		t.Fatalf("want 0/3 for a file absent from the report, got %d/%d", got.Covered, got.Total)
	}
	if got.Percent != 0 {
		t.Errorf("percent = %v, want 0 — an absent file is untested, not unmeasured", got.Percent)
	}
}

// The mirror rule: a change to files no coverage tool measures must not be
// scored 0% or every documentation PR fails the gate.
func TestNonSourceFilesAreExcludedEntirely(t *testing.T) {
	diff := Lines{"README.md": {1: true, 2: true}, "config.yaml": {5: true}}
	covered := Lines{"src/app.ts": {1: true}}

	got := Measure(diff, covered, covered)
	if got.Total != 0 {
		t.Fatalf("docs/config must not enter the denominator, got total=%d", got.Total)
	}
	if got.Percent != 100 {
		t.Errorf("percent = %v, want 100 — nothing measurable changed", got.Percent)
	}
}

func TestUncoveredLinesAreReportedSoTheyCanBeFixed(t *testing.T) {
	diff := Lines{"src/app.ts": {1: true, 2: true, 3: true, 4: true}}
	meas := Lines{"src/app.ts": {1: true, 2: true, 3: true, 4: true}}
	covered := Lines{"src/app.ts": {1: true, 3: true}}

	got := Measure(diff, meas, covered)
	if got.Covered != 2 || got.Total != 4 || got.Percent != 50 {
		t.Fatalf("got %d/%d = %v%%, want 2/4 = 50%%", got.Covered, got.Total, got.Percent)
	}
	if len(got.Files) != 1 || fmt.Sprint(got.Files[0].Uncovered) != "[2 4]" {
		t.Errorf("uncovered lines = %v, want [2 4] — the point is to say where to write the test", got.Files)
	}
}

// Reports are rooted differently by nearly every tool, so matching is by suffix
// — but on a path boundary, or `app.js` matches `vendor/notapp.js`.
func TestPathMatchingIsSuffixOnAPathBoundary(t *testing.T) {
	diff := Lines{"app.js": {1: true}}
	covered := Lines{"/build/src/app.js": {1: true}}
	if got := Measure(diff, covered, covered); got.Covered != 1 {
		t.Errorf("a differently-rooted report path should still match: %+v", got)
	}

	diff2 := Lines{"app.js": {1: true}}
	covered2 := Lines{"vendor/notapp.js": {1: true}}
	if got := Measure(diff2, covered2, covered2); got.Covered != 0 {
		t.Errorf("notapp.js must NOT match app.js: %+v", got)
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

// --- the denominator rules the first real run exposed -------------------------

// A coverage tool reports STATEMENTS. Comments, blank lines and imports appear
// nowhere in its output, so counting them as uncovered penalises documentation
// and makes the ratio meaningless. Measured on this package's own first run: 37
// of one file's "uncovered" lines were its package doc and imports.
func TestUnmeasurableLinesAreNotInTheDenominator(t *testing.T) {
	// The change touched 10 lines; only 5 and 6 are statements.
	diff := Lines{"src/app.ts": {1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true}}
	meas := Lines{"src/app.ts": {5: true, 6: true}}
	covered := Lines{"src/app.ts": {5: true, 6: true}}

	got := Measure(diff, meas, covered)
	if got.Total != 2 || got.Covered != 2 || got.Percent != 100 {
		t.Fatalf("got %d/%d = %v%%, want 2/2 = 100%% — comments must not count", got.Covered, got.Total, got.Percent)
	}
}

// You do not write a test for a test. On this package's own first run its test
// file contributed 222 lines of pure noise to the denominator.
func TestTestFilesAreExcludedFromTheDenominator(t *testing.T) {
	for _, f := range []string{
		"internal/x/thing_test.go", "src/Component.test.tsx", "src/util.spec.ts",
		"lib/widget_test.dart", "tests/FeeTests.cs", "app/__tests__/helper.ts", "api/test_client.py",
	} {
		if !isTestFile(f) {
			t.Errorf("%s should be recognized as a test file", f)
		}
	}
	for _, f := range []string{"src/app.ts", "internal/x/thing.go", "src/Component.tsx", "lib/widget.dart"} {
		if isTestFile(f) {
			t.Errorf("%s is production code, not a test file", f)
		}
	}

	diff := Lines{"src/app.ts": {1: true}, "src/app.test.ts": {1: true, 2: true, 3: true}}
	meas := Lines{"src/app.ts": {1: true}}
	covered := Lines{"src/app.ts": {1: true}}

	got := Measure(diff, meas, covered)
	if got.Total != 1 {
		t.Fatalf("the test file entered the denominator: total=%d, want 1", got.Total)
	}
}

// --- the error branches ------------------------------------------------------
//
// Every one of these is a path where the gate could fail OPEN — return an empty
// or partial measurement that scores 100% — so they are exactly the branches
// worth pinning rather than the ones worth excusing.

func TestCoveredSurfacesAnUnreadableReport(t *testing.T) {
	if _, _, err := Covered(filepath.Join(t.TempDir(), "nope.out")); err == nil {
		t.Error("a missing report must error; silently measuring nothing scores 100%")
	}
}

func TestDiffLinesSurfacesAGitFailure(t *testing.T) {
	run := func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("fatal: bad revision")
	}
	_, err := DiffLines(run, "no-such-ref")
	if err == nil {
		t.Fatal("a failed git diff must error; an empty diff measures 0 lines and passes")
	}
	if !contains(err.Error(), "no-such-ref") {
		t.Errorf("the error should name the ref that failed: %v", err)
	}
}

// Malformed input must be skipped, never panic and never counted — a report
// half-parsed into a smaller denominator scores higher than the truth.
func TestParsersSkipMalformedRecordsWithoutCounting(t *testing.T) {
	goP := write(t, "cover.out", `mode: set
this-line-has-no-colon-span
github.com/x/y/a.go:notanumber,2.3 1 1
github.com/x/y/a.go:5.1,4.9 1 1
github.com/x/y/a.go:10.1,10.9 1 notanumber
github.com/x/y/a.go:20.1,20.9 1 1
`)
	meas, cov, err := Covered(goP)
	if err != nil {
		t.Fatal(err)
	}
	if len(meas["github.com/x/y/a.go"]) != 1 || !cov["github.com/x/y/a.go"][20] {
		t.Errorf("only the one well-formed block should count: meas=%v cov=%v",
			meas["github.com/x/y/a.go"], cov["github.com/x/y/a.go"])
	}

	lcovP := write(t, "lcov.info", `SF:src/a.ts
DA:notanumber,1
DA:3
DA:7,alsonotanumber
DA:9,1
end_of_record
`)
	meas2, cov2, err := Covered(lcovP)
	if err != nil {
		t.Fatal(err)
	}
	if len(meas2["src/a.ts"]) != 1 || !cov2["src/a.ts"][9] {
		t.Errorf("only the one well-formed DA should count: meas=%v", meas2["src/a.ts"])
	}
}

// A DA line before any SF has no file to belong to. Attributing it to whatever
// file came last would inflate that file's measurable set.
func TestLCOVIgnoresLinesWithNoCurrentFile(t *testing.T) {
	p := write(t, "lcov.info", "DA:1,1\nSF:src/a.ts\nDA:2,1\nend_of_record\nDA:3,1\n")
	meas, _, err := Covered(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(meas) != 1 || len(meas["src/a.ts"]) != 1 || !meas["src/a.ts"][2] {
		t.Errorf("only the DA inside a record should count: %v", meas)
	}
}

func TestCoberturaSurfacesMalformedXML(t *testing.T) {
	if _, _, err := Covered(write(t, "c.xml", "<coverage><packages>truncated")); err == nil {
		t.Error("unparseable XML must error, not yield an empty set that scores 100 percent")
	}
}

// A hunk header git never produces must be skipped rather than mis-parsed into
// a wrong line range.
func TestDiffLinesSkipsUnparseableHunkHeaders(t *testing.T) {
	run, _ := fakeGit(`+++ b/a.ts
@@ garbage @@
@@ -1 +notanumber @@
@@ -1 +5,notanumber @@
@@ -1 +9 @@
+ok
`)
	got, err := DiffLines(run, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(got["a.ts"]) != 1 || !got["a.ts"][9] {
		t.Errorf("only the well-formed hunk should contribute: %v", got["a.ts"])
	}
}

// A file with no measurable changed lines must drop out entirely rather than
// enter as 0/0 — an empty FileResult in the list reads like a finding.
func TestMeasureDropsAFileWithNoMeasurableChanges(t *testing.T) {
	diff := Lines{"src/a.ts": {1: true, 2: true}}
	meas := Lines{"src/a.ts": {50: true}} // the change touched only comments
	res := Measure(diff, meas, Lines{"src/a.ts": {50: true}})
	if len(res.Files) != 0 || res.Total != 0 || res.Percent != 100 {
		t.Errorf("a file whose changed lines are all unmeasurable should drop out: %+v", res)
	}
}
