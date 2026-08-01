package promotiongate

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Runner executes an external command and returns its combined output. It's the
// seam that keeps the gate testable: the real gate shells out via ExecRunner,
// while tests inject a fake to assert the exact gh argv and cover every hold path
// plus the promote path without touching the network or merging anything.
type Runner func(name string, args ...string) ([]byte, error)

// ExecRunner runs the real gh binary.
func ExecRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func (g Gate) gh(args ...string) (string, error) {
	b, err := g.Run("gh", args...)
	if err != nil {
		return "", fmt.Errorf("gh %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(b)))
	}
	return strings.TrimSpace(string(b)), nil
}

// findPromotionPR returns the number of the open dev→release PR, or 0 when there
// is none. The ceremony's open-or-find step never opens a second one
// (backends/github/adapter.md §10, /sprint-review 6a), so the first match is the
// promotion PR.
func (g Gate) findPromotionPR(repo string) (int, error) {
	out, err := g.gh("pr", "list", "--repo", repo, "--state", "open",
		"--base", g.Release, "--head", g.Dev, "--json", "number")
	if err != nil {
		return 0, err
	}
	var prs []struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return 0, fmt.Errorf("parse gh pr list: %w", err)
	}
	if len(prs) == 0 {
		return 0, nil
	}
	return prs[0].Number, nil
}

// viewPR reads the head commit and every comment in ONE call, as the adapter's
// concept-#16 read binding specifies. One call is not just economy: the head and
// the records it is compared against come from the same snapshot, so a comment
// posted between two separate reads can't make the comparison lie.
func (g Gate) viewPR(repo string, pr int) (string, []Comment, error) {
	out, err := g.gh("pr", "view", strconv.Itoa(pr), "--repo", repo, "--json", "headRefOid,comments")
	if err != nil {
		return "", nil, err
	}
	var v struct {
		HeadRefOid string    `json:"headRefOid"`
		Comments   []Comment `json:"comments"`
	}
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		return "", nil, fmt.Errorf("parse gh pr view: %w", err)
	}
	return v.HeadRefOid, v.Comments, nil
}

// merge promotes the PR, pinned to the approved commit.
//
// --merge is mandatory: --squash/--rebase rewrite the SHA and false-block the
// engine's MergedToBase verify (adapter §10). --match-head-commit is what makes
// GitHub itself refuse the merge when the head moved since the read above — the
// provider closes the verify→merge window rather than the caller closing it by
// acting fast, so this gate has two independent lines of defence.
func (g Gate) merge(repo string, pr int, sha string) error {
	_, err := g.gh("pr", "merge", strconv.Itoa(pr), "--repo", repo, "--merge", "--match-head-commit", sha)
	return err
}
