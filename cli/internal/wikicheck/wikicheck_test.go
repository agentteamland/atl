package wikicheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// project builds a minimal healthy corpus: a CLAUDE.md linking every wiki page,
// and pages whose cross-links resolve.
func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	wiki := filepath.Join(root, ".atl", "wiki")
	if err := os.MkdirAll(wiki, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(wiki, "alpha.md"), "# Alpha\n\nSee [beta](beta.md).\n")
	write(t, filepath.Join(wiki, "beta.md"), "# Beta\n\nNothing here links out.\n")
	write(t, filepath.Join(root, "CLAUDE.md"),
		"# Project\n\n- [Alpha](.atl/wiki/alpha.md) — a page\n- [Beta](.atl/wiki/beta.md) — another\n")
	return root
}

func write(t *testing.T, p, body string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func checks(f []Finding) string {
	var s []string
	for _, x := range f {
		s = append(s, x.Check+":"+x.Path+" — "+x.Detail)
	}
	return strings.Join(s, "\n")
}

// A healthy corpus is silent. Every assertion below is a mutation of this state,
// which is the only thing that distinguishes a check that works from one that
// cannot fire — all three of these report zero on the real corpus today, so
// "green" and "vacuous" would otherwise be the same observation.
func TestHealthyCorpusIsSilent(t *testing.T) {
	if got := RunAll(Input{ProjectRoot: project(t)}); len(got) != 0 {
		t.Fatalf("healthy corpus produced findings:\n%s", checks(got))
	}
}

func TestIndexPointingAtADeletedPage(t *testing.T) {
	root := project(t)
	if err := os.Remove(filepath.Join(root, ".atl", "wiki", "beta.md")); err != nil {
		t.Fatal(err)
	}
	got := RunAll(Input{ProjectRoot: root})
	if len(got) == 0 || !strings.Contains(checks(got), "index-targets") {
		t.Fatalf("a deleted page left an index entry and nothing reported it:\n%s", checks(got))
	}
}

// The retrieval index has been observed serving a page deleted the night before,
// and the only tell was a blank excerpt. This is the check that would have said so.
func TestUnreachablePageIsReported(t *testing.T) {
	root := project(t)
	write(t, filepath.Join(root, ".atl", "wiki", "orphan.md"), "# Orphan\n")
	got := RunAll(Input{ProjectRoot: root})
	if !strings.Contains(checks(got), "reachability:.atl/wiki/orphan.md") {
		t.Fatalf("a page nothing links to went unreported:\n%s", checks(got))
	}
}

func TestBrokenCrossLinkIsReported(t *testing.T) {
	root := project(t)
	write(t, filepath.Join(root, ".atl", "wiki", "alpha.md"), "# Alpha\n\nSee [gone](gone.md).\n")
	got := RunAll(Input{ProjectRoot: root})
	if !strings.Contains(checks(got), "links:.atl/wiki/alpha.md") {
		t.Fatalf("a broken cross-link went unreported:\n%s", checks(got))
	}
}

// The two false-positive traps, both measured at 100% of a naive checker's
// findings on the real corpus. This corpus contains a page documenting that
// `atl docs check` has exactly this bug, so repeating it here would be
// self-refuting — these two cases are the pin against that.
func TestIllustrativeLinksAreNotFindings(t *testing.T) {
	root := project(t)
	write(t, filepath.Join(root, ".atl", "wiki", "alpha.md"), strings.Join([]string{
		"# Alpha",
		"",
		"Write the entry as `- [Title](.atl/wiki/topic.md) — hook`, an inline example.",
		"",
		"```markdown",
		"- [source](path/to/brainstorm.md)",
		"- [Details](children/topic-1.md)",
		"```",
		"",
		"And an elided one: [source](.atl/wiki/...).",
		"",
		"See [beta](beta.md).",
	}, "\n"))
	if got := RunAll(Input{ProjectRoot: root}); len(got) != 0 {
		t.Fatalf("illustrative link syntax was reported as broken:\n%s", checks(got))
	}
}

// A link may carry a fragment or a title; neither is part of the path.
func TestFragmentsAndTitlesResolve(t *testing.T) {
	root := project(t)
	write(t, filepath.Join(root, ".atl", "wiki", "alpha.md"),
		"# Alpha\n\n[a](beta.md#some-heading) and [b](beta.md \"a title\").\n")
	if got := RunAll(Input{ProjectRoot: root}); len(got) != 0 {
		t.Fatalf("a fragment or title was treated as part of the path:\n%s", checks(got))
	}
}

// External and absolute targets are not this checker's business.
func TestExternalTargetsAreIgnored(t *testing.T) {
	root := project(t)
	write(t, filepath.Join(root, ".atl", "wiki", "alpha.md"),
		"# Alpha\n\n[x](https://example.com/a.md) [y](/etc/hosts) [z](mailto:a@b.c) [w](beta.md)\n")
	if got := RunAll(Input{ProjectRoot: root}); len(got) != 0 {
		t.Fatalf("an external target was reported:\n%s", checks(got))
	}
}

// A project with no wiki is not a broken project — the checks are simply not
// applicable, and must not invent findings from an absent directory.
func TestNoWikiIsSilent(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "CLAUDE.md"), "# Project\n")
	if got := RunAll(Input{ProjectRoot: root}); len(got) != 0 {
		t.Fatalf("a project with no wiki produced findings:\n%s", checks(got))
	}
}
