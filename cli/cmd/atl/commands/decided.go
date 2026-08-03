package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// decidedRoots are the surfaces that can hold a decision, widest-first. Includes
// cli/ because a decision is sometimes only recorded in the code that implements
// it — one measured failure was refuted by a single line of a command's help text
// ("it runs automatically in the background tick") and by nothing else.
//
// Deliberately wider than the retrieval corpus. Retrieval must be selective; this
// is a direct search whose whole job is to be exhaustive, and where a false
// negative is the expensive outcome.
var decidedRoots = []string{
	".atl/docs", ".atl/brain-storms", ".atl/wiki", ".atl/journal",
	".claude", "docs", "cli", "core", "teams",
}

// decidedExts bounds the search to text a human wrote. A hit inside a vendored
// tree or a binary is noise in a tool whose empty result is meant to be read as
// an answer.
var decidedExts = map[string]bool{".md": true, ".go": true, ".json": true, ".sh": true, ".yml": true, ".yaml": true}

var decidedCmd = &cobra.Command{
	Use:   "decided <query>",
	Short: "Search every decision surface — and report honestly when there is nothing",
	Long: "Search the project's decision surfaces for a term, and say what was searched.\n\n" +
		"The intended use is the question that precedes writing a rationale: does the\n" +
		"record actually hold the decision I am about to assert? A zero result is the\n" +
		"finding, which is why the output names the roots and the commit it searched\n" +
		"rather than printing nothing.\n\n" +
		"It reports what the SEARCH did. \"0 matches\" is evidence about this query, not\n" +
		"proof that no decision exists — a term you did not think of is indistinguishable\n" +
		"from a decision that does not exist. Try the other words before concluding.",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			return err
		}
		q := strings.Join(args, " ")
		hits, searched := searchDecided(root, q)
		fmt.Fprint(cmd.OutOrStdout(), renderDecided(q, hits, searched))
		return nil
	},
}

// decidedHit is one matching line.
type decidedHit struct {
	Path string
	Line int
	Text string
}

// searchDecided walks the decision surfaces for a case-insensitive match,
// returning the hits and the roots that actually existed. A root that is absent
// is not searched and not claimed — the output must not imply coverage it did
// not have.
func searchDecided(root, query string) ([]decidedHit, []string) {
	needle := strings.ToLower(query)
	var hits []decidedHit
	var searched []string
	for _, rel := range decidedRoots {
		dir := filepath.Join(root, rel)
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			continue
		}
		searched = append(searched, rel)
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				n := d.Name()
				if p != dir && (n == "node_modules" || n == "vendor" || n == ".git" || n == "worktrees") {
					return filepath.SkipDir
				}
				return nil
			}
			if !decidedExts[strings.ToLower(filepath.Ext(p))] {
				return nil
			}
			f, ferr := os.Open(p)
			if ferr != nil {
				return nil
			}
			defer f.Close()
			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for n := 1; sc.Scan(); n++ {
				if strings.Contains(strings.ToLower(sc.Text()), needle) {
					rp, rerr := filepath.Rel(root, p)
					if rerr != nil {
						rp = p
					}
					hits = append(hits, decidedHit{rp, n, strings.TrimSpace(sc.Text())})
				}
			}
			return nil
		})
	}
	sort.Slice(hits, func(a, b int) bool {
		if hits[a].Path != hits[b].Path {
			return hits[a].Path < hits[b].Path
		}
		return hits[a].Line < hits[b].Line
	})
	return hits, searched
}

// renderDecided formats the result. The empty case gets the most care: it is the
// one this command exists for, and a bare "no matches" would be read as "no
// decision exists" — a stronger claim than a text search can make.
func renderDecided(query string, hits []decidedHit, searched []string) string {
	if len(searched) == 0 {
		return "atl decided: no decision surfaces here — run this in a project with .atl/ or .claude/\n"
	}
	var b strings.Builder
	if len(hits) == 0 {
		fmt.Fprintf(&b, "0 matches for %q\n", query)
		fmt.Fprintf(&b, "searched: %s\n\n", strings.Join(searched, " "))
		b.WriteString("That is a fact about this QUERY, not about the record. A decision written\n")
		b.WriteString("in other words is indistinguishable from one that does not exist, so try\n")
		b.WriteString("the synonyms before you draw a conclusion from this.\n")
		return b.String()
	}
	const maxShown = 40
	fmt.Fprintf(&b, "%d matches for %q\n", len(hits), query)
	fmt.Fprintf(&b, "searched: %s\n\n", strings.Join(searched, " "))
	for i, h := range hits {
		if i == maxShown {
			fmt.Fprintf(&b, "  … %d more\n", len(hits)-maxShown)
			break
		}
		t := h.Text
		if len(t) > 110 {
			t = t[:110] + "…"
		}
		fmt.Fprintf(&b, "  %s:%d\n    %s\n", h.Path, h.Line, t)
	}
	return b.String()
}

func init() { rootCmd.AddCommand(decidedCmd) }
