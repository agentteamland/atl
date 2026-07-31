package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newPinFixture writes a project with a brain-storms dir and a CLAUDE.md whose
// pin block is the given raw body, and returns (claudeMD, dir).
func newPinFixture(t *testing.T, pinBody string, files map[string]string) (string, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".atl", "brain-storms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	claudeMD := filepath.Join(root, "CLAUDE.md")
	body := "# Project\n\nIntro.\n"
	if pinBody != "" {
		body += "\n" + pinBlockStart + "\n" + pinBody + "\n" + pinBlockEnd + "\n"
	}
	if err := os.WriteFile(claudeMD, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return claudeMD, dir
}

func brainstorm(status, body string) string {
	return "---\nstatus: " + status + "\nscope: project\ndate: 2026-07-31\n---\n\n# Topic\n\n" + body + "\n"
}

func TestScanPinsStalePin(t *testing.T) {
	claudeMD, dir := newPinFixture(t,
		"- **[shipped](.atl/brain-storms/shipped.md)** (project, 2026-07-01) — a decision that closed",
		map[string]string{"shipped.md": brainstorm("completed", "Final Decisions.")})

	got := ScanPins(claudeMD, dir)
	if len(got) != 1 || got[0].Kind != StalePin || got[0].Name != "shipped.md" || got[0].Status != "completed" {
		t.Fatalf("want one stale pin for shipped.md, got %+v", got)
	}
}

// Every closed-status spelling the real corpora use must be caught — the skill
// writes "completed", but hand-closed brainstorms use closed / closed-deferred /
// done / settled, and hardcoding one spelling silently misses the rest.
func TestScanPinsStalePinAcrossClosedSpellings(t *testing.T) {
	for _, status := range []string{"completed", "closed", "closed-absorbed", "closed-deferred", "done", "settled", "COMPLETED"} {
		claudeMD, dir := newPinFixture(t,
			"- **[x](.atl/brain-storms/x.md)** — pinned",
			map[string]string{"x.md": brainstorm(status, "body")})
		if got := ScanPins(claudeMD, dir); len(got) != 1 || got[0].Kind != StalePin {
			t.Errorf("status %q: want a stale pin, got %+v", status, got)
		}
	}
}

func TestScanPinsMissingPin(t *testing.T) {
	// No pin block at all — the recovery case the brainstorm rule describes.
	claudeMD, dir := newPinFixture(t, "",
		map[string]string{"open.md": brainstorm("active", "Open Items.")})

	got := ScanPins(claudeMD, dir)
	if len(got) != 1 || got[0].Kind != MissingPin || got[0].Name != "open.md" {
		t.Fatalf("want one missing pin for open.md, got %+v", got)
	}
}

func TestScanPinsClean(t *testing.T) {
	claudeMD, dir := newPinFixture(t,
		"- **[open](.atl/brain-storms/open.md)** (project, 2026-07-31) — in progress",
		map[string]string{
			"open.md":   brainstorm("active", "Open Items."),
			"closed.md": brainstorm("completed", "Final Decisions."),
		})

	if got := ScanPins(claudeMD, dir); len(got) != 0 {
		t.Fatalf("pinned-active + unpinned-closed is the correct state, got drift %+v", got)
	}
}

// The trap core/rules/brainstorm.md names explicitly: a brainstorm legitimately
// quotes the literal string "status: active" in its prose, so reading the body
// instead of the frontmatter invents an active brainstorm and reports it as
// unpinned — a false positive on entirely legitimate content. Each shape below
// is one a body scan gets wrong and the fenced read gets right; the last is the
// real corpus's own shape (a spec draft carrying no frontmatter at all).
func TestScanPinsReadsFrontmatterNotBody(t *testing.T) {
	files := map[string]string{
		"closed-quoting.md": brainstorm("completed",
			"We opened this with `status: active` in the frontmatter.\n\nstatus: active\n"),
		"no-status-key.md": "---\nscope: project\ndate: 2026-07-31\n---\n\n# Topic\n\nstatus: active\n",
		"no-frontmatter.md": "# Detail-design specification (DRAFT)\n\n" +
			"The frontmatter a brainstorm starts with reads:\n\nstatus: active\n",
	}
	claudeMD, dir := newPinFixture(t, "", files)

	if got := ScanPins(claudeMD, dir); len(got) != 0 {
		t.Fatalf("prose quoting \"status: active\" is not an active brainstorm, got %+v", got)
	}
	for name, want := range map[string]string{
		"closed-quoting.md": "completed",
		"no-status-key.md":  "",
		"no-frontmatter.md": "",
	} {
		if s := frontmatterStatus(filepath.Join(dir, name)); s != want {
			t.Errorf("frontmatterStatus(%s) = %q, want %q", name, s, want)
		}
	}
}

// Real projects hand-write the pin block as prose paragraphs rather than the
// skill's bullet list. Tying the parser to the `- **[x](y)**` shape reports
// every one of their pins as missing.
func TestScanPinsParagraphBlockFormat(t *testing.T) {
	claudeMD, dir := newPinFixture(t,
		"**Aktif brainstorm:** [Rota tabanlı sipariş](.atl/brain-storms/routes.md) — uzun bir özet.\n\n"+
			"**Aktif brainstorm:** [Video teslimi](.atl/brain-storms/video.md) (2026-06-02) — ikinci konu.",
		map[string]string{
			"routes.md": brainstorm("active", "body"),
			"video.md":  brainstorm("active", "body"),
		})

	if got := ScanPins(claudeMD, dir); len(got) != 0 {
		t.Fatalf("a paragraph-shaped pin block still pins both brainstorms, got drift %+v", got)
	}
}

// A bullet may cite an issue, a PR, or a wiki page alongside its pin. Only links
// resolving into the brain-storms directory are pins.
func TestScanPinsIgnoresNonBrainstormLinks(t *testing.T) {
	claudeMD, dir := newPinFixture(t,
		"- **[open](.atl/brain-storms/open.md)** — see [atl#332](https://github.com/agentteamland/atl/issues/332) "+
			"and [the wiki page](.atl/wiki/pins.md)",
		map[string]string{"open.md": brainstorm("active", "body")})

	if got := ScanPins(claudeMD, dir); len(got) != 0 {
		t.Fatalf("non-brainstorm links are not pins, got drift %+v", got)
	}
}

// An unrecognized status is left alone rather than reported: a hand-written
// "paused" pin is legitimate content, and firing on it is how a detector gets
// turned off. Same for a file with no frontmatter at all (the corpus has one).
func TestScanPinsLeavesUnknownStatusAlone(t *testing.T) {
	claudeMD, dir := newPinFixture(t,
		"- **[paused](.atl/brain-storms/paused.md)** — on hold\n- **[draft](.atl/brain-storms/draft.md)** — a spec draft",
		map[string]string{
			"paused.md": brainstorm("paused", "body"),
			"draft.md":  "# Draft spec\n\nNo frontmatter at all.\n",
		})

	if got := ScanPins(claudeMD, dir); len(got) != 0 {
		t.Fatalf("unknown/absent status must not be reported, got drift %+v", got)
	}
}

// A pin link may carry a fragment, a markdown title, or angle brackets. Each of
// those shapes hides the path from a parser that demands `)` right after it, so
// a correctly-pinned brainstorm reads as unpinned — a false positive on content
// that is exactly right.
func TestScanPinsLinkTargetShapes(t *testing.T) {
	for _, pin := range []string{
		"- [x](.atl/brain-storms/x.md)",
		"- [x](.atl/brain-storms/x.md#final-decisions)",
		"- [x](.atl/brain-storms/x.md \"the topic\")",
		"- [x](<.atl/brain-storms/x.md>)",
		"- [x](./.atl/brain-storms/x.md)",
	} {
		claudeMD, dir := newPinFixture(t, pin, map[string]string{"x.md": brainstorm("active", "body")})
		if got := ScanPins(claudeMD, dir); len(got) != 0 {
			t.Errorf("pin %q is a pin, got drift %+v", pin, got)
		}
		// …and the same shape must still catch the stale direction.
		claudeMD, dir = newPinFixture(t, pin, map[string]string{"x.md": brainstorm("completed", "body")})
		if got := ScanPins(claudeMD, dir); len(got) != 1 || got[0].Kind != StalePin {
			t.Errorf("pin %q on a closed brainstorm is stale, got %+v", pin, got)
		}
	}
}

// The global scope, pinned exactly the way core/skills/brainstorm/SKILL.md step 5
// documents it: `brain-storms/x.md` relative to ~/.claude/CLAUDE.md, while the
// brainstorm itself lives in ~/.atl/brain-storms. Requiring the link to resolve
// into that exact directory reports every correctly-pinned global brainstorm as
// unpinned, forever, in every session.
func TestScanPinsGlobalScopeLinkShapes(t *testing.T) {
	for _, pin := range []string{
		"- **[life](brain-storms/life.md)** (global, 2026-08-01) — the skill's documented shape",
		"- **[life](../.atl/brain-storms/life.md)** (global, 2026-08-01) — the resolvable shape",
	} {
		home := t.TempDir()
		dir := filepath.Join(home, ".atl", "brain-storms")
		claude := filepath.Join(home, ".claude")
		for _, d := range []string{dir, claude} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, "life.md"), []byte(brainstorm("active", "body")), 0o644); err != nil {
			t.Fatal(err)
		}
		claudeMD := filepath.Join(claude, "CLAUDE.md")
		if err := os.WriteFile(claudeMD, []byte("# Global\n\n"+pinBlockStart+"\n"+pin+"\n"+pinBlockEnd+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := ScanPins(claudeMD, dir); len(got) != 0 {
			t.Errorf("global pin %q is a pin, got drift %+v", pin, got)
		}
	}
}

// A block that opened and never closed makes the pin set unknowable. Reading to
// EOF instead turns every brainstorm link further down the file into a bogus
// stale-pin report — and a dropped end marker is itself a half-finished closure,
// the exact failure this check exists for.
func TestScanPinsUnterminatedBlock(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".atl", "brain-storms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"shipped-a.md", "shipped-b.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(brainstorm("completed", "body")), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	claudeMD := filepath.Join(root, "CLAUDE.md")
	body := "# P\n\n" + pinBlockStart + "\n- nothing active right now\n\n" +
		"## Settled decisions\n\n" +
		"- [a](.atl/brain-storms/shipped-a.md) — shipped\n" +
		"- [b](.atl/brain-storms/shipped-b.md) — shipped\n"
	if err := os.WriteFile(claudeMD, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got := ScanPins(claudeMD, dir)
	if len(got) != 1 || got[0].Kind != UnterminatedBlock {
		t.Fatalf("an unterminated block must be reported as such, not as %d stale pins: %+v", len(got), got)
	}
	if got[0].Name != claudeMD {
		t.Errorf("the report must name the file to fix, got %q", got[0].Name)
	}
}

func TestScanPinsNoBrainstormSurface(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# Project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ScanPins(filepath.Join(root, "CLAUDE.md"), filepath.Join(root, ".atl", "brain-storms")); got != nil {
		t.Fatalf("a project with no brain-storms dir must be silent, got %+v", got)
	}
}

func TestBrainstormPinCheck(t *testing.T) {
	// Point the global scope at an empty home so only the project scope is live.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	root := t.TempDir()
	dir := filepath.Join(root, ".atl", "brain-storms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeClaude := func(pin string) {
		body := "# P\n\n" + pinBlockStart + "\n" + pin + "\n" + pinBlockEnd + "\n"
		if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("open.md", brainstorm("active", "body"))
	writeClaude("- **[open](.atl/brain-storms/open.md)** — in progress")
	if r := BrainstormPinCheck(root)(); r.Status != OK {
		t.Fatalf("reconciled pins should be OK, got %v (%s)", r.Status, r.Detail)
	}

	// Close it without touching the pin — exactly the drift the check exists for.
	write("open.md", brainstorm("completed", "Final Decisions."))
	r := BrainstormPinCheck(root)()
	if r.Status != Warn {
		t.Fatalf("a stale pin should Warn, got %v (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "open.md") {
		t.Fatalf("detail must name the drifted file, got %q", r.Detail)
	}
	if r.Healed {
		t.Fatal("the check reports drift; it must never rewrite the user's CLAUDE.md")
	}
	// Warn, never Fail: `atl doctor` exits non-zero on Fail, and drift in a user
	// document must not gate a script or block a session hook.
	if r.Status == Fail {
		t.Fatal("pin drift must not be Fail")
	}
}
