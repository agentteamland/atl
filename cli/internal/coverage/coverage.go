// Package coverage measures DIFF coverage: of the lines a change added or
// modified, how many does the test suite actually execute?
//
// Why diff coverage and not a whole-project percentage: a project-wide number
// is unusable as a gate on an existing codebase (it starts far below any
// meaningful target, so a threshold blocks every unit on day one and gets
// switched off within a week), and it is insensitive on a large one (a hundred
// untested new lines barely move it). The lines a change wrote are the lines
// the change is answerable for, whatever the age of everything around them.
//
// The measurement is deliberately narrow. Coverage proves a line RAN; it says
// nothing about whether anything CHECKED the result — a test that calls the code
// and asserts nothing scores 100%. That second half is a human/reviewer
// judgement (does a test go red when the change is reverted?) and this package
// does not pretend to make it. It closes the other hole: a number nobody
// measured.
package coverage

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Runner runs an external command and returns its combined output — the git
// seam, mirroring internal/dispatch's Runner so tests inject a fake instead of
// building a repo.
type Runner func(name string, args ...string) ([]byte, error)

// Lines maps a slash-separated file path to a set of line numbers.
type Lines map[string]map[int]bool

func (l Lines) add(file string, line int) {
	if l[file] == nil {
		l[file] = map[int]bool{}
	}
	l[file][line] = true
}

// FileResult is one file's contribution to the measurement.
type FileResult struct {
	Path      string `json:"path"`
	Covered   int    `json:"covered"`
	Total     int    `json:"total"`
	Uncovered []int  `json:"uncovered,omitempty"` // the lines to go write a test for
}

// Result is the whole measurement. Percent is 100 when Total is 0 — a change
// that added no measurable source lines (docs, config) is not a coverage
// failure, and reporting 0% there would block every documentation PR.
type Result struct {
	Files   []FileResult `json:"files"`
	Covered int          `json:"covered"`
	Total   int          `json:"total"`
	Percent float64      `json:"percent"`

	// Provenance — carried on the result so a rendered measurement names its own
	// inputs. A number whose base ref and report path you cannot see is not
	// reviewable: a stale report scores exactly as green as a fresh one.
	Base   string  `json:"base,omitempty"`
	Report string  `json:"report,omitempty"`
	Min    float64 `json:"min,omitempty"`
}

// DiffLines returns, per file, the line numbers that base...HEAD ADDED or
// MODIFIED — the lines the change is answerable for.
//
// `--unified=0` so each hunk header names exactly the changed lines with no
// surrounding context; context lines are not this change's work and counting
// them would let a change ride on its neighbours' tests. Deletions contribute
// nothing: a removed line cannot be covered, and counting it would make deleting
// dead code fail a coverage gate.
func DiffLines(run Runner, base string) (Lines, error) {
	out, err := run("git", "diff", "--unified=0", "--no-color", "--diff-filter=d", base+"...HEAD")
	if err != nil {
		return nil, fmt.Errorf("git diff against %s: %w", base, err)
	}
	lines := Lines{}
	var file string
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		t := sc.Text()
		switch {
		case strings.HasPrefix(t, "+++ b/"):
			file = strings.TrimPrefix(t, "+++ b/")
		case strings.HasPrefix(t, "+++ "):
			file = "" // /dev/null — a deletion; nothing to attribute
		case strings.HasPrefix(t, "@@") && file != "":
			start, count, ok := parseHunk(t)
			if !ok {
				continue
			}
			for i := 0; i < count; i++ {
				lines.add(file, start+i)
			}
		}
	}
	return lines, sc.Err()
}

// parseHunk reads the NEW-side span from `@@ -a,b +c,d @@`. An omitted count
// means 1 (git's shorthand); a count of 0 is a pure deletion and adds nothing.
func parseHunk(h string) (start, count int, ok bool) {
	i := strings.Index(h, "+")
	if i < 0 {
		return 0, 0, false
	}
	rest := h[i+1:]
	if j := strings.IndexAny(rest, " @"); j >= 0 {
		rest = rest[:j]
	}
	numStr, cntStr := rest, "1"
	if c := strings.Index(rest, ","); c >= 0 {
		numStr, cntStr = rest[:c], rest[c+1:]
	}
	start, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, 0, false
	}
	count, err = strconv.Atoi(cntStr)
	if err != nil {
		return 0, 0, false
	}
	return start, count, true
}

// Covered parses a coverage report into two sets: the lines the tool could
// MEASURE at all, and the subset of those it saw EXECUTED. The format is
// detected from the content, not the filename, so a report written to an
// unconventional path still works.
//
// The two-set shape is what makes the ratio mean anything. A coverage tool
// reports STATEMENTS — a comment, a blank line, an import block and a bare `}`
// appear nowhere in its output. Treating "absent from the report" as "uncovered"
// therefore scores a well-documented file terribly and rewards writing none, so
// the denominator is measurable-and-changed, never merely changed. Measured on
// this package's own first run: 37 of one file's "uncovered" lines were its
// package doc and imports.
func Covered(path string) (measurable, covered Lines, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	s := string(b)
	switch {
	case strings.HasPrefix(strings.TrimSpace(s), "mode:"):
		return parseGo(s)
	case strings.Contains(s, "\nSF:") || strings.HasPrefix(s, "SF:"):
		return parseLCOV(s)
	case strings.Contains(s, "<coverage"):
		return parseCobertura(b)
	}
	return nil, nil, fmt.Errorf("%s: unrecognized coverage format (expected Go coverprofile, lcov, or cobertura XML)", path)
}

// parseGo reads a `go test -coverprofile` file: `file:startLine.col,endLine.col numStmt count`.
// A block with count 0 was never executed, so its lines are not added.
func parseGo(s string) (Lines, Lines, error) {
	measurable, covered := Lines{}, Lines{}
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if t == "" || strings.HasPrefix(t, "mode:") {
			continue
		}
		colon := strings.LastIndex(t, ":")
		if colon < 0 {
			continue
		}
		file, spans := t[:colon], strings.Fields(t[colon+1:])
		if len(spans) != 3 {
			continue
		}
		count, err := strconv.Atoi(spans[2])
		if err != nil {
			continue
		}
		from, to, ok := goSpan(spans[0])
		if !ok {
			continue
		}
		for l := from; l <= to; l++ {
			measurable.add(normalize(file), l)
			if count > 0 {
				covered.add(normalize(file), l)
			}
		}
	}
	return measurable, covered, sc.Err()
}

// goSpan reads `startLine.startCol,endLine.endCol`.
func goSpan(s string) (from, to int, ok bool) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	first, err1 := strconv.Atoi(strings.SplitN(parts[0], ".", 2)[0])
	last, err2 := strconv.Atoi(strings.SplitN(parts[1], ".", 2)[0])
	if err1 != nil || err2 != nil || last < first {
		return 0, 0, false
	}
	return first, last, true
}

// parseLCOV reads lcov: `SF:<file>` then `DA:<line>,<hits>`. Emitted by c8 /
// vitest and by `flutter test --coverage`, so one parser covers both stacks.
func parseLCOV(s string) (Lines, Lines, error) {
	measurable, covered := Lines{}, Lines{}
	var file string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(t, "SF:"):
			file = normalize(strings.TrimPrefix(t, "SF:"))
		case strings.HasPrefix(t, "DA:") && file != "":
			f := strings.SplitN(strings.TrimPrefix(t, "DA:"), ",", 2)
			if len(f) != 2 {
				continue
			}
			line, err1 := strconv.Atoi(f[0])
			hits, err2 := strconv.Atoi(strings.SplitN(f[1], ",", 2)[0])
			if err1 != nil || err2 != nil {
				continue
			}
			measurable.add(file, line)
			if hits > 0 {
				covered.add(file, line)
			}
		case t == "end_of_record":
			file = ""
		}
	}
	return measurable, covered, sc.Err()
}

// coberturaXML is the subset of the schema this needs. `filename` is relative
// to a `<source>` root, which is dropped — matching is by suffix (see Measure).
type coberturaXML struct {
	Packages []struct {
		Classes []struct {
			Filename string `xml:"filename,attr"`
			Lines    []struct {
				Number int `xml:"number,attr"`
				Hits   int `xml:"hits,attr"`
			} `xml:"lines>line"`
		} `xml:"classes>class"`
	} `xml:"packages>package"`
}

// parseCobertura reads the XML coverlet (.NET) and many other tools emit.
func parseCobertura(b []byte) (Lines, Lines, error) {
	var doc coberturaXML
	if err := xml.Unmarshal(b, &doc); err != nil {
		return nil, nil, fmt.Errorf("cobertura xml: %w", err)
	}
	measurable, covered := Lines{}, Lines{}
	for _, p := range doc.Packages {
		for _, c := range p.Classes {
			for _, l := range c.Lines {
				measurable.add(normalize(c.Filename), l.Number)
				if l.Hits > 0 {
					covered.add(normalize(c.Filename), l.Number)
				}
			}
		}
	}
	return measurable, covered, nil
}

func normalize(p string) string {
	return strings.TrimPrefix(filepath.ToSlash(p), "./")
}

// isTestFile reports whether a path is a test file by the naming conventions the
// major runtimes share. You do not write a test for a test, so a changed test
// file must not enter the denominator — measured on this package's own first
// run, its test file alone contributed 222 lines of pure noise.
//
// This is a heuristic list and it is stated as one. It cannot be derived from the
// report the way source extensions can, because most tools omit test files from
// coverage output entirely — which is exactly why they would otherwise land in
// the "absent, therefore zero" bucket below.
func isTestFile(p string) bool {
	b := strings.ToLower(filepath.Base(p))
	switch {
	case strings.HasSuffix(b, "_test.go"), // go
		strings.HasSuffix(b, "_test.dart"), // dart / flutter
		strings.HasSuffix(b, "tests.cs"),   // .net, the common convention
		strings.Contains(b, ".test."),      // js/ts: foo.test.ts, foo.test.tsx
		strings.Contains(b, ".spec."),      // js/ts: foo.spec.ts
		strings.HasPrefix(b, "test_"):      // python
		return true
	}
	return strings.Contains(filepath.ToSlash(p), "/__tests__/")
}

// Measure intersects the changed lines with the covered ones.
//
// Three rules decide whether this gate is real, and all three are about what
// belongs in the DENOMINATOR — the numerator is the easy half:
//
//   - For a file the report knows, only its MEASURABLE lines count. A coverage
//     tool reports statements, so comments, blank lines and imports appear
//     nowhere in its output; counting them as uncovered would penalise writing
//     documentation and make the ratio meaningless.
//   - A changed source file the report never mentions counts as ZERO covered
//     over ALL its changed lines — deliberately harsh. A brand-new untested file
//     is absent from coverage output, and skipping absent files would score it
//     100%: the gate would be greenest exactly where the code is least tested.
//     The inflation is acceptable because the correct action is identical (write
//     a test), and once one exists the file enters the report and the measurement
//     becomes exact.
//   - Test files and non-source files are excluded entirely. Markdown and JSON are
//     not measurable; a test file is not something you write a test for.
//
// Source extensions are derived from the report itself rather than hardcoded, so
// a stack this package has never seen still measures correctly.
func Measure(diff, measurable, covered Lines) Result {
	sourceExt := map[string]bool{}
	for f := range measurable {
		sourceExt[filepath.Ext(f)] = true
	}

	res := Result{}
	for file, want := range diff {
		if !sourceExt[filepath.Ext(file)] || isTestFile(file) {
			continue
		}
		known := lookup(measurable, file)
		got := lookup(covered, file)

		fr := FileResult{Path: file}
		for _, l := range sortedKeys(want) {
			if known != nil && !known[l] {
				continue // not a statement — a comment, a blank line, an import
			}
			fr.Total++
			if got[l] {
				fr.Covered++
			} else {
				fr.Uncovered = append(fr.Uncovered, l)
			}
		}
		if fr.Total == 0 {
			continue // nothing measurable changed in this file
		}
		res.Files = append(res.Files, fr)
		res.Covered += fr.Covered
		res.Total += fr.Total
	}
	sort.Slice(res.Files, func(i, j int) bool { return res.Files[i].Path < res.Files[j].Path })

	res.Percent = 100
	if res.Total > 0 {
		res.Percent = float64(res.Covered) * 100 / float64(res.Total)
	}
	return res
}

// lookup matches a repo-relative path against the report's own paths, which are
// rooted differently by nearly every tool (absolute, module-relative, prefixed
// by a source root). Exact first, then the longest suffix match on a path
// boundary — the boundary check is what stops `app.js` from matching
// `vendor/notapp.js`. A missing file yields an empty set, which is the
// zero-covered case Measure depends on.
func lookup(covered Lines, file string) map[int]bool {
	if m, ok := covered[file]; ok {
		return m
	}
	best, bestLen := map[int]bool(nil), -1
	for cf, m := range covered {
		if cf == file || strings.HasSuffix(cf, "/"+file) || strings.HasSuffix(file, "/"+cf) {
			if n := len(cf); n > bestLen {
				best, bestLen = m, n
			}
		}
	}
	return best
}

func sortedKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
