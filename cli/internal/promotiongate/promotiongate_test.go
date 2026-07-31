package promotiongate

import (
	"errors"
	"strings"
	"testing"
)

const (
	shaA = "738cdb2a1f4e6b90c3d5827a4e1b06f9d3a7c215"
	shaB = "1abfcf1e02d47b6a95c8130fe27b4a6d8091c3ee"
)

func comment(author, ts, body string) Comment {
	c := Comment{Body: body, CreatedAt: ts}
	c.Author.Login = author
	return c
}

func record(sha string) string {
	return sentinel + "\n\n" + heading + "\n" + sha + "\n\n## Decision\nAPPROVE\n"
}

// TestParseRecords pins the record contract: which comments are records at all,
// and which commit id each one yields.
func TestParseRecords(t *testing.T) {
	cases := []struct {
		name     string
		comments []Comment
		want     []Record
	}{
		{
			name:     "well-formed record",
			comments: []Comment{comment("po", "2026-07-31T10:00:00Z", record(shaA))},
			want:     []Record{{Author: "po", CreatedAt: "2026-07-31T10:00:00Z", Commit: shaA}},
		},
		{
			name: "multiple records are all collected, oldest first",
			comments: []Comment{
				comment("po", "t1", record(shaA)),
				comment("dev", "t2", "just a normal comment"),
				comment("po", "t3", record(shaB)),
			},
			want: []Record{
				{Author: "po", CreatedAt: "t1", Commit: shaA},
				{Author: "po", CreatedAt: "t3", Commit: shaB},
			},
		},
		{
			name:     "CRLF body still parses (GitHub returns CRLF)",
			comments: []Comment{comment("po", "t", strings.ReplaceAll(record(shaA), "\n", "\r\n"))},
			want:     []Record{{Author: "po", CreatedAt: "t", Commit: shaA}},
		},
		{
			name:     "sentinel mid-text is NOT a record",
			comments: []Comment{comment("po", "t", "As agreed, post a "+sentinel+" comment.\n\n"+heading+"\n"+shaA)},
			want:     nil,
		},
		{
			name:     "sentinel on a later line is NOT a record",
			comments: []Comment{comment("po", "t", "Approving now.\n"+sentinel+"\n"+heading+"\n"+shaA)},
			want:     nil,
		},
		{
			name:     "missing heading yields no commit",
			comments: []Comment{comment("po", "t", sentinel+"\n\nLooks good, "+shaA+"\n")},
			want:     []Record{{Author: "po", CreatedAt: "t", Commit: ""}},
		},
		{
			name:     "heading with no id yields no commit",
			comments: []Comment{comment("po", "t", sentinel+"\n\n"+heading+"\n\n## Decision\nAPPROVE\n")},
			want:     []Record{{Author: "po", CreatedAt: "t", Commit: ""}},
		},
		{
			name:     "uppercase hex is not a commit id",
			comments: []Comment{comment("po", "t", sentinel+"\n\n"+heading+"\n"+strings.ToUpper(shaA)+"\n")},
			want:     []Record{{Author: "po", CreatedAt: "t", Commit: ""}},
		},
		{
			name:     "short hex is not a commit id",
			comments: []Comment{comment("po", "t", sentinel+"\n\n"+heading+"\n"+shaA[:12]+"\n")},
			want:     []Record{{Author: "po", CreatedAt: "t", Commit: ""}},
		},
		{
			name:     "id before the heading is ignored — only lines after it count",
			comments: []Comment{comment("po", "t", sentinel+"\n\n## Sprint\nSprint 4 · "+shaB+"\n\n"+heading+"\n"+shaA+"\n")},
			want:     []Record{{Author: "po", CreatedAt: "t", Commit: shaA}},
		},
		{
			name:     "no comments at all",
			comments: nil,
			want:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseRecords(tc.comments)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d record(s) %+v, want %d %+v", len(got), got, len(tc.want), tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("record %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// fakeGH answers gh calls from canned stdout keyed by the subcommand ("pr list",
// "pr view", "pr merge") and records every argv, so a test asserts the exact call
// the gate made — no network, and a merge that never merges anything.
type fakeGH struct {
	calls [][]string
	out   map[string]string
	err   map[string]error
}

func (f *fakeGH) runner(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	key := strings.Join(args[:min(2, len(args))], " ")
	return []byte(f.out[key]), f.err[key]
}

func (f *fakeGH) merged() bool {
	for _, c := range f.calls {
		if len(c) >= 3 && c[1] == "pr" && c[2] == "merge" {
			return true
		}
	}
	return false
}

func viewJSON(head string, bodies ...string) string {
	var sb strings.Builder
	sb.WriteString(`{"headRefOid":"` + head + `","comments":[`)
	for i, b := range bodies {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"author":{"login":"po"},"createdAt":"2026-07-31T10:0` + string(rune('0'+i)) + `:00Z","body":`)
		sb.WriteString(quote(b))
		sb.WriteString("}")
	}
	sb.WriteString("]}")
	return sb.String()
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `"`, `\"`), "\n", `\n`) + `"`
}

func newGate(f *fakeGH) Gate {
	return Gate{Owner: "acme", Repo: "widgets", Dev: "dev", Release: "release", Run: f.runner}
}

// TestVerifyPromotes proves the one promoting path: a record naming the CURRENT
// head merges, pinned to that commit, with --merge (never squash/rebase).
func TestVerifyPromotes(t *testing.T) {
	f := &fakeGH{out: map[string]string{
		"pr list": `[{"number":42}]`,
		"pr view": viewJSON(shaA, record(shaA)),
	}}
	res := newGate(f).Verify()

	if res.Verdict != Promoted {
		t.Fatalf("verdict = %q (%s), want promoted: %s", res.Verdict, res.Reason, res.Message)
	}
	if res.PR != 42 || res.Head != shaA || res.Approved != shaA {
		t.Errorf("pr=%d head=%q approved=%q", res.PR, res.Head, res.Approved)
	}
	if len(res.Records) != 1 || res.Records[0].Author != "po" {
		t.Errorf("records = %+v, want one authored record", res.Records)
	}
	want := "gh pr merge 42 --repo acme/widgets --merge --match-head-commit " + shaA
	var got string
	for _, c := range f.calls {
		if len(c) >= 3 && c[2] == "merge" {
			got = strings.Join(c, " ")
		}
	}
	if got != want {
		t.Errorf("merge argv:\n got %q\nwant %q", got, want)
	}
}

// TestVerifyHolds covers every non-promoting outcome. Each one must merge nothing.
func TestVerifyHolds(t *testing.T) {
	cases := []struct {
		name       string
		fake       *fakeGH
		wantReason string
		wantIn     string // a phrase the PO-facing message must carry
	}{
		{
			name:       "no open promotion PR",
			fake:       &fakeGH{out: map[string]string{"pr list": `[]`}},
			wantReason: ReasonNoOpenPR,
			wantIn:     "no open promotion PR",
		},
		{
			name: "PR list read failed",
			fake: &fakeGH{
				out: map[string]string{"pr list": "gh: could not resolve repo"},
				err: map[string]error{"pr list": errors.New("exit status 1")},
			},
			wantReason: ReasonReadFailed,
			wantIn:     "Unverified is not approved",
		},
		{
			name: "PR view read failed",
			fake: &fakeGH{
				out: map[string]string{"pr list": `[{"number":42}]`, "pr view": "API rate limit exceeded"},
				err: map[string]error{"pr view": errors.New("exit status 1")},
			},
			wantReason: ReasonReadFailed,
			wantIn:     "Unverified is not approved",
		},
		{
			name: "no head commit in the response is a read failure, not a match",
			fake: &fakeGH{out: map[string]string{
				"pr list": `[{"number":42}]`,
				// A record with no parseable id alongside an unread head: the two "" values
				// must never compare equal into a merge.
				"pr view": viewJSON("", sentinel+"\n\n"+heading+"\n\n"),
			}},
			wantReason: ReasonReadFailed,
			wantIn:     "no head commit",
		},
		{
			name: "no approval record",
			fake: &fakeGH{out: map[string]string{
				"pr list": `[{"number":42}]`,
				"pr view": viewJSON(shaA, "LGTM, ship it"),
			}},
			wantReason: ReasonNoRecord,
			wantIn:     "I will not promote on a conversational approval",
		},
		{
			name: "record carries no parseable commit id",
			fake: &fakeGH{out: map[string]string{
				"pr list": `[{"number":42}]`,
				"pr view": viewJSON(shaA, sentinel+"\n\n"+heading+"\n\n## Decision\nAPPROVE\n"),
			}},
			wantReason: ReasonUnparseable,
			wantIn:     "cannot read a commit id",
		},
		{
			name: "record names a superseded commit — dev advanced",
			fake: &fakeGH{out: map[string]string{
				"pr list": `[{"number":42}]`,
				"pr view": viewJSON(shaB, record(shaA)),
			}},
			wantReason: ReasonSuperseded,
			wantIn:     shaA,
		},
		{
			name: "every record is superseded — the newest parseable one is quoted",
			fake: &fakeGH{out: map[string]string{
				"pr list": `[{"number":42}]`,
				"pr view": viewJSON("cccccccccccccccccccccccccccccccccccccccc", record(shaA), record(shaB)),
			}},
			wantReason: ReasonSuperseded,
			wantIn:     shaB,
		},
		{
			name: "merge refused by the provider is not a promotion",
			fake: &fakeGH{
				out: map[string]string{
					"pr list":  `[{"number":42}]`,
					"pr view":  viewJSON(shaA, record(shaA)),
					"pr merge": "Head branch was modified. Review and try the merge again.",
				},
				err: map[string]error{"pr merge": errors.New("exit status 1")},
			},
			wantReason: ReasonMergeRefused,
			wantIn:     "Nothing was promoted",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := newGate(tc.fake).Verify()
			if res.Verdict != Hold {
				t.Fatalf("verdict = %q, want hold: %s", res.Verdict, res.Message)
			}
			if res.Reason != tc.wantReason {
				t.Errorf("reason = %q, want %q (message: %s)", res.Reason, tc.wantReason, res.Message)
			}
			if !strings.Contains(res.Message, tc.wantIn) {
				t.Errorf("message does not carry %q:\n%s", tc.wantIn, res.Message)
			}
			// The one invariant every hold shares.
			if tc.wantReason != ReasonMergeRefused && tc.fake.merged() {
				t.Error("a hold merged the PR")
			}
		})
	}
}

// TestVerifyPRoverride proves --pr skips the branch-pair lookup and verifies the
// named PR instead.
func TestVerifyPRoverride(t *testing.T) {
	f := &fakeGH{out: map[string]string{"pr view": viewJSON(shaA, record(shaA))}}
	g := newGate(f)
	g.PR = 7
	res := g.Verify()

	if res.Verdict != Promoted || res.PR != 7 {
		t.Fatalf("verdict=%q pr=%d: %s", res.Verdict, res.PR, res.Message)
	}
	for _, c := range f.calls {
		if len(c) >= 3 && c[2] == "list" {
			t.Error("--pr must skip the open-PR lookup")
		}
	}
}

// TestVerifyReadArgv pins the two READ calls byte for byte, because both are
// load-bearing contracts that no other assertion covers and whose breakage is
// SILENT — it degrades to a permanent hold ("no open promotion PR", "no head
// commit") while the PR sits right there, i.e. the gate stops gating and looks
// like it is working.
//
//   - `pr list` must query the configured branch PAIR in the right direction
//     (--base <release> --head <dev>). Swap them and the promotion PR is never
//     found. Non-default names here also prove the pair comes from the config
//     rather than being hardcoded.
//   - `pr view` must read `headRefOid,comments` in ONE call. That is not economy:
//     it is what makes the head and the records it is compared against come from
//     the same snapshot, which every surface documents as the reason a comment
//     posted mid-check cannot make the comparison lie. Splitting it into two
//     calls, or dropping headRefOid, breaks that with nothing else to notice.
func TestVerifyReadArgv(t *testing.T) {
	f := &fakeGH{out: map[string]string{
		"pr list": `[{"number":42}]`,
		"pr view": viewJSON(shaA, record(shaA)),
	}}
	g := newGate(f)
	g.Dev, g.Release = "trunk", "prod" // not the defaults — a hardcoded pair would show
	if res := g.Verify(); res.Verdict != Promoted {
		t.Fatalf("verdict=%q: %s", res.Verdict, res.Message)
	}

	want := []string{
		"gh pr list --repo acme/widgets --state open --base prod --head trunk --json number",
		"gh pr view 42 --repo acme/widgets --json headRefOid,comments",
		"gh pr merge 42 --repo acme/widgets --merge --match-head-commit " + shaA,
	}
	if len(f.calls) != len(want) {
		t.Fatalf("made %d gh call(s), want exactly %d: %v", len(f.calls), len(want), f.calls)
	}
	for i, w := range want {
		if got := strings.Join(f.calls[i], " "); got != w {
			t.Errorf("gh call %d:\n got %q\nwant %q", i, got, w)
		}
	}
}

// TestHoldBackendUnbound proves a non-GitHub backend holds rather than falling
// back to a conversational promotion.
func TestHoldBackendUnbound(t *testing.T) {
	res := HoldBackendUnbound("azure")
	if res.Verdict != Hold || res.Reason != ReasonBackendUnbound {
		t.Fatalf("verdict=%q reason=%q", res.Verdict, res.Reason)
	}
	if !strings.Contains(res.Message, "cannot read the promotion PR's current head commit") {
		t.Errorf("message: %s", res.Message)
	}
}
