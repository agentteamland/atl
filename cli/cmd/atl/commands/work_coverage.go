package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/agentteamland/atl/cli/internal/coverage"
	"github.com/agentteamland/atl/cli/internal/dispatch"
	"github.com/spf13/cobra"
)

// reportCandidates are the paths the common runners write, in the order they are
// probed. Discovery exists so the caller does not have to know which runner the
// project uses — the pack knows the command, this knows where its output lands.
//
// One entry per family rather than per stack: lcov is emitted by c8/vitest AND by
// `flutter test --coverage`, so the web and mobile stacks share a line.
var reportCandidates = []string{
	"coverage/lcov.info",              // c8 / vitest / nyc, and flutter --coverage
	"lcov.info",                       //
	"coverage/cobertura-coverage.xml", // some JS reporters
	"cover.out",                       // go test -coverprofile
	"coverage.out",                    //
}

// coberturaGlobs find coverlet's output, which lands under a per-run GUID
// directory that cannot be named ahead of time.
var coberturaGlobs = []string{
	"TestResults/*/coverage.cobertura.xml",
	"**/TestResults/*/coverage.cobertura.xml",
	"coverage/*.cobertura.xml",
}

// findReport probes the known locations and returns the first hit. It reports
// which path it used, because a measurement whose input you cannot name is not a
// measurement — a stale report from a previous run scores just as green as a
// fresh one, and silently.
func findReport(root string) (string, error) {
	for _, rel := range reportCandidates {
		p := filepath.Join(root, rel)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	for _, g := range coberturaGlobs {
		if hits, _ := filepath.Glob(filepath.Join(root, g)); len(hits) > 0 {
			return hits[0], nil
		}
	}
	return "", fmt.Errorf("no coverage report found under %s — run the pack's coverage command first "+
		"(see its production-unit.md step 4b), or pass --report", root)
}

// measureCoverage is the whole command minus cobra: resolve the base and the
// report, measure, and report. Extracted so the decision logic is reachable by a
// test — a RunE closure that reads the process's cwd and shells out is not, and
// "the CLI wrapper is untestable" is how the least-tested code ends up being the
// code that decides whether a gate passes.
//
// warn receives the dirty-tree note; the caller routes it to stderr.
func measureCoverage(root, base, report string, min float64, run coverage.Runner, warn func(string)) (coverage.Result, error) {
	var zero coverage.Result

	if base == "" {
		// Default to the project's own integration branch, never a hardcoded
		// "dev" — a project that renamed its pair must not be measured against a
		// ref that does not exist.
		cfg, err := dispatch.LoadDeliveryConfig(root)
		if err != nil {
			return zero, err
		}
		if cfg == nil {
			return zero, fmt.Errorf("no %s and no --base — pass --base, or run this in a delivery project",
				dispatch.DeliveryConfigPath(root))
		}
		base = "origin/" + cfg.DevBranch()
	}

	if report == "" {
		var err error
		if report, err = findReport(root); err != nil {
			return zero, err
		}
	}

	// `base...HEAD` sees COMMITTED work only. Run mid-edit, an uncommitted change
	// is invisible and the measurement comes back a free 100% — a pass that looks
	// exactly like a real one. Say so rather than letting the number stand alone.
	if out, err := run("git", "status", "--porcelain"); err == nil && len(out) > 0 {
		warn("note: the working tree is dirty — this measures COMMITTED work only, " +
			"so uncommitted changes are not in the denominator. Commit first for the real number.")
	}

	diff, err := coverage.DiffLines(run, base)
	if err != nil {
		return zero, err
	}
	// A profile older than the source it describes carries line numbers from a
	// different file, so every line in it is attributed to the wrong statement.
	// This is not hypothetical: `go test -coverprofile` writes a profile even when
	// the test result is CACHED, and that profile is the cached run's — measured
	// here, a stale profile put three comment lines inside an "uncovered block"
	// and reported a 100% change as 63.6%.
	//
	// It fails in both directions, and the other one is worse: a stale profile can
	// mark newly-added lines as covered, which is a false green on the gate that
	// decides whether a unit is done.
	if stale, src := staleProfile(report, sortedFiles(diffFilesOf(diff))); stale {
		warn("warning: " + report + " is older than " + src + " — it may describe different " +
			"source. `go test` writes a coverage profile even when the result is cached; " +
			"re-run with -count=1.")
	}

	measurable, covered, err := coverage.Covered(report)
	if err != nil {
		return zero, err
	}
	res := coverage.Measure(diff, measurable, covered)
	res.Base, res.Report, res.Min = base, report, min
	return res, nil
}

var workCoverageCmd = &cobra.Command{
	Use:   "coverage",
	Short: "Measure DIFF coverage — of the lines this change wrote, how many does the suite execute?",
	Long: "Intersect the lines `<base>...HEAD` added or modified with the lines a\n" +
		"coverage report says were executed, and fail below --min.\n\n" +
		"Diff coverage rather than a project percentage: a whole-project number is\n" +
		"unusable as a gate on an existing codebase and insensitive on a large one.\n" +
		"The lines a change wrote are the lines it is answerable for.\n\n" +
		"This closes the hole where a number is CLAIMED rather than measured. It does\n" +
		"NOT prove the tests assert anything — coverage says a line ran, never that\n" +
		"something checked it. The other half of the gate (a test that goes red when\n" +
		"the change is reverted) stays a human judgement.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			return err
		}
		res, err := measureCoverage(root, workCoverageBase, workCoverageReport, workCoverageMin,
			coverage.Runner(dispatch.ExecRunner),
			func(m string) { fmt.Fprintln(os.Stderr, m) })
		if err != nil {
			return err
		}
		fmt.Print(renderCoverage(res, workCoverageJSON))
		if res.Percent < res.Min {
			// A non-zero exit is the whole point: a caller that only reads stdout
			// can miss a number, but it cannot miss a failure.
			return fmt.Errorf("diff coverage %.1f%% is below the %.0f%% minimum — "+
				"write a test for the uncovered lines above, or record the exception on the work item",
				res.Percent, res.Min)
		}
		return nil
	},
}

// renderCoverage formats the measurement. Separated from the command so its
// output — the thing a reviewer reads and the JSON form attached as evidence —
// is asserted rather than eyeballed once.
func renderCoverage(res coverage.Result, asJSON bool) string {
	if asJSON {
		b, err := json.MarshalIndent(struct {
			coverage.Result
			Passed bool `json:"passed"`
		}{res, res.Percent >= res.Min}, "", "  ")
		if err != nil {
			return fmt.Sprintf("render: %v\n", err)
		}
		return string(b) + "\n"
	}
	out := fmt.Sprintf("diff coverage: %.1f%% (%d/%d lines)  base=%s  report=%s\n",
		res.Percent, res.Covered, res.Total, res.Base, res.Report)
	for _, f := range res.Files {
		if len(f.Uncovered) > 0 {
			out += fmt.Sprintf("  %s — %d/%d, uncovered: %v\n", f.Path, f.Covered, f.Total, f.Uncovered)
		}
	}
	if res.Total == 0 {
		out += "  (this change touched no measurable source lines)\n"
	}
	return out
}

var (
	workCoverageBase   string
	workCoverageReport string
	workCoverageMin    float64
	workCoverageJSON   bool
)

func init() {
	workCoverageCmd.Flags().StringVar(&workCoverageBase, "base", "", "compare against this ref (default: origin/<config branchPair.dev>)")
	workCoverageCmd.Flags().StringVar(&workCoverageReport, "report", "", "coverage report path (default: auto-discover)")
	workCoverageCmd.Flags().Float64Var(&workCoverageMin, "min", 90, "minimum diff coverage percent")
	workCoverageCmd.Flags().BoolVar(&workCoverageJSON, "json", false, "emit the machine-readable measurement (the form to attach as evidence)")
	workCmd.AddCommand(workCoverageCmd)
}

// diffFilesOf lists the files a diff touched.
func diffFilesOf(diff coverage.Lines) []string {
	out := make([]string, 0, len(diff))
	for f := range diff {
		out = append(out, f)
	}
	return out
}

// sortedFiles returns paths in a stable order, so the staleness warning names the
// same file run to run rather than whichever the map iteration happened to yield.
func sortedFiles(files []string) []string {
	sort.Strings(files)
	return files
}

// staleProfile reports whether the coverage report predates any of the source
// files it is being measured against, returning the first offender.
//
// Deliberately conservative — a missing file or an unreadable stat is not
// staleness, because a false warning on every run is how a real one stops being
// read. Compares against the files in the DIFF rather than every file in the
// profile: those are the ones whose line numbers this measurement depends on.
func staleProfile(report string, sources []string) (bool, string) {
	ri, err := os.Stat(report)
	if err != nil {
		return false, ""
	}
	for _, f := range sources {
		si, err := os.Stat(f)
		if err != nil {
			continue // deleted, or outside the working tree
		}
		if si.ModTime().After(ri.ModTime()) {
			return true, f
		}
	}
	return false, ""
}
