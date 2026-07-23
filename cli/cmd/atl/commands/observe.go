package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/agentteamland/atl/cli/internal/sweepstate"
	"github.com/spf13/cobra"
)

var observeCmd = &cobra.Command{
	Use:   "observe",
	Short: "Proactive-observer status + cursor (the /observe backstop)",
	Long: "The deterministic half of the proactive observer: report whether a sweep is\n" +
		"due and (with --record) stamp the cursor the /observe skill sets after a sweep.\n" +
		"The audit itself — backlog-trigger ripeness + the latent-gap sweep — is LLM work\n" +
		"the /observe skill runs (the CLI/Skill boundary: CLI = deterministic, Skill =\n" +
		"LLM). Outside a project with an .atl/ decision surface it does nothing, exit 0.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		record, _ := cmd.Flags().GetBool("record")

		root, err := observeRoot()
		if err != nil {
			fmt.Println("atl observe: no .atl/ surface here — nothing to observe")
			return nil
		}
		cursor := sweepstate.Observe.ForProject(root)
		if record {
			sha := gitHEAD(root)
			if sha == "" {
				fmt.Println("atl observe: not a git repo here — nothing to record")
				return nil
			}
			_ = cursor.Record(sha, time.Now())
			fmt.Println("atl observe: recorded HEAD as the last observer sweep")
			return nil
		}
		if cursor.Due(root) {
			fmt.Println("atl observe: a proactive observer sweep is due — run /observe")
		} else {
			fmt.Println("atl observe: no sweep due right now")
		}
		return nil
	},
}

// observeRoot walks up from the working directory to the nearest project that holds an
// ATL decision surface (a .atl/ directory) — the root the observer records against.
func observeRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := wd; ; {
		if fi, err := os.Stat(filepath.Join(dir, ".atl")); err == nil && fi.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .atl/ surface found")
		}
		dir = parent
	}
}

// observeSessionSignal surfaces a "sweep due" signal at session start, but only in a
// project that has an ATL decision surface (.atl/) — silent everywhere else, so a
// session in a project that never uses ATL's knowledge layer pays nothing. It watches
// the .atl/ surface for cadence: a commit there since the last sweep (gated by the
// ~1-day runaway-guard) means fresh material — a decision, a ship, a journal entry —
// that a proactive audit should look over. The signal itself is cheap (a git log); the
// expensive LLM sweep runs only if /observe is invoked in response. Best-effort; a hook
// must never block, so any error yields silence.
func observeSessionSignal(projectRoot string) {
	if projectRoot == "" {
		return
	}
	if fi, err := os.Stat(filepath.Join(projectRoot, ".atl")); err != nil || !fi.IsDir() {
		return // no ATL decision surface here — the observer is dormant
	}
	if sweepstate.Observe.ForProject(projectRoot).Due(projectRoot) {
		fmt.Println("atl: a proactive observer sweep is due — run /observe to surface ripe backlog triggers and latent gaps (shipped-vs-designed, growth/scale risks, unshipped decisions, your setup)")
	}
}

func init() {
	observeCmd.Flags().Bool("record", false, "record HEAD as the last observer sweep (after an /observe run)")
}
