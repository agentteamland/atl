// Package promotiongate is the deterministic dev→release promotion gate.
//
// The delivery-team promotes a sprint by merging the dev→release PR, and that
// merge is authorized by a commit-bound approval record the human product owner
// posts on that PR (backend-interface concept #16): a comment whose FIRST line is
// the exact sentinel `**[Promotion Approval]**`, naming the approved commit under
// a `## Approved Commit` heading. The record authorizes ONE commit — an approval
// naming anything other than the PR's current head is not an approval for what
// would actually be promoted, and is never carried forward.
//
// That verification first shipped as skill prose, and a real run proved prose is
// followed inconsistently: on one turn the ceremony held correctly on a moved
// head, on the very next it never mentioned the record and fell back to asking the
// human to approve conversationally — the exact gate the design removes. Whatever
// depends on remembering gets forgotten, and the failure is silent. So the
// decision lives here, in code.
//
// Verify VERIFIES AND MERGES in one call on purpose: a separate "now merge" step
// is a step a caller can reach without the check.
package promotiongate

import (
	"fmt"
	"regexp"
	"strings"
)

// Verdicts. A promotion either happened or it did not — a hold is not a reject:
// nothing is tagged carryover, no rejection is filed, the gate simply waits.
const (
	Promoted = "promoted"
	Hold     = "hold"
)

// Hold reasons — the machine-readable half of a hold, for the ceremony to relay
// and the e2e to assert on. Result.Message is the human half.
const (
	ReasonBackendUnbound = "backend-unbound"
	ReasonNoOpenPR       = "no-open-pr"
	ReasonNoRecord       = "no-record"
	ReasonUnparseable    = "unparseable-record"
	ReasonSuperseded     = "superseded"
	ReasonReadFailed     = "read-failed"
	ReasonMergeRefused   = "merge-refused"
)

// The record's locators, byte-for-byte as knowledge/backend-interface.md #16 and
// backends/github/adapter.md §7 define them, and as the PO is told to write them.
// Changing either here without changing them there breaks the contract silently.
const (
	sentinel = "**[Promotion Approval]**"
	heading  = "## Approved Commit"
)

// commitRe matches a 40-character lowercase hex commit id. Lowercase-only and
// word-bounded on purpose: the contract says "a 40-character lowercase hex id", so
// an uppercase or wrong-length id is NOT a commit id — the record reads as
// unparseable and the PO is asked to re-post it, rather than being silently
// coerced into something that might not name the commit they meant.
var commitRe = regexp.MustCompile(`\b[0-9a-f]{40}\b`)

// Comment is the subset of `gh pr view --json comments` a record is read from.
type Comment struct {
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
}

// Record is one parsed approval record. Commit is "" when the record carries no
// readable id — a different outcome from "no record at all", so the PO is told
// which of the two actually happened.
type Record struct {
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt"`
	Commit    string `json:"commit"`
}

// Result is the gate's full verdict — the JSON the ceremony relays verbatim.
type Result struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message"`
	PR      int    `json:"pr,omitempty"`
	Head    string `json:"head,omitempty"`
	// Approved is the commit an approval record names. On a promote it is the
	// merged commit; on a superseded hold it is the STALE one the record named.
	// Verdict is the authority, never this field's presence.
	Approved string   `json:"approved,omitempty"`
	Records  []Record `json:"records"`
}

// Gate is one promotion decision: the repo and branch pair to act on, plus the
// exec seam every gh call goes through.
type Gate struct {
	Owner   string
	Repo    string
	Dev     string
	Release string
	PR      int // 0 = resolve the open dev→release PR
	Run     Runner
}

// ParseRecords collects EVERY approval record on the PR, in the order gh returns
// comments (oldest first). The approval channel is append-and-supersede, so
// converging on a single comment — the read-back rule for every other sentinel —
// is wrong here: selection happens later, by commit id, never by recency.
func ParseRecords(comments []Comment) []Record {
	var out []Record
	for _, c := range comments {
		if !hasSentinel(c.Body) {
			continue
		}
		out = append(out, Record{
			Author:    c.Author.Login,
			CreatedAt: c.CreatedAt,
			Commit:    approvedCommit(c.Body),
		})
	}
	return out
}

// hasSentinel reports whether the body's FIRST line is the sentinel — not whether
// the body contains it. A comment that merely quotes the sentinel mid-text (a PO
// explaining the format, a ceremony echoing a hold message back onto the PR) is
// not a record and must never authorize a merge. Surrounding whitespace on that
// line is tolerated because GitHub returns comment bodies with CRLF endings, which
// an exact byte compare would reject for every real record.
func hasSentinel(body string) bool {
	first, _, _ := strings.Cut(body, "\n")
	return strings.TrimSpace(first) == sentinel
}

// approvedCommit returns the first 40-char lowercase hex id on a line AFTER the
// `## Approved Commit` heading, or "" when the heading is missing or nothing under
// it parses. Scanning to the end of the body rather than stopping at the next
// heading is what the adapter contract says ("the first ... id on a line after that
// record's `## Approved Commit` heading"); everything after the heading is the
// record's payload and the rest of the body is audit context.
func approvedCommit(body string) string {
	lines := strings.Split(body, "\n")
	for i, ln := range lines {
		if strings.TrimSpace(ln) != heading {
			continue
		}
		for _, rest := range lines[i+1:] {
			if m := commitRe.FindString(rest); m != "" {
				return m
			}
		}
		return ""
	}
	return ""
}

// Verify resolves the promotion PR, compares the approval records against the PR's
// current head, and merges only on an exact match. Every other outcome is a hold:
// no merge, no state change, nothing carried forward.
func (g Gate) Verify() Result {
	repo := g.Owner + "/" + g.Repo

	pr := g.PR
	if pr == 0 {
		n, err := g.findPromotionPR(repo)
		if err != nil {
			return holdf(ReasonReadFailed, "HOLD — could not read the open promotion PR from `%s` into `%s` on %s (%v). Unverified is not approved.", g.Dev, g.Release, repo, err)
		}
		if n == 0 {
			return holdf(ReasonNoOpenPR, "HOLD — no open promotion PR from `%s` into `%s` on %s. Opening that PR is not the promotion, but there is nothing to verify or merge until it exists: open it first (/sprint-review step 6a), then re-run.", g.Dev, g.Release, repo)
		}
		pr = n
	}

	head, comments, err := g.viewPR(repo, pr)
	if err != nil {
		res := holdf(ReasonReadFailed, "HOLD — could not read the promotion PR's approval record and current head commit (%v). Unverified is not approved.", err)
		res.PR = pr
		return res
	}
	// An empty head is a read failure, never a value to compare against. Rejecting
	// it here is also what stops a record with no parseable commit id ("") from
	// matching an unread head ("") below and merging on nothing.
	if head == "" {
		res := holdf(ReasonReadFailed, "HOLD — the promotion PR %s#%d reported no head commit, so there is nothing to verify your approval against. Unverified is not approved.", repo, pr)
		res.PR = pr
		return res
	}

	records := ParseRecords(comments)
	res := Result{Verdict: Hold, PR: pr, Head: head, Records: records}

	if len(records) == 0 {
		res.Reason = ReasonNoRecord
		res.Message = fmt.Sprintf("HOLD — no promotion approval on record. The promotion PR is %s#%d, head commit %s. To approve, add a comment to that PR whose first line is exactly %s, followed by a %q heading and %s. I will verify it and merge. I will not promote on a conversational approval.",
			repo, pr, head, sentinel, heading, head)
		return res
	}

	for _, r := range records {
		if r.Commit != head {
			continue
		}
		if err := g.merge(repo, pr, head); err != nil {
			res.Reason = ReasonMergeRefused
			res.Approved = head
			res.Message = fmt.Sprintf("HOLD — the approval for %s verified, but the merge of %s#%d was refused (%v). --match-head-commit means the provider itself refuses to merge anything but that commit, so `dev` most likely advanced between the check and the merge. Nothing was promoted; re-run after adding a %s record for the new head.",
				head, repo, pr, err, sentinel)
			return res
		}
		res.Verdict = Promoted
		res.Reason = ""
		res.Approved = head
		res.Message = fmt.Sprintf("Promoted — %s#%d merged at commit %s, approved by %s at %s.", repo, pr, head, r.Author, r.CreatedAt)
		return res
	}

	// Nothing names the current head. Report the NEWEST record that carries an id,
	// because that is the approval the PO most likely just posted and the one they
	// need to see named back. Selection for approval above is by commit id and never
	// by recency; recency only decides which stale record is worth quoting.
	var named *Record
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Commit != "" {
			named = &records[i]
			break
		}
	}
	if named == nil {
		// Records exist but not one of them yields an id — unreadable, not superseded.
		last := records[len(records)-1]
		res.Reason = ReasonUnparseable
		res.Message = fmt.Sprintf("HOLD — a %s record exists on PR %s#%d (by %s, %s) but I cannot read a commit id from it. Expected a 40-character lowercase hex id on a line after %q. Re-post the record for %s.",
			sentinel, repo, pr, last.Author, last.CreatedAt, heading, head)
		return res
	}

	res.Reason = ReasonSuperseded
	res.Approved = named.Commit
	res.Message = fmt.Sprintf(`HOLD — the approval on record is for commit %s
(set by %s at %s), but the promotion PR's head is now %s.
`+"`%s`"+` has advanced since that approval, so the approved state is not the state
that would be promoted. I have NOT promoted anything and I have NOT carried the
approval forward.
To promote the current state, re-read the refreshed report above and add a new
%s record for %s. To promote exactly what you
approved, reset `+"`%s`"+` to %s first.`,
		named.Commit, named.Author, named.CreatedAt, head, g.Dev, sentinel, head, g.Dev, named.Commit)
	return res
}

// HoldBackendUnbound is the verdict for a backend that binds only the record leg
// of concept #16. On Azure the head-commit read is unresolved — which response
// field of the branch read carries the commit id is not evidenced — so there is
// nothing to compare an approved commit against. Unverified is never approved, so
// the gate holds there; it does not fall back to a conversational approval and it
// does not substitute a local git read for the adapter.
func HoldBackendUnbound(backend string) Result {
	return holdf(ReasonBackendUnbound, `HOLD — on the %s backend I cannot read the promotion PR's current head commit, so I
cannot verify that your approval names the commit that would be promoted. I have NOT
promoted anything; unverified is not approved. The promotion PR and its approval
record (if any) are preserved. Unblocking this needs the head-commit read bound in
the Azure adapter — resolve the branch read's response field against a live server,
then bind it.`, backend)
}

func holdf(reason, format string, a ...any) Result {
	return Result{Verdict: Hold, Reason: reason, Message: fmt.Sprintf(format, a...)}
}
