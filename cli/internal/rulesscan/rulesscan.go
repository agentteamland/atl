// Package rulesscan is the deterministic collect half of rules-distill: it
// gathers the normative/imperative lines across the skill + agent corpus so the
// LLM /rules-distill skill can judge which recurring principles deserve to become
// a core rule. The collect only gathers (over-collecting is fine — the judgment
// is the skill's); rules/ directories are skipped because rules are the distill
// *target*, not a source.
package rulesscan

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Statement is one normative/imperative line found in the corpus — a candidate
// principle for /rules-distill to cluster and judge.
type Statement struct {
	File string // path relative to the repo root
	Line int
	Text string
}

// triggerRe matches strong imperative/normative signal phrases. A line carrying
// one (word-bounded, case-insensitive) is collected. It over-collects on purpose
// — the collect step only gathers candidates; the LLM decides which are real
// recurring principles. Kept to the *strong* signals (always/never/must/don't/
// avoid + the grep-before-edit idiom) so the candidate list stays a signal, not a
// dump; weaker words (prefer/ensure/require) were dropped as ~pure noise.
var triggerRe = regexp.MustCompile(`(?i)(\balways\b|\bnever\b|\bmust\b|\bdon't\b|\bdo not\b|\bavoid\b|grep .*before|before .*edit|first grep)`)

// Collect walks the skill + agent markdown under coreDir and teamsDir and returns
// every line carrying a normative/imperative trigger, with its source location.
// rules/ subtrees are skipped. Sorted by file then line.
func Collect(coreDir, teamsDir string) ([]Statement, error) {
	var out []Statement
	roots := []struct{ dir, base string }{}
	if coreDir != "" {
		roots = append(roots, struct{ dir, base string }{coreDir, "core"})
	}
	if teamsDir != "" {
		roots = append(roots, struct{ dir, base string }{teamsDir, "teams"})
	}
	for _, r := range roots {
		err := filepath.WalkDir(r.dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if d.IsDir() {
				if d.Name() == "rules" {
					return filepath.SkipDir // rules are the distill target, not a source
				}
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				return nil
			}
			rel, rerr := filepath.Rel(r.dir, path)
			if rerr != nil {
				return rerr
			}
			return collectFile(path, filepath.ToSlash(filepath.Join(r.base, rel)), &out)
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

func collectFile(path, relPath string, out *[]Statement) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	ln := 0
	for sc.Scan() {
		ln++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // skip blanks and headings
		}
		if triggerRe.MatchString(line) {
			*out = append(*out, Statement{File: relPath, Line: ln, Text: line})
		}
	}
	return sc.Err()
}
