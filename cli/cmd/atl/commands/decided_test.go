package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedDecided(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// The command's reason for existing: a decision recorded only in the code that
// implements it. One measured failure was refuted by a single line of a
// command's help text and by nothing else, so cli/ is in the search set.
func TestSearchDecidedFindsADecisionRecordedOnlyInCode(t *testing.T) {
	root := seedDecided(t, map[string]string{
		"cli/cmd/atl/commands/promote.go": "// Long: \"lifts gains…\\n\" +\n\t\"It runs automatically in the background tick; this is the manual surface.\"\n",
		".atl/docs/unrelated.md":          "nothing to see\n",
	})
	hits, searched := searchDecided(root, "background tick")
	if len(hits) != 1 {
		t.Fatalf("want the one code line, got %d: %+v", len(hits), hits)
	}
	if !strings.HasPrefix(hits[0].Path, "cli/") || hits[0].Line != 2 {
		t.Errorf("wrong hit: %+v", hits[0])
	}
	if !strings.Contains(strings.Join(searched, " "), "cli") {
		t.Errorf("cli/ must be searched and reported, got %v", searched)
	}
}

// A root that does not exist must not be searched OR claimed — the output is
// read as coverage, so naming a directory that was never walked is a lie about
// what the zero result covers.
func TestSearchDecidedReportsOnlyTheRootsThatExist(t *testing.T) {
	root := seedDecided(t, map[string]string{".atl/wiki/a.md": "hello\n"})
	_, searched := searchDecided(root, "hello")
	if len(searched) != 1 || searched[0] != ".atl/wiki" {
		t.Errorf("only the existing root may be claimed, got %v", searched)
	}
}

// The empty result is the whole point, and it must not overstate itself. "0
// matches" is a fact about the query; a decision written in other words is
// indistinguishable from one nobody made.
func TestRenderDecidedDoesNotClaimNoDecisionExists(t *testing.T) {
	out := renderDecided("brief and stop", nil, []string{".atl/docs", "cli"})
	for _, want := range []string{"0 matches", "searched: .atl/docs cli", "about this QUERY", "synonyms"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{"no decision exists", "was never decided", "nothing was decided"} {
		if strings.Contains(strings.ToLower(out), forbidden) {
			t.Errorf("must not assert %q — a text search cannot know that:\n%s", forbidden, out)
		}
	}
}

// A project with none of the surfaces gets told so, rather than a "0 matches"
// that reads as a searched-and-found-nothing answer.
func TestRenderDecidedSaysWhenThereIsNothingToSearch(t *testing.T) {
	if out := renderDecided("x", nil, nil); !strings.Contains(out, "no decision surfaces here") {
		t.Errorf("got %q", out)
	}
}

// Vendored and generated trees are skipped: a hit inside node_modules is noise
// in a tool whose empty result is meant to be read as an answer.
func TestSearchDecidedSkipsVendoredTrees(t *testing.T) {
	root := seedDecided(t, map[string]string{
		"cli/node_modules/pkg/readme.md": "we decided everything\n",
		"cli/vendor/x/doc.md":            "we decided everything\n",
		".claude/worktrees/w/a.md":       "we decided everything\n",
		".atl/docs/real.md":              "we decided everything\n",
	})
	hits, _ := searchDecided(root, "we decided everything")
	if len(hits) != 1 || !strings.HasPrefix(hits[0].Path, ".atl/docs/") {
		t.Errorf("only the real decision surface may match, got %+v", hits)
	}
}

// Long result sets truncate — a tally nobody scrolls is a tally nobody reads.
func TestRenderDecidedTruncatesLongResults(t *testing.T) {
	var hits []decidedHit
	for i := 0; i < 45; i++ {
		hits = append(hits, decidedHit{Path: "a.md", Line: i + 1, Text: "match"})
	}
	if out := renderDecided("q", hits, []string{"a"}); !strings.Contains(out, "… 5 more") {
		t.Errorf("want a truncation note, got:\n%s", out)
	}
}

// The command end to end. The glue between "where am I" and "what did the search
// find" is what a refactor breaks silently, since both halves keep passing their
// own tests — the "correct body nothing calls" shape.
func TestDecidedCommandSearchesTheCurrentProject(t *testing.T) {
	root := seedDecided(t, map[string]string{
		".atl/docs/a-decision.md": "we settled on the two-branch flow\n",
	})
	t.Chdir(root)

	var out bytes.Buffer
	decidedCmd.SetOut(&out)
	if err := decidedCmd.RunE(decidedCmd, []string{"two-branch", "flow"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "1 matches") || !strings.Contains(got, "a-decision.md") {
		t.Errorf("multi-word query must be joined and matched, got:\n%s", got)
	}
}

// The zero path, through the command rather than the renderer — this is the one
// a human runs when they are about to assert a decision.
func TestDecidedCommandReportsZeroThroughTheCommand(t *testing.T) {
	t.Chdir(seedDecided(t, map[string]string{".atl/docs/x.md": "unrelated\n"}))
	var out bytes.Buffer
	decidedCmd.SetOut(&out)
	if err := decidedCmd.RunE(decidedCmd, []string{"nothing like this"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "0 matches") || !strings.Contains(out.String(), "about this QUERY") {
		t.Errorf("got:\n%s", out.String())
	}
}

// A project with no decision surfaces at all — the command must say so rather
// than print a zero that reads as searched-and-found-nothing.
func TestDecidedCommandOnAProjectWithNoSurfaces(t *testing.T) {
	t.Chdir(t.TempDir())
	var out bytes.Buffer
	decidedCmd.SetOut(&out)
	if err := decidedCmd.RunE(decidedCmd, []string{"anything"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "no decision surfaces here") {
		t.Errorf("got %q", out.String())
	}
}

// Only text a human wrote is searched: a match inside a binary or an unknown
// extension is noise in a tool whose empty result is meant to be read.
func TestSearchDecidedOnlySearchesTextExtensions(t *testing.T) {
	root := seedDecided(t, map[string]string{
		".atl/docs/real.md":  "the marker phrase\n",
		".atl/docs/blob.bin": "the marker phrase\n",
		".atl/docs/img.png":  "the marker phrase\n",
	})
	hits, _ := searchDecided(root, "the marker phrase")
	if len(hits) != 1 || !strings.HasSuffix(hits[0].Path, "real.md") {
		t.Errorf("only text extensions may match, got %+v", hits)
	}
}

// Matching is case-insensitive on both sides: the record's wording and the
// caller's rarely agree on capitalisation, and a miss here reads as "no decision".
func TestSearchDecidedIsCaseInsensitive(t *testing.T) {
	root := seedDecided(t, map[string]string{".atl/docs/d.md": "We DECIDED the Branch Pair\n"})
	if hits, _ := searchDecided(root, "we decided the branch pair"); len(hits) != 1 {
		t.Errorf("case must not decide a match, got %d", len(hits))
	}
}

// Results are ordered by path then line, so two runs over the same tree read the
// same — a search whose output reshuffles is one nobody trusts to diff.
func TestSearchDecidedOrdersStably(t *testing.T) {
	root := seedDecided(t, map[string]string{
		".atl/docs/b.md": "hit\nhit\n",
		".atl/docs/a.md": "hit\n",
		".atl/wiki/c.md": "hit\n",
	})
	hits, _ := searchDecided(root, "hit")
	var got []string
	for _, h := range hits {
		got = append(got, h.Path)
	}
	want := []string{".atl/docs/a.md", ".atl/docs/b.md", ".atl/docs/b.md", ".atl/wiki/c.md"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// A very long matched line is truncated: the result is scanned, and one runaway
// minified line would push everything after it off the screen.
func TestRenderDecidedTruncatesALongLine(t *testing.T) {
	long := strings.Repeat("x", 300)
	out := renderDecided("x", []decidedHit{{Path: "a.md", Line: 1, Text: long}}, []string{"a"})
	if !strings.Contains(out, "…") {
		t.Errorf("a 300-char line must be truncated, got:\n%s", out)
	}
	if strings.Contains(out, strings.Repeat("x", 200)) {
		t.Error("the full line leaked through the truncation")
	}
}

// An unreadable file is skipped, not fatal. This search is best-effort by
// design: one permission error must not turn an exhaustive sweep into a zero
// result, which is the reading that would do real damage.
func TestSearchDecidedSkipsUnreadableFiles(t *testing.T) {
	root := seedDecided(t, map[string]string{
		".atl/docs/locked.md":   "the marker phrase\n",
		".atl/docs/readable.md": "the marker phrase\n",
	})
	locked := filepath.Join(root, ".atl", "docs", "locked.md")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Skip("cannot chmod here:", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o644) })

	hits, searched := searchDecided(root, "the marker phrase")
	if len(searched) == 0 {
		t.Fatal("the search must still report what it covered")
	}
	var readable bool
	for _, h := range hits {
		if strings.HasSuffix(h.Path, "readable.md") {
			readable = true
		}
	}
	if !readable {
		t.Errorf("an unreadable sibling must not suppress a real hit, got %+v", hits)
	}
}
