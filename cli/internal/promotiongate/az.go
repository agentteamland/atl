package promotiongate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The Azure DevOps binding of the gate (backends/azure/adapter.md §10).
//
// The three reads go through the official `az` CLI with its azure-devops
// extension, the way gh.go goes through `gh`. The completion does NOT: `az devops
// invoke --area git --resource pullRequests` always resolves to the project-scoped
// GET-by-id route, which refuses PATCH, and `az repos pr update --status completed`
// pins the head IT reads rather than the one this gate verified. So the completion
// is one REST PATCH atl issues itself, carrying the verified commit as
// lastMergeSourceCommit. Azure refuses that PATCH when the PR's source has moved
// past the commit — HTTP 409, TF401192, GitPullRequestStaleException, observed on a
// live server — which makes it the twin of gh's --match-head-commit: the provider,
// not the caller's speed, closes the verify→merge window.

// azAPIVersion is the REST version the reads and the completion are issued at.
const azAPIVersion = "7.1"

// HTTPDoer sends one authenticated request and returns the status code and body.
// It is the seam the completion PATCH goes through, so tests never touch Azure.
type HTTPDoer func(method, url string, body []byte, pat string) (int, []byte, error)

// HTTPRequest is the real HTTPDoer: Basic auth with an empty user and the PAT as
// the password, which is how Azure DevOps accepts a personal access token. The
// PAT is never part of the URL or of any error this returns.
func HTTPRequest(method, u string, body []byte, pat string) (int, []byte, error) {
	req, err := http.NewRequest(method, u, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth("", pat)
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return resp.StatusCode, b, err
}

// orgURL turns config.org into the organization URL: a bare name becomes
// https://dev.azure.com/<name>; a full URL is used as given.
func (g Gate) orgURL() string {
	if strings.HasPrefix(g.Org, "https://") || strings.HasPrefix(g.Org, "http://") {
		return strings.TrimRight(g.Org, "/")
	}
	return "https://dev.azure.com/" + g.Org
}

// az runs one az call. --org is always explicit and --detect is off, so az never
// falls back to its own configured defaults or the git remote of the working
// directory — either of which can name a different organization than this
// project's config.
func (g Gate) az(args ...string) (string, error) {
	b, err := g.Run("az", args...)
	if err != nil {
		return "", fmt.Errorf("az %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(b)))
	}
	return strings.TrimSpace(string(b)), nil
}

// azFindPromotionPR returns the id of the single active dev→release PR, 0 when
// there is none, and an ambiguous-pr hold when there is more than one — the
// approval names a commit, and with two candidate PRs there is no single head to
// compare it against.
func (g Gate) azFindPromotionPR() (int, error) {
	out, err := g.az("repos", "pr", "list",
		"--org", g.orgURL(), "--project", g.Project, "--repository", g.Repo,
		"--source-branch", g.Dev, "--target-branch", g.Release, "--status", "active",
		"--detect", "false", "--only-show-errors", "-o", "json")
	if err != nil {
		return 0, err
	}
	var prs []struct {
		ID int `json:"pullRequestId"`
	}
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return 0, fmt.Errorf("parse az repos pr list: %w", err)
	}
	switch len(prs) {
	case 0:
		return 0, nil
	case 1:
		return prs[0].ID, nil
	}
	ids := make([]string, len(prs))
	for i, p := range prs {
		ids[i] = strconv.Itoa(p.ID)
	}
	return 0, &mergeHold{ReasonAmbiguousPR, fmt.Sprintf(
		"HOLD — %d open promotion PRs from `%s` into `%s` on %s (%s). An approval names one commit on one PR; with more than one there is nothing unambiguous to verify. Abandon the extras, or re-run with --pr <id>.",
		len(prs), g.Dev, g.Release, g.label(), strings.Join(ids, ", "))}
}

// azPR is the subset of `az repos pr show` the gate reads.
type azPR struct {
	Status                string `json:"status"`
	MergeStatus           string `json:"mergeStatus"`
	LastMergeSourceCommit struct {
		CommitID string `json:"commitId"`
	} `json:"lastMergeSourceCommit"`
}

func (g Gate) azShowPR(pr int) (azPR, error) {
	out, err := g.az("repos", "pr", "show", "--id", strconv.Itoa(pr),
		"--org", g.orgURL(), "--detect", "false", "--only-show-errors", "-o", "json")
	if err != nil {
		return azPR{}, err
	}
	var v azPR
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		return azPR{}, fmt.Errorf("parse az repos pr show: %w", err)
	}
	return v, nil
}

// azThreads is the subset of the pullRequestThreads response the records are read
// from. System comments (ref updates, status changes) and deleted ones are skipped.
type azThreads struct {
	Value []struct {
		IsDeleted bool `json:"isDeleted"`
		Comments  []struct {
			Content       string `json:"content"`
			PublishedDate string `json:"publishedDate"`
			CommentType   string `json:"commentType"`
			IsDeleted     bool   `json:"isDeleted"`
			Author        struct {
				UniqueName string `json:"uniqueName"`
			} `json:"author"`
		} `json:"comments"`
	} `json:"value"`
}

// azViewPR reads the PR's current head commit and every comment on it. Unlike
// GitHub this is two calls, not one snapshot; that is safe because nothing is
// merged on the strength of these reads alone — the completion re-reads the head
// immediately before it is sent and pins the verified commit, so a head that moved
// in between is refused by Azure itself.
func (g Gate) azViewPR(pr int) (string, []Comment, error) {
	v, err := g.azShowPR(pr)
	if err != nil {
		return "", nil, err
	}
	if v.Status != "active" {
		return "", nil, fmt.Errorf("pull request %d is %q, not active", pr, v.Status)
	}

	out, err := g.az("devops", "invoke", "--org", g.orgURL(),
		"--area", "git", "--resource", "pullRequestThreads",
		"--route-parameters", "project="+g.Project, "repositoryId="+g.Repo, "pullRequestId="+strconv.Itoa(pr),
		"--api-version", azAPIVersion, "--only-show-errors", "-o", "json")
	if err != nil {
		return "", nil, err
	}
	var th azThreads
	if err := json.Unmarshal([]byte(out), &th); err != nil {
		return "", nil, fmt.Errorf("parse pull request threads: %w", err)
	}

	type dated struct {
		c Comment
		t time.Time
	}
	var all []dated
	for _, t := range th.Value {
		if t.IsDeleted {
			continue
		}
		for _, c := range t.Comments {
			if c.IsDeleted || c.CommentType == "system" {
				continue
			}
			cm := Comment{Body: c.Content, CreatedAt: c.PublishedDate}
			cm.Author.Login = c.Author.UniqueName
			ts, _ := time.Parse(time.RFC3339Nano, c.PublishedDate)
			all = append(all, dated{cm, ts})
		}
	}
	// Oldest first, as ParseRecords expects. Threads come back in thread order, not
	// comment order, and published dates carry varying fractional digits, so sort
	// on the parsed time rather than the string.
	sort.SliceStable(all, func(i, j int) bool { return all[i].t.Before(all[j].t) })
	comments := make([]Comment, len(all))
	for i, d := range all {
		comments[i] = d.c
	}
	return v.LastMergeSourceCommit.CommitID, comments, nil
}

// azMerge completes the PR pinned to the verified commit, then waits until Azure
// reports it completed at that commit. It never sets autoComplete, never deletes
// the source branch, and never transitions work items.
func (g Gate) azMerge(pr int, sha string) error {
	if g.PAT == "" {
		ref := g.PATRef
		if ref == "" {
			ref = "AZURE_DEVOPS_PAT"
		}
		return &mergeHold{ReasonNoCredential, fmt.Sprintf(
			"HOLD — the approval for %s verified on %s#%d, but completing an Azure PR needs a personal access token and $%s (config.pat.ref) is not set in this environment. Nothing was promoted. Export it and re-run; the approval stays valid while the head is unchanged.",
			sha, g.label(), pr, ref)}
	}

	// Re-read the head immediately before the write and re-compare; never reuse the
	// value read earlier. The pin below would refuse a moved head anyway — this makes
	// the hold name the new commit instead of relaying a server error.
	now, err := g.azShowPR(pr)
	if err != nil {
		return &mergeHold{ReasonReadFailed, fmt.Sprintf(
			"HOLD — the approval for %s verified, but re-reading %s#%d immediately before completing it failed (%v). Nothing was promoted. Unverified is not approved.",
			sha, g.label(), pr, err)}
	}
	if now.LastMergeSourceCommit.CommitID != sha {
		return &mergeHold{ReasonMergeRefused, fmt.Sprintf(
			"HOLD — the approval for %s verified, but %s#%d's head moved to %s before the completion was sent, so `%s` advanced during the check. Nothing was promoted; re-run after adding a %s record for the new head.",
			sha, g.label(), pr, now.LastMergeSourceCommit.CommitID, g.Dev, sentinel)}
	}

	body, err := json.Marshal(map[string]any{
		"status":                "completed",
		"lastMergeSourceCommit": map[string]string{"commitId": sha},
		"completionOptions": map[string]any{
			"mergeStrategy":       "noFastForward",
			"deleteSourceBranch":  false,
			"transitionWorkItems": false,
		},
	})
	if err != nil {
		return err
	}
	do := g.HTTP
	if do == nil {
		do = HTTPRequest
	}
	u := fmt.Sprintf("%s/%s/_apis/git/repositories/%s/pullrequests/%d?api-version=%s",
		g.orgURL(), url.PathEscape(g.Project), url.PathEscape(g.Repo), pr, azAPIVersion)
	code, resp, err := do(http.MethodPatch, u, body, g.PAT)
	if err != nil {
		return fmt.Errorf("PATCH pull request %d: %w", pr, err)
	}
	if code < 200 || code > 299 {
		return fmt.Errorf("HTTP %d: %s", code, azErrorMessage(resp))
	}

	// Azure completes asynchronously: the PATCH answers with the PR still active and
	// mergeStatus "queued". Only a PR reported completed at the approved commit is a
	// promotion.
	sleep := g.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	attempts := g.PollAttempts
	if attempts <= 0 {
		attempts = 30
	}
	last := "not read"
	for i := 0; i < attempts; i++ {
		if i > 0 {
			sleep(2 * time.Second)
		}
		v, err := g.azShowPR(pr)
		if err != nil {
			last = err.Error()
			continue
		}
		last = fmt.Sprintf("status %s, mergeStatus %s", v.Status, v.MergeStatus)
		switch {
		case v.Status == "completed" && v.LastMergeSourceCommit.CommitID == sha:
			return nil
		case v.Status == "completed":
			return &mergeHold{ReasonMergeUnconfirmed, fmt.Sprintf(
				"HOLD — %s#%d reports completed, but at source commit %s, not the approved %s. Inspect the PR before anything else is promoted.",
				g.label(), pr, v.LastMergeSourceCommit.CommitID, sha)}
		case v.Status == "abandoned":
			return fmt.Errorf("the pull request was abandoned")
		case v.MergeStatus == "conflicts" || v.MergeStatus == "failure" || v.MergeStatus == "rejectedByPolicy":
			return fmt.Errorf("Azure reports mergeStatus %s", v.MergeStatus)
		}
	}
	return &mergeHold{ReasonMergeUnconfirmed, fmt.Sprintf(
		"HOLD — the completion of %s#%d at %s was accepted, but Azure had not reported the PR completed after %d checks (last: %s). I am not reporting a promotion I have not seen. Check the PR, then re-run: a completed PR has nothing left to merge.",
		g.label(), pr, sha, attempts, last)}
}

// azErrorMessage pulls the human message out of an Azure DevOps error body
// ({"message": "TF401192: ...", ...}), falling back to the raw body.
func azErrorMessage(b []byte) string {
	var e struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(b, &e) == nil && e.Message != "" {
		return e.Message
	}
	return strings.TrimSpace(string(b))
}
