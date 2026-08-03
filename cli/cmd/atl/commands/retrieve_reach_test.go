package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The corpus boundary decided six measured failures, so it is pinned by a test
// rather than by a comment. Each row is a real refuting file from that set: the
// page whose absence let a claim stand, and where it actually lives.
//
// This goes red if the corpus is ever narrowed again — which is how it got here.
// Retrieval covered 36.7% of this repository's knowledge markdown and scored 0
// of 6, and for five of those six the page could not have been surfaced at all.
func TestCorpusDirsCoversTheKnowledgeThatRefutesClaims(t *testing.T) {
	root := t.TempDir()
	dirs, err := corpusDirs(root)
	if err != nil {
		t.Fatal(err)
	}
	// corpusDirs resolves the project layer under the project root, and the global
	// layer under HOME; both are absolute, so compare on prefixes.
	covers := func(rel string) bool {
		want := filepath.Join(root, filepath.FromSlash(rel))
		for _, d := range dirs {
			if want == d || strings.HasPrefix(want, d+string(filepath.Separator)) {
				return true
			}
		}
		return false
	}

	for _, tc := range []struct {
		file, why string
		want      bool
	}{
		{".atl/wiki/unenforceable-rule-not-written.md", "the rule broken hours after it was written", true},
		{".atl/journal/2026-08-03.md", "history", true},
		{".atl/docs/atl-v2.md", "states that promotion already runs automatically", true},
		{".claude/knowledge/pack-format.md", "states the area vocabulary is project-shaped", true},
		{".claude/agents/tech-lead/children/decomposition-blueprint.md", "'areas are project-shaped, NOT stack-shaped'", true},
		{".claude/packs/api/production-unit.md", "the stack craft a worker is told to load", true},
		{".claude/skills/work-start/SKILL.md", "the skill whose authored rationale was false", true},
		{".claude/backends/github/adapter.md", "the backend contract every operation follows", true},

		// Deliberately out. A brainstorm records rejected options by mandate, and a
		// chunk of one, split from the verdict that rejected it, reads exactly like
		// a decision — which is the failure shape this change exists to reduce.
		{".atl/brain-storms/delivery-drive-loop.md", "process, not current truth", false},
	} {
		if got := covers(tc.file); got != tc.want {
			verb := "must be reachable"
			if !tc.want {
				verb = "must NOT be indexed"
			}
			t.Errorf("%s %s — %s (got covered=%v)", tc.file, verb, tc.why, got)
		}
	}
}

// docs/ stays gated on the delivery marker: an ordinary repo's docs/ is often a
// vendored site tree that would flood the corpus.
func TestCorpusDirsAddsRepoDocsOnlyForADeliveryProject(t *testing.T) {
	root := t.TempDir()
	has := func() bool {
		dirs, err := corpusDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range dirs {
			if d == filepath.Join(root, "docs") {
				return true
			}
		}
		return false
	}
	if has() {
		t.Error("docs/ must not be indexed without a .delivery marker")
	}
	if err := os.MkdirAll(filepath.Join(root, ".delivery"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".delivery", "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !has() {
		t.Error("a delivery project keeps its durable knowledge in docs/ — it must be indexed")
	}
}
