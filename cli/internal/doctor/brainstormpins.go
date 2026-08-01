package doctor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The deterministic half of the brainstorm two-signal discipline
// (core/rules/brainstorm.md): CLAUDE.md's `<!-- brainstorm:active -->` pin block
// must agree with each brainstorm file's frontmatter `status:`.
//
// `/brainstorm done` is supposed to drop a closed brainstorm's bullet, but that
// step is prose in an LLM skill — it gets skipped when a closure runs long, when
// a brainstorm is closed by hand, or when a session ends mid-closure, and until
// now nothing noticed. That block is loaded into EVERY session's context by
// design, so a stale bullet does not sit there harmlessly: it actively tells
// future sessions that a closed decision is still open.
//
// REPORT, NOT SELF-HEAL. The doctor's existing self-heals only touch ATL-owned
// artifacts — a missing installed file re-fetched from its pinned source, a
// dropped hook re-bound by an idempotent installer. CLAUDE.md is the user's own
// always-loaded instruction file, and rewriting it deterministically is a
// different class of act. Self-healing would also cover only one direction:
// removing a stale bullet is mechanical, but *writing* a missing one needs a
// one-line summary of the topic, which is judgment. So the check reports both
// directions in one voice (plus a block it cannot read at all) — and because
// session-start stdout lands in Claude's context, the report is itself the
// trigger to fix it with the skill that owns the file.

const (
	pinBlockStart = "<!-- brainstorm:active:start -->"
	pinBlockEnd   = "<!-- brainstorm:active:end -->"
)

// mdLinkTarget matches the target of a markdown link, e.g. `](.atl/brain-storms/x.md)`.
// The closing paren is deliberately NOT required: a link may carry a title
// (`](x.md "topic")`) or wrap its target in angle brackets (`](<x.md>)`), and
// demanding `)` right after the path silently matches neither — which reads as
// "not pinned" and reports a correctly-pinned brainstorm as drift.
var mdLinkTarget = regexp.MustCompile(`\]\(\s*<?([^)>\s]+)`)

// DriftKind classifies one disagreement.
type DriftKind int

const (
	StalePin          DriftKind = iota // pinned, but the brainstorm's frontmatter says it closed
	MissingPin                         // active, but nothing in the block pins it
	UnterminatedBlock                  // the pin block opened and never closed — pins unreadable
)

// PinDrift is one disagreement between the pin block and a brainstorm's frontmatter.
type PinDrift struct {
	Name   string // brainstorm file name ("atl-v2.md"); the CLAUDE.md path for UnterminatedBlock
	Status string // its frontmatter status
	Kind   DriftKind
}

// ScanPins compares one scope's pin block against its brainstorms' frontmatter.
// claudeMD is that scope's CLAUDE.md, dir its brain-storms directory. A scope
// with no brain-storms directory has nothing to check and yields nothing.
func ScanPins(claudeMD, dir string) []PinDrift {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // no brainstorm surface in this scope — stay silent
	}
	pinned, ok := pinnedBrainstorms(claudeMD, dir)
	if !ok {
		// The block opened and never closed, so where it ends is a guess. Say that
		// instead of guessing: a CLAUDE.md that links its closed brainstorms further
		// down turns every one of those links into a bogus stale-pin report —
		// measured at 22 on the maintainer workspace's own CLAUDE.md with the end
		// marker dropped. And that state is not hypothetical: a half-finished
		// closure losing the end marker is the very failure this check exists for.
		return []PinDrift{{Name: claudeMD, Kind: UnterminatedBlock}}
	}
	var drift []PinDrift
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		status := frontmatterStatus(filepath.Join(dir, name))
		switch {
		case pinned[name] && isClosedStatus(status):
			drift = append(drift, PinDrift{Name: name, Status: status, Kind: StalePin})
		case !pinned[name] && status == "active":
			drift = append(drift, PinDrift{Name: name, Status: status, Kind: MissingPin})
		}
	}
	return drift
}

// pinnedBrainstorms returns the brainstorm files the CLAUDE.md pin block links
// to. Only links resolving into a brain-storms directory count — a bullet may
// also cite an issue, a PR, or a wiki page, and none of those are pins. ok is
// false when the block opened but never closed, i.e. the pin set is unknowable.
//
// Deliberately format-agnostic: it reads link targets, not the bullet shape the
// skill documents. Real projects carry hand-written pin blocks that are
// paragraphs rather than a bullet list, and a parser tied to the `- **[x](y)**`
// shape would report every one of their pins as missing.
func pinnedBrainstorms(claudeMD, dir string) (map[string]bool, bool) {
	pinned := map[string]bool{}
	b, err := os.ReadFile(claudeMD)
	if err != nil {
		return pinned, true // no CLAUDE.md → nothing is pinned
	}
	body := string(b)
	i := strings.Index(body, pinBlockStart)
	if i < 0 {
		return pinned, true
	}
	block := body[i+len(pinBlockStart):]
	j := strings.Index(block, pinBlockEnd)
	if j < 0 {
		return pinned, false
	}
	block = block[:j]
	base := filepath.Dir(claudeMD)
	dir = filepath.Clean(dir)
	for _, m := range mdLinkTarget.FindAllStringSubmatch(block, -1) {
		if name, isPin := pinTarget(m[1], base, dir); isPin {
			pinned[name] = true
		}
	}
	return pinned, true
}

// pinTarget resolves one markdown link target against the CLAUDE.md's own
// directory and reports the brainstorm file it pins, if any.
//
// A link into ANY directory named brain-storms counts, not only into dir — the
// global pin the brainstorm skill documents is `brain-storms/x.md` relative to
// ~/.claude/CLAUDE.md, which resolves to ~/.claude/brain-storms, while global
// brainstorms live in ~/.atl/brain-storms. Requiring an exact match reports
// every global brainstorm pinned exactly as instructed as unpinned — a false
// positive on correct content, every session, which is how a detector gets
// switched off. Nothing is invented by the widening: only names that exist in
// dir are ever looked up.
func pinTarget(target, base, dir string) (string, bool) {
	if i := strings.IndexAny(target, "#?"); i >= 0 {
		target = target[:i] // a fragment or query is not part of the path
	}
	if target == "" {
		return "", false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(base, target)
	}
	d, f := filepath.Split(filepath.Clean(target))
	d = filepath.Clean(d)
	if d == dir || filepath.Base(d) == "brain-storms" {
		return f, true
	}
	return "", false
}

// frontmatterStatus reads `status:` from the leading `---` fenced frontmatter,
// and only from there. The body is never matched: a correctly-closed brainstorm
// legitimately quotes the literal string "status: active" in its prose, so a
// body grep flags closed brainstorms as active — the trap core/rules/brainstorm.md
// names explicitly. Returns "" when there is no frontmatter or no status line.
func frontmatterStatus(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() || strings.TrimSpace(sc.Text()) != "---" {
		return "" // no frontmatter
	}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "---" {
			return "" // frontmatter closed without a status
		}
		if v, ok := strings.CutPrefix(line, "status:"); ok {
			return strings.ToLower(strings.Trim(strings.TrimSpace(v), `"'`))
		}
	}
	return ""
}

// isClosedStatus reports whether a frontmatter status means the brainstorm is
// finished. A deliberately closed list, not "anything that isn't active": real
// corpora use completed / done / settled / closed-<reason>, but an unrecognized
// status (a hand-written "paused", a typo, an empty frontmatter) is left alone
// rather than reported. A detector that fires on legitimate content gets turned
// off, and a missed stale pin costs far less than a false one.
func isClosedStatus(status string) bool {
	switch status {
	case "completed", "done", "settled":
		return true
	}
	// closed / closed-absorbed / closed-deferred — the family is qualified with a
	// reason suffix.
	return strings.HasPrefix(status, "closed")
}

// BrainstormPinCheck is the doctor check for pin/frontmatter drift, over both
// scopes the brainstorm rule names: the project's .atl/brain-storms and the
// global ~/.atl/brain-storms.
func BrainstormPinCheck(projectRoot string) Check {
	return func() Result {
		var stale, missing, broken []string
		for _, s := range pinScopes(projectRoot) {
			for _, d := range ScanPins(s.claudeMD, s.dir) {
				switch d.Kind {
				case StalePin:
					stale = append(stale, fmt.Sprintf("%s (status: %s)", d.Name, d.Status))
				case MissingPin:
					missing = append(missing, d.Name)
				case UnterminatedBlock:
					broken = append(broken, d.Name)
				}
			}
		}
		var parts []string
		if len(broken) > 0 {
			parts = append(parts, fmt.Sprintf("the pin block in %s opened but never closed — its pins can't be read",
				strings.Join(broken, ", ")))
		}
		if len(stale) > 0 {
			parts = append(parts, fmt.Sprintf("%d closed brainstorm(s) still pinned in CLAUDE.md — every session is told the decision is open: %s",
				len(stale), strings.Join(stale, ", ")))
		}
		if len(missing) > 0 {
			parts = append(parts, fmt.Sprintf("%d active brainstorm(s) with no pin — future sessions won't see them: %s",
				len(missing), strings.Join(missing, ", ")))
		}
		if len(parts) > 0 {
			return Result{Name: "brainstorm-pins", Status: Warn,
				Detail: strings.Join(parts, "; ") + " — fix the `<!-- brainstorm:active -->` block (brainstorm rule)"}
		}
		return Result{Name: "brainstorm-pins", Status: OK, Detail: "pins agree with brainstorm frontmatter"}
	}
}

// pinScopes pairs each scope's CLAUDE.md with its brain-storms directory. The
// project pin lives at the project root; the global one in ~/.claude, while its
// brainstorms live in ~/.atl — the split the brainstorm skill writes to.
func pinScopes(projectRoot string) []struct{ claudeMD, dir string } {
	var scopes []struct{ claudeMD, dir string }
	if projectRoot != "" {
		scopes = append(scopes, struct{ claudeMD, dir string }{
			filepath.Join(projectRoot, "CLAUDE.md"),
			filepath.Join(projectRoot, ".atl", "brain-storms"),
		})
	}
	if home, err := os.UserHomeDir(); err == nil {
		scopes = append(scopes, struct{ claudeMD, dir string }{
			filepath.Join(home, ".claude", "CLAUDE.md"),
			filepath.Join(home, ".atl", "brain-storms"),
		})
	}
	return scopes
}
