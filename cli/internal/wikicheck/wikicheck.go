// Package wikicheck holds the deterministic integrity checks for a project's
// `.atl/wiki` — the current-truth knowledge layer.
//
// # What this package is NOT
//
// It is not a drift gate, and saying so is load-bearing. The docs site can be
// checked for drift because it has a ground truth to compare against — the code.
// A wiki page's claim ("this repo is public", "the store has no remote") has no
// single referent, so *correctness* here is not mechanically decidable, and a
// check that pretends otherwise manufactures the false coverage this corpus has
// measured elsewhere.
//
// That was tested rather than assumed. The obvious candidate — "does a repo path
// cited by a page still exist?" — was measured at 76-90% false positive over the
// real corpus, and the cause is structural, not tunable: a knowledge corpus's
// genre is largely *documenting that X is dead*, which is byte-identical to
// *citing a dead X*. The check fires hardest on the pages that did the correction
// work best. It is deliberately absent.
//
// # What it is
//
// Two honest jobs:
//
//  1. A regression guard on **discoverability and link integrity**. A page absent
//     from the CLAUDE.md index is not merely harder to find — the index is what a
//     generated retrieval query draws its vocabulary from, measured at 100% vs 58%
//     recall@5 index-blind. An unindexed page is outside the vocabulary the
//     consult mechanism was measured to depend on.
//  2. A **prioritised target list** for the LLM half (the /observe sweep's
//     current-truth lens), which is where actual staleness is found.
//
// All three checks report zero findings on a healthy corpus, so their value is
// entirely in going red later. They are mutation-tested for that reason: a check
// that has never fired is indistinguishable from one that cannot.
package wikicheck

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Severity ranks a finding. Fail items are facts about files — a target that is
// not on disk, a page nothing links to. There is no Warn tier here on purpose:
// a check that cannot be certain does not belong in this package.
type Severity string

const Fail Severity = "fail"

// Finding is a single integrity item.
type Finding struct {
	Check    string // "index-targets" | "reachability" | "links"
	Severity Severity
	Path     string // the file the finding is about, relative to the project root
	Detail   string
}

// Input is what the checks need: the project root holding .atl/wiki and CLAUDE.md.
type Input struct {
	ProjectRoot string
}

// mdLink matches a markdown link target. Fenced blocks and inline code spans are
// stripped before this runs — see stripCode, and the tests that pin why.
var mdLink = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)

// RunAll runs every check and returns the findings, sorted for stable output.
func RunAll(in Input) []Finding {
	var out []Finding
	out = append(out, IndexTargets(in)...)
	out = append(out, Reachability(in)...)
	out = append(out, Links(in)...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Check != out[j].Check {
			return out[i].Check < out[j].Check
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// IndexTargets reports an entry in CLAUDE.md pointing at a .atl/wiki page that is
// not on disk — the deleted-page case. It matters beyond tidiness: the retrieval
// index has been observed serving a page deleted the night before, and the only
// tell was a blank excerpt.
func IndexTargets(in Input) []Finding {
	body, err := os.ReadFile(filepath.Join(in.ProjectRoot, "CLAUDE.md"))
	if err != nil {
		return nil
	}
	var out []Finding
	seen := map[string]bool{}
	for _, m := range mdLink.FindAllStringSubmatch(stripCode(string(body)), -1) {
		target := linkPath(m[1])
		if !strings.HasPrefix(target, ".atl/wiki/") || seen[target] {
			continue
		}
		seen[target] = true
		if _, err := os.Stat(filepath.Join(in.ProjectRoot, target)); err != nil {
			out = append(out, Finding{
				Check: "index-targets", Severity: Fail, Path: "CLAUDE.md",
				Detail: "links to " + target + ", which is not on disk",
			})
		}
	}
	return out
}

// Reachability reports a wiki page that CLAUDE.md does not link at all.
//
// The predicate is deliberately "linked anywhere in CLAUDE.md", not "has an entry
// in the wiki:index block". The stricter form was measured and is 100% false
// positive on the real corpus: its single finding was a page held outside the
// index on purpose and linked three times elsewhere in the same file. A check
// whose only finding is wrong is worse than no check.
func Reachability(in Input) []Finding {
	dir := filepath.Join(in.ProjectRoot, ".atl", "wiki")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	body, err := os.ReadFile(filepath.Join(in.ProjectRoot, "CLAUDE.md"))
	if err != nil {
		return nil
	}
	text := string(body)
	var out []Finding
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		rel := ".atl/wiki/" + e.Name()
		if strings.Contains(text, rel) {
			continue
		}
		out = append(out, Finding{
			Check: "reachability", Severity: Fail, Path: rel,
			Detail: "no link from CLAUDE.md — it is outside the vocabulary a generated retrieval query draws on",
		})
	}
	return out
}

// Links reports a relative link inside a wiki page whose target is not on disk.
//
// A cross-reference is a claim about the filesystem at the moment it was written,
// and nothing in the commit path re-checks it — a page can be reorganised out from
// under a link that was correct when authored, and the stale and the live form are
// indistinguishable in review.
//
// Two exclusions, both measured as 100% of the naive checker's findings:
// fenced blocks and inline code spans (illustrative placeholders like
// `[source](path/to/file.md)`), and any target containing an ellipsis. This corpus
// contains a page documenting that `atl docs check` has exactly this bug; three of
// the naive checker's nine findings came from that page, so repeating it here
// would be self-refuting.
func Links(in Input) []Finding {
	dir := filepath.Join(in.ProjectRoot, ".atl", "wiki")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Finding
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		for _, m := range mdLink.FindAllStringSubmatch(stripCode(string(body)), -1) {
			target := linkPath(m[1])
			if target == "" || seen[target] || !isLocal(target) {
				continue
			}
			seen[target] = true
			if _, err := os.Stat(filepath.Join(dir, target)); err != nil {
				out = append(out, Finding{
					Check: "links", Severity: Fail, Path: ".atl/wiki/" + e.Name(),
					Detail: "broken link to " + target,
				})
			}
		}
	}
	return out
}

// stripCode removes fenced blocks and inline code spans, in that order. Both hold
// illustrative link syntax that is not a claim about any real file.
func stripCode(s string) string {
	var b strings.Builder
	inFence := false
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			b.WriteString(stripSpans(line))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// stripSpans blanks `...` inline code runs, keeping the line's other content.
func stripSpans(line string) string {
	var b strings.Builder
	in := false
	for _, r := range line {
		if r == '`' {
			in = !in
			continue
		}
		if !in {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// linkPath trims a link target down to its path: no title, no fragment, no query.
func linkPath(raw string) string {
	t := strings.TrimSpace(raw)
	if i := strings.IndexAny(t, " \t"); i >= 0 {
		t = t[:i] // a link title after the path
	}
	if i := strings.IndexAny(t, "#?"); i >= 0 {
		t = t[:i]
	}
	return strings.TrimSpace(t)
}

// isLocal reports whether a target is a relative path this checker can resolve.
// Absolute paths, URLs, mail links and elided placeholders are all somebody
// else's problem — or nobody's.
func isLocal(t string) bool {
	switch {
	case t == "",
		strings.Contains(t, "://"),
		strings.HasPrefix(t, "/"),
		strings.HasPrefix(t, "mailto:"),
		strings.Contains(t, "..."),
		strings.Contains(t, "…"),
		strings.Contains(t, "<"),
		strings.Contains(t, "{"):
		return false
	}
	return true
}
