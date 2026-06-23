// Package docscheck holds the deterministic documentation-drift checks for the
// AgentTeamLand v2 docs site. Every check here is LLM-free and zero-false-positive
// by construction: it compares the VitePress site against the code (the command
// tree, the skill directories, the file layout, a curated stale-token denylist).
// The semantic (prose-vs-behavior) half lives in the /docs-audit skill, not here —
// that is the CLI/Skill boundary (CLI = deterministic, Skill = LLM).
package docscheck

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Severity ranks a finding. Fail items break the CI gate; Warn items are
// surfaced but do not fail (best-effort checks that could otherwise mis-fire).
type Severity string

const (
	Fail Severity = "fail"
	Warn Severity = "warn"
)

// Finding is a single drift item.
type Finding struct {
	Check    string   // "coverage" | "parity" | "tokens" | "links" | "flags" | "external"
	Severity Severity
	Path     string // doc page relative to the site dir, or "" when not page-specific
	Detail   string
}

// Command is the documented surface of one CLI command.
type Command struct {
	Name  string   // cobra command name, e.g. "install"
	Flags []string // long flag names, e.g. {"global", "project"}
}

// Input bundles everything the deterministic checks need.
type Input struct {
	SiteDir  string    // <repo>/docs/site
	CoreDir  string    // <repo>/core (for skill coverage); "" to skip
	Commands []Command // the live cobra command tree
	Denylist []string  // stale tokens that must never appear
}

// DefaultDenylist lists stale INSTRUCTIONAL patterns that must never appear in the
// v2 docs as live instructions: install/usage commands for the retired channels.
// It is deliberately pattern-based, not bare-noun based. The bare nouns ("winget",
// "team.schema.json", "/save-learnings") legitimately appear in v1→v2
// historical-contrast prose ("winget was retired", "v1's /save-learnings, now
// /drain"), so a substring match on them is ~100% false-positive on the real site —
// the round-4 lesson, reconfirmed here at the mechanism level. Concept-rename drift
// in prose is the LLM semantic check's job, not this deterministic net's; an
// *instruction* like "brew install atl" never appears in historical contrast, so it
// is the zero-false-positive slice that stays deterministic.
var DefaultDenylist = []string{
	"brew install",
	"brew upgrade",
	"scoop install",
	"winget install",
	"winget upgrade",
}

// cliPageAllowlist are pages under cli/ that intentionally map to no command.
var cliPageAllowlist = map[string]bool{"overview": true, "index": true}

// coverageCmdAllowlist are commands intentionally given no cli/<name>.md page of
// their own: documented alongside a sibling, or internal hook commands users never
// type. Kept tiny + commented so it can't quietly hide real gaps.
var coverageCmdAllowlist = map[string]bool{
	"unpin":         true, // documented with pin in cli/pin.md
	"session-start": true, // internal SessionStart-hook command, not a user verb
}

// skillsPageAllowlist are pages under skills/ that intentionally map to no skill.
var skillsPageAllowlist = map[string]bool{"overview": true, "index": true}

// RunAll runs every LLM-free check (everything but the networked External check).
func RunAll(in Input) []Finding {
	var f []Finding
	f = append(f, Coverage(in.SiteDir, in.Commands)...)
	if in.CoreDir != "" {
		f = append(f, SkillCoverage(in.SiteDir, in.CoreDir)...)
	}
	f = append(f, Parity(in.SiteDir)...)
	f = append(f, Tokens(in.SiteDir, in.Denylist)...)
	f = append(f, Links(in.SiteDir)...)
	f = append(f, Flags(in.SiteDir, in.Commands)...)
	return f
}

// Coverage checks that every CLI command has a cli/<name>.md page and that every
// cli/*.md page maps to a shipping command (both directions). Fail-level.
func Coverage(siteDir string, commands []Command) []Finding {
	var f []Finding
	have := pageStems(filepath.Join(siteDir, "cli"))
	cmdSet := map[string]bool{}
	for _, c := range commands {
		cmdSet[c.Name] = true
		if !have[c.Name] && !coverageCmdAllowlist[c.Name] {
			f = append(f, Finding{"coverage", Fail, filepath.ToSlash(filepath.Join("cli", c.Name+".md")),
				"command `atl " + c.Name + "` has no docs page"})
		}
	}
	for _, name := range orphans(have, cmdSet, cliPageAllowlist) {
		f = append(f, Finding{"coverage", Fail, filepath.ToSlash(filepath.Join("cli", name+".md")),
			"docs page describes no shipping command `atl " + name + "`"})
	}
	return f
}

// SkillCoverage checks that every core skill has a skills/<name>.md page and vice
// versa. Skipped silently when coreDir has no skills/ (e.g. a non-monorepo repo).
func SkillCoverage(siteDir, coreDir string) []Finding {
	var f []Finding
	srcEntries, err := os.ReadDir(filepath.Join(coreDir, "skills"))
	if err != nil {
		return nil
	}
	have := pageStems(filepath.Join(siteDir, "skills"))
	srcSet := map[string]bool{}
	for _, e := range srcEntries {
		if !e.IsDir() {
			continue
		}
		srcSet[e.Name()] = true
		if !have[e.Name()] {
			f = append(f, Finding{"coverage", Fail, filepath.ToSlash(filepath.Join("skills", e.Name()+".md")),
				"skill `/" + e.Name() + "` has no docs page"})
		}
	}
	for _, name := range orphans(have, srcSet, skillsPageAllowlist) {
		f = append(f, Finding{"coverage", Fail, filepath.ToSlash(filepath.Join("skills", name+".md")),
			"docs page describes no shipping skill `/" + name + "`"})
	}
	return f
}

// Parity checks that every EN page has a TR mirror under tr/. Fail-level.
func Parity(siteDir string) []Finding {
	var f []Finding
	trDir := filepath.Join(siteDir, "tr")
	_ = filepath.WalkDir(siteDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".vitepress" || path == trDir {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(siteDir, path)
		if _, err := os.Stat(filepath.Join(trDir, rel)); err != nil {
			f = append(f, Finding{"parity", Fail, filepath.ToSlash(rel),
				"EN page has no TR mirror at tr/" + filepath.ToSlash(rel)})
		}
		return nil
	})
	return f
}

// Tokens greps every page for forbidden stale tokens. Fail-level.
func Tokens(siteDir string, denylist []string) []Finding {
	var f []Finding
	walkMarkdown(siteDir, func(rel string, content []byte) {
		for i, line := range strings.Split(string(content), "\n") {
			for _, tok := range denylist {
				if strings.Contains(line, tok) {
					f = append(f, Finding{"tokens", Fail, rel,
						fmt.Sprintf("line %d: stale instruction %q (retired channel — v2 ships install.sh + GitHub Releases)", i+1, tok)})
				}
			}
		}
	})
	return f
}

var mdLink = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)

// Links checks that internal relative links resolve to an existing file. Warn-level:
// VitePress's own build is the authority on dead links; this is the fast Go preview
// (so session-start can run without Node) and is conservative to avoid false fails.
func Links(siteDir string) []Finding {
	var f []Finding
	walkMarkdown(siteDir, func(rel string, content []byte) {
		fromDir := filepath.Dir(filepath.Join(siteDir, filepath.FromSlash(rel)))
		for _, m := range mdLink.FindAllSubmatch(content, -1) {
			target := strings.TrimSpace(string(m[1]))
			if i := strings.IndexAny(target, " \t#"); i >= 0 {
				target = target[:i]
			}
			if target == "" || isExternal(target) || strings.HasPrefix(target, "mailto:") {
				continue
			}
			if strings.ContainsAny(target, "{}<>") || strings.Contains(target, "...") {
				continue // template placeholder / illustrative path, not a real link
			}
			if !resolveLink(siteDir, fromDir, target) {
				f = append(f, Finding{"links", Warn, rel, "internal link may be broken: " + target})
			}
		}
	})
	return f
}

// Flags checks that every long flag of a command appears in its doc page. Warn-level
// (a flag can be described in prose without the literal `--name`).
func Flags(siteDir string, commands []Command) []Finding {
	var f []Finding
	for _, c := range commands {
		if len(c.Flags) == 0 {
			continue
		}
		b, err := os.ReadFile(filepath.Join(siteDir, "cli", c.Name+".md"))
		if err != nil {
			continue // a missing page is already a coverage failure
		}
		text := string(b)
		for _, fl := range c.Flags {
			if !strings.Contains(text, "--"+fl) {
				f = append(f, Finding{"flags", Warn, filepath.ToSlash(filepath.Join("cli", c.Name+".md")),
					"flag --" + fl + " not documented"})
			}
		}
	}
	return f
}

// External checks that external links return < 400 over HTTP. Warn-level, networked,
// opt-in (slow + sensitive to transient outages). Not part of RunAll.
func External(siteDir string) []Finding {
	urls := map[string]string{} // url -> first page that references it
	walkMarkdown(siteDir, func(rel string, content []byte) {
		for _, m := range mdLink.FindAllSubmatch(content, -1) {
			t := strings.TrimSpace(string(m[1]))
			if i := strings.IndexAny(t, " \t"); i >= 0 {
				t = t[:i]
			}
			if isExternal(t) {
				if _, ok := urls[t]; !ok {
					urls[t] = rel
				}
			}
		}
	})
	keys := make([]string, 0, len(urls))
	for u := range urls {
		keys = append(keys, u)
	}
	sort.Strings(keys)

	client := &http.Client{Timeout: 10 * time.Second}
	var f []Finding
	for _, u := range keys {
		if !reachable(client, u) {
			f = append(f, Finding{"external", Warn, urls[u], "external link unreachable: " + u})
		}
	}
	return f
}

// --- helpers ---

func isExternal(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// pageStems returns the set of *.md basenames (without extension) in a directory.
func pageStems(dir string) map[string]bool {
	out := map[string]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out[strings.TrimSuffix(e.Name(), ".md")] = true
		}
	}
	return out
}

// orphans returns sorted page stems present on disk but absent from want and the allowlist.
func orphans(have, want, allow map[string]bool) []string {
	var out []string
	for name := range have {
		if want[name] || allow[name] {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func walkMarkdown(siteDir string, fn func(rel string, content []byte)) {
	_ = filepath.WalkDir(siteDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".vitepress" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(siteDir, path)
		fn(filepath.ToSlash(rel), b)
		return nil
	})
}

// resolveLink mimics VitePress link resolution closely enough to confirm a target
// exists: absolute "/x" is rooted at the site dir, relative paths at the file's dir;
// a missing extension is tried as ".md", a "/" as "/index.md", ".html" as ".md".
func resolveLink(siteDir, fromDir, target string) bool {
	var base string
	if strings.HasPrefix(target, "/") {
		base = filepath.Join(siteDir, filepath.FromSlash(strings.TrimPrefix(target, "/")))
	} else {
		base = filepath.Join(fromDir, filepath.FromSlash(target))
	}
	candidates := []string{base, base + ".md", filepath.Join(base, "index.md")}
	if strings.HasSuffix(base, ".html") {
		candidates = append(candidates, strings.TrimSuffix(base, ".html")+".md")
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

func reachable(client *http.Client, url string) bool {
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return false
		}
		req.Header.Set("User-Agent", "atl-docs-check")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 400 {
			return true
		}
	}
	return false
}
