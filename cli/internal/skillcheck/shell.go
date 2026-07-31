package skillcheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ShellBodies flags shell constructs inside a SKILL.md's fenced sh/bash blocks
// that behave differently — or fatally — in the shell that actually runs them.
//
// A skill body is executable: the agent pastes the block into whatever shell its
// Bash tool runs, so the ```bash fence is a LABEL, not the executor (it is zsh on
// macOS). zsh's default `nomatch` makes an unmatched glob a fatal error where
// bash leaves it literal, and one such abort landed after a `rm -rf`, wiping a
// backup and printing none of the skill's outcome markers (atl#331, atl#334).
//
// This is deliberately a PATTERN match over known-unportable constructs, NOT a
// parse. `sh -n`, `bash -n` and `zsh -n` all accept every construct flagged here
// (verified) — `nomatch` is runtime behavior, so a syntax check would catch none
// of it. Do not "improve" this into a parse; there is no value there.
//
// The bound, so no reader over-reads it: this is NOT a general "is this body
// shell-agnostic" check. It knows exactly two constructs plus one escalation.
// The nearest sibling it does NOT know is a bare glob outside a `for` list
// (`cp -R "$SRC"/* "$DEST/"`), which aborts under zsh identically. That is not
// an oversight: widening the glob rule to every command word was measured over
// the corpus and every single hit was a false positive (a trailing comment
// ending in `?`, a `case` pattern, JavaScript quoted in a shell fence), and a
// noisy gate gets disabled. Narrow and honest beats broad and ignored.
func ShellBodies(coreDir, teamsDir string) []Finding {
	var f []Finding
	if coreDir != "" {
		f = append(f, shellInSkillDir(filepath.Join(coreDir, "skills"), "core/skills")...)
	}
	for _, team := range teamNames(teamsDir) {
		base := filepath.Join(teamsDir, team, "skills")
		f = append(f, shellInSkillDir(base, "teams/"+team+"/skills")...)
	}
	return f
}

func shellInSkillDir(dir, rel string) []Finding {
	var f []Finding
	for _, name := range subdirs(dir) {
		md, mdRel := skillFile(filepath.Join(dir, name), filepath.Join(rel, name))
		b, err := os.ReadFile(md)
		if err != nil {
			continue // a missing SKILL.md is the frontmatter check's finding, not ours
		}
		for _, blk := range shellBlocks(string(b)) {
			f = append(f, scanShellBlock(blk, mdRel)...)
		}
	}
	return f
}

var (
	shellFenceOpen = regexp.MustCompile("^```(bash|sh|zsh|shell)\\s*$")
	// A fence carrying an info string is definitely an OPENER. A bare ``` is
	// ambiguous — as likely a stray closer — so it is never treated as one.
	fenceTagged = regexp.MustCompile("^`{3,}\\s*[A-Za-z]")
	forInList   = regexp.MustCompile(`^for\s+[A-Za-z_][A-Za-z0-9_]*\s+in\s+(.+)$`)
	testCommand = regexp.MustCompile(`^(\[\[?|test)\s`)
	rmRecursive = regexp.MustCompile(`\brm\s+(-[A-Za-z]+\s+)*-[A-Za-z]*[rR]`)
	funcOpen    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\s*\(\)\s*\{?\s*$`)
	// The leading (^|[^<]) keeps a here-STRING (`cmd <<<"word"`) from reading as
	// a heredoc opener: that would put the scanner into skip-until-`word` mode
	// and silently drop the rest of the block — the check going quiet with no
	// sign, which is the one failure a gate must not have.
	heredocOpen = regexp.MustCompile(`(^|[^<])<<-?\s*['"]?([A-Za-z_][A-Za-z0-9_]*)['"]?`)
	// Quoted spans and command substitutions: their contents are not glob
	// patterns the caller's shell expands, so they are stripped before matching.
	shellLiteral = regexp.MustCompile("'[^']*'|\"[^\"]*\"|\\$\\([^)]*\\)|`[^`]*`")
)

// codeLine is one executable line of a fenced block: comments, blank lines and
// heredoc bodies are dropped, so only text the shell would run reaches a check.
type codeLine struct {
	n    int    // 1-based line number in the SKILL.md
	text string // the line, trimmed
}

// shellBlocks returns the executable lines of each fenced sh/bash/zsh/shell block.
func shellBlocks(src string) [][]codeLine {
	var out [][]codeLine
	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		run := fenceRun(t)
		if run == 0 {
			continue
		}
		if !shellFenceOpen.MatchString(t) {
			// A tagged non-shell fence is prose, so skip its whole body: a shell
			// block QUOTED inside a wider fence is a doc SHOWING the anti-pattern,
			// not a body anyone runs, and flagging it is the false positive that
			// gets a gate disabled. Two guards keep the skip from blinding the
			// scanner instead: an untagged ``` is never assumed to be an opener,
			// and a fence with no closer at all is not skipped over.
			if !fenceTagged.MatchString(t) {
				continue
			}
			if end := closingFence(lines, i+1, run); end >= 0 {
				i = end
			}
			continue
		}
		body := i + 1
		end := closingFence(lines, body, run)
		if end < 0 {
			end = len(lines)
		}
		if code := codeLines(lines[body:end], body+1); len(code) > 0 {
			out = append(out, code)
		}
		i = end
	}
	return out
}

// fenceRun is the length of a line's leading backtick run, or 0 when the line is
// not a fence delimiter.
func fenceRun(line string) int {
	n := 0
	for n < len(line) && line[n] == '`' {
		n++
	}
	if n < 3 {
		return 0
	}
	return n
}

// closingFence returns the index of the first line at or after from that closes
// a fence opened with run backticks, or -1 when the fence is never closed.
func closingFence(lines []string, from, run int) int {
	for i := from; i < len(lines); i++ {
		if fenceRun(strings.TrimSpace(lines[i])) >= run {
			return i
		}
	}
	return -1
}

func codeLines(body []string, firstLine int) []codeLine {
	var out []codeLine
	heredoc := ""
	for i, raw := range body {
		t := strings.TrimSpace(raw)
		if heredoc != "" {
			if t == heredoc {
				heredoc = ""
			}
			continue // a heredoc body is data the shell never executes
		}
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if m := heredocOpen.FindStringSubmatch(t); m != nil {
			heredoc = m[2] // the opening line itself is still code
		}
		out = append(out, codeLine{firstLine + i, t})
	}
	return out
}

func scanShellBlock(code []codeLine, relPath string) []Finding {
	var f []Finding
	errexit := hasErrexit(code)
	tails := tailStatements(code)
	for i, l := range code {
		var detail string
		switch glob := globInForList(l.text); {
		case glob != "":
			detail = "`for` iterates a bare glob (`" + glob + "`) — zsh's default `nomatch` makes an unmatched glob FATAL where bash leaves it literal; iterate a `find` result or copy with `cp -R \"$SRC/.\"` instead"
		case errexit && tails[i] && testAndNoFallback(l.text):
			detail = "`" + firstWords(l.text) + "` is the last statement under `set -e`, so a false test makes the whole unit exit non-zero (verified in sh, bash and zsh) — use `if … then … fi`, or add `|| true`"
		}
		if detail == "" {
			continue
		}
		if rm := destructiveBefore(code, i); rm > 0 {
			// The ordering that turned a portability nit into data loss: the
			// deletion has already happened when the fragile line aborts, and
			// with no trap the script prints no outcome at all.
			detail += "; it also follows `rm -r` at line " + strconv.Itoa(rm) + " with no `trap`, so an abort here leaves the deletion done and reports nothing"
		}
		f = append(f, Finding{"shell", Fail, relPath, "line " + strconv.Itoa(l.n) + ": " + detail})
	}
	return f
}

// hasErrexit reports whether the block turns errexit on, in any of its spellings
// (`set -e`, `set -eu`, `set -euo pipefail`, `set -o errexit`).
func hasErrexit(code []codeLine) bool {
	for _, l := range code {
		fields := strings.Fields(l.text)
		if len(fields) == 0 || fields[0] != "set" {
			continue
		}
		for i := 1; i < len(fields); i++ {
			if fields[i] == "-o" && i+1 < len(fields) && fields[i+1] == "errexit" {
				return true
			}
			if strings.HasPrefix(fields[i], "-") && !strings.HasPrefix(fields[i], "--") && strings.Contains(fields[i], "e") {
				return true
			}
		}
	}
	return false
}

// tailStatements marks the statements where a non-zero status actually escapes:
// the last statement of the script, and the last statement of a function body.
// Those are the only two positions where `[ … ] && cmd` bites — verified under
// sh, bash and zsh: mid-script, and as the last statement of a loop body, an if
// branch or a brace group, POSIX exempts every AND-OR member but the last from
// errexit, so the false test is harmless there and must NOT be flagged.
func tailStatements(code []codeLine) []bool {
	tails := make([]bool, len(code))
	if len(code) > 0 {
		tails[len(code)-1] = true
	}
	inFunc := false
	for i, l := range code {
		switch {
		case funcOpen.MatchString(l.text):
			inFunc = true
		case inFunc && l.text == "}":
			if i > 0 {
				tails[i-1] = true
			}
			inFunc = false
		}
	}
	return tails
}

// globInForList returns the first word of a `for … in <list>` carrying an
// unquoted glob, or "" when the line is not such a loop or the list has none.
func globInForList(line string) string {
	m := forInList.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	list := m[1]
	if i := strings.Index(list, ";"); i >= 0 {
		list = list[:i]
	}
	for _, w := range strings.Fields(list) {
		if w == "do" {
			break
		}
		if strings.ContainsAny(shellLiteral.ReplaceAllString(w, ""), "*?") {
			return w
		}
	}
	return ""
}

// testAndNoFallback reports whether the statement is the `[ … ] && cmd` idiom
// with no `|| …` fallback to absorb a false test.
func testAndNoFallback(line string) bool {
	if !testCommand.MatchString(line) {
		return false
	}
	i := strings.Index(line, "&&")
	return i >= 0 && !strings.Contains(line[i:], "||")
}

// destructiveBefore returns the line number of the last recursive `rm` preceding
// statement i, or 0 when there is none — or when the block installs a `trap`,
// which is the author handling the abort themselves.
func destructiveBefore(code []codeLine, i int) int {
	found := 0
	for _, l := range code[:i] {
		if strings.HasPrefix(l.text, "trap ") {
			return 0
		}
		if rmRecursive.MatchString(shellLiteral.ReplaceAllString(l.text, "")) {
			found = l.n
		}
	}
	return found
}

// firstWords quotes a statement compactly for a finding message.
func firstWords(line string) string {
	if len(line) <= 48 {
		return line
	}
	return line[:45] + "…"
}
