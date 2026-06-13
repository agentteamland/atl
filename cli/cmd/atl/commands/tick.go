package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/agentteamland/atl/cli/internal/drain"
	"github.com/spf13/cobra"
)

var tickCmd = &cobra.Command{
	Use:   "tick",
	Short: "Run the in-session maintenance tick",
	Long: "Run the in-session maintenance tick — the work the three-speed cadence\n" +
		"fires every few minutes via the prompt-piggyback throttle. The v2 target is\n" +
		"drain + doctor + fan-out; this step wires the drain: it parses capture\n" +
		"markers from conversation text and transfers them into the durable queue\n" +
		"(exactly once — re-running is a safe no-op).\n\n" +
		"Reads text from --file, or stdin if --file is omitted. Real transcript\n" +
		"discovery + a per-source cursor land in a later step.",
	RunE: func(cmd *cobra.Command, args []string) error {
		text, err := readTickInput(cmd)
		if err != nil {
			return err
		}
		st, project, err := openQueue()
		if err != nil {
			return err
		}
		defer st.Close()

		r, err := drain.Drain(text, project, st)
		if err != nil {
			return err
		}
		fmt.Printf("tick: drained %d marker(s) — %d new, %d already queued\n",
			r.Found, r.Enqueued, r.Found-r.Enqueued)
		return nil
	},
}

func readTickInput(cmd *cobra.Command) (string, error) {
	if file, _ := cmd.Flags().GetString("file"); file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("read --file: %w", err)
		}
		return string(b), nil
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return string(b), nil
}

func init() {
	tickCmd.Flags().String("file", "", "read conversation text from this file (default: stdin)")
}
