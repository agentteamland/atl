package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentteamland/atl/cli/internal/docscheck"
	"github.com/agentteamland/atl/cli/internal/docsstate"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Documentation-correctness checks for the docs site",
	Long: "Deterministic, LLM-free drift checks for the AgentTeamLand docs site.\n" +
		"The semantic (prose-vs-behavior) half lives in the /docs-audit skill — this\n" +
		"is the CLI/Skill boundary (CLI = deterministic, Skill = LLM).",
}

var docsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Run deterministic drift checks against the docs site",
	Long: "Check the VitePress docs site for mechanical drift: command + skill coverage,\n" +
		"EN<->TR parity, a stale-token denylist, internal links, and CLI-flag parity.\n" +
		"Exits non-zero on any failure-level finding (warnings never fail). Outside a\n" +
		"repo that has a docs site it does nothing and exits 0 (the pre-flight skip).",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		external, _ := cmd.Flags().GetBool("external")
		record, _ := cmd.Flags().GetBool("record-audit")

		siteDir, repoRoot, err := findSiteDir()
		if err != nil {
			fmt.Println("atl docs: no docs site here — nothing to check")
			return nil
		}

		in := docscheck.Input{
			SiteDir:  siteDir,
			CoreDir:  filepath.Join(repoRoot, "core"),
			Commands: commandDocs(),
			Denylist: docscheck.DefaultDenylist,
		}
		findings := docscheck.RunAll(in)
		if external {
			findings = append(findings, docscheck.External(siteDir)...)
		}

		fails, warns := splitFindings(findings)
		printFindings(findings)

		if record && len(fails) == 0 {
			if sha := gitHEAD(repoRoot); sha != "" {
				_ = docsstate.Record(sha, time.Now())
			}
		}

		switch {
		case len(fails) > 0:
			return fmt.Errorf("%d documentation drift item(s), %d warning(s) — fix before shipping", len(fails), len(warns))
		case len(warns) > 0:
			fmt.Printf("atl docs: no failures (%d warning(s))\n", len(warns))
		default:
			fmt.Println("atl docs: clean")
		}
		return nil
	},
}

func init() {
	docsCheckCmd.Flags().Bool("external", false, "also check external links over HTTP (slow, networked)")
	docsCheckCmd.Flags().Bool("record-audit", false, "record HEAD as the last-audited commit when free of failures")
	docsCmd.AddCommand(docsCheckCmd)
}

// findSiteDir walks up from the cwd to the repo that holds docs/site/.vitepress,
// returning the site dir and the repo root. The .vitepress probe is the
// "does this repo have a docs site" pre-flight.
func findSiteDir() (siteDir, repoRoot string, err error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	for {
		site := filepath.Join(dir, "docs", "site")
		if st, e := os.Stat(filepath.Join(site, ".vitepress")); e == nil && st.IsDir() {
			return site, dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", fmt.Errorf("no docs site found from %s upward", dir)
		}
		dir = parent
	}
}

// commandDocs flattens the live cobra tree into the documented surface the checks
// compare against — so coverage stays accurate with no hand-maintained inventory.
func commandDocs() []docscheck.Command {
	var out []docscheck.Command
	for _, c := range rootCmd.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		var flags []string
		c.Flags().VisitAll(func(fl *pflag.Flag) {
			if !fl.Hidden {
				flags = append(flags, fl.Name)
			}
		})
		out = append(out, docscheck.Command{Name: c.Name(), Flags: flags})
	}
	return out
}

func gitHEAD(repoRoot string) string {
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// docsSessionSignal surfaces docs-correctness notes at session start, but only when
// the current project has a docs site (monorepo-internal): a deterministic drift
// count (the fast Layer of docs-sync v2) and, when a full sweep is due, a
// /docs-audit signal. Silent in every project without a docs site (so end-user
// sessions pay nothing). Best-effort — it never errors; a hook must not block.
func docsSessionSignal() {
	site, repoRoot, err := findSiteDir()
	if err != nil {
		return // no docs site here — the pre-flight skip
	}
	in := docscheck.Input{
		SiteDir:  site,
		CoreDir:  filepath.Join(repoRoot, "core"),
		Commands: commandDocs(),
		Denylist: docscheck.DefaultDenylist,
	}
	if fails, _ := splitFindings(docscheck.RunAll(in)); len(fails) > 0 {
		fmt.Printf("atl docs: %d drift item(s) on the docs site — run `atl docs check`\n", len(fails))
	}
	if docsAuditDue(repoRoot) {
		fmt.Println("atl docs: a full audit is due — run /docs-audit to sweep the docs site for semantic drift")
	}
}

// docsAuditDue reports whether a /docs-audit full sweep is due: doc-affecting
// commits have landed since the last recorded audit, gated by a ~1-day
// runaway-guard. This is the backstop's pre-flight — the deterministic + change-time
// layers are the primary defense, so the sweep only needs to run sparsely.
func docsAuditDue(repoRoot string) bool {
	st, err := docsstate.Load()
	if err != nil {
		return false
	}
	if st.LastAuditAt != "" {
		if t, perr := time.Parse(time.RFC3339, st.LastAuditAt); perr == nil && time.Since(t) < 24*time.Hour {
			return false // runaway-guard: don't sweep again within ~1 day
		}
	}
	return docAffectingCommitsSince(repoRoot, st.LastAuditSHA)
}

// docAffectingCommitsSince reports whether any commit touching docs/, core/, or
// cli/ has landed since sinceSHA (or, when sinceSHA is empty, whether the repo has
// any such commit at all — i.e. it was never audited).
func docAffectingCommitsSince(repoRoot, sinceSHA string) bool {
	run := func(sha string) ([]byte, error) {
		args := []string{"-C", repoRoot, "log", "--oneline", "-1"}
		if sha != "" {
			args = append(args, sha+"..HEAD")
		}
		args = append(args, "--", "docs", "core", "cli")
		return exec.Command("git", args...).Output()
	}
	out, err := run(sinceSHA)
	if err != nil {
		// A recorded SHA that git no longer knows (rebase / squash / gc pruned it)
		// makes `<sha>..HEAD` fail — treat that as never-audited (retry with no range)
		// so the backstop doesn't go permanently silent, rather than reading the git
		// failure as "no doc-affecting commits". A failure with no range means it's not
		// a git repo (or has no history), where there is genuinely nothing to audit.
		if sinceSHA == "" {
			return false
		}
		if out, err = run(""); err != nil {
			return false
		}
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func splitFindings(fs []docscheck.Finding) (fails, warns []docscheck.Finding) {
	for _, f := range fs {
		if f.Severity == docscheck.Fail {
			fails = append(fails, f)
		} else {
			warns = append(warns, f)
		}
	}
	return fails, warns
}

func printFindings(fs []docscheck.Finding) {
	for _, f := range fs {
		marker := "FAIL"
		if f.Severity == docscheck.Warn {
			marker = "warn"
		}
		loc := f.Path
		if loc == "" {
			loc = "-"
		}
		fmt.Printf("  [%s] %s · %s — %s\n", marker, f.Check, loc, f.Detail)
	}
}
