package promotiongate

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// The fixtures below are shaped on what a live Azure DevOps server returned
// (2026-11-17): `az repos pr list` rows carry pullRequestId; the head
// is lastMergeSourceCommit.commitId; thread comments carry content, publishedDate,
// commentType ("text" | "system") and author.uniqueName; a completion pinned to a
// stale commit is refused with HTTP 409 and the body below.

const azStaleBody = `{"$id":"1","innerException":null,"message":"TF401192: The source branch has been modified since the last merge attempt.","typeName":"Microsoft.TeamFoundation.Git.Server.GitPullRequestStaleException, Microsoft.TeamFoundation.Git.Server","typeKey":"GitPullRequestStaleException","errorCode":0,"eventId":3000}`

// fakeAZ answers az calls keyed by the first three args ("repos pr list",
// "repos pr show", "devops invoke --org"). `pr show` is called several times (the
// view read, the re-read before completing, the completion polls), so its answers
// are a queue whose last entry repeats.
type fakeAZ struct {
	calls   [][]string
	list    string
	listErr error
	show    []string
	showErr error
	threads string

	httpCalls []httpCall
	httpCode  int
	httpBody  string
}

type httpCall struct {
	method, url, pat string
	body             []byte
}

func (f *fakeAZ) runner(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	switch strings.Join(args[:min(3, len(args))], " ") {
	case "repos pr list":
		return []byte(f.list), f.listErr
	case "repos pr show":
		if f.showErr != nil {
			return []byte("ERROR: TF401180: The requested pull request was not found."), f.showErr
		}
		out := f.show[0]
		if len(f.show) > 1 {
			f.show = f.show[1:]
		}
		return []byte(out), nil
	case "devops invoke --org":
		return []byte(f.threads), nil
	}
	return nil, errors.New("unexpected az call: " + strings.Join(args, " "))
}

func (f *fakeAZ) http(method, url string, body []byte, pat string) (int, []byte, error) {
	f.httpCalls = append(f.httpCalls, httpCall{method, url, pat, body})
	code := f.httpCode
	if code == 0 {
		code = 200
	}
	return code, []byte(f.httpBody), nil
}

func azShow(status, mergeStatus, head string) string {
	return `{"pullRequestId":1457,"status":"` + status + `","mergeStatus":"` + mergeStatus +
		`","sourceRefName":"refs/heads/dev","targetRefName":"refs/heads/release",` +
		`"lastMergeSourceCommit":{"commitId":"` + head + `"}}`
}

type azComment struct {
	content, published, kind, author string
	deleted                          bool
}

func azThreadsJSON(cs ...azComment) string {
	type author struct {
		UniqueName  string `json:"uniqueName"`
		DisplayName string `json:"displayName"`
	}
	type comment struct {
		ID            int    `json:"id"`
		Content       string `json:"content"`
		PublishedDate string `json:"publishedDate"`
		CommentType   string `json:"commentType"`
		IsDeleted     bool   `json:"isDeleted,omitempty"`
		Author        author `json:"author"`
	}
	type thread struct {
		ID        int       `json:"id"`
		IsDeleted bool      `json:"isDeleted"`
		Comments  []comment `json:"comments"`
	}
	doc := struct {
		Token *string  `json:"continuation_token"`
		Count int      `json:"count"`
		Value []thread `json:"value"`
	}{Count: len(cs)}
	for i, c := range cs {
		doc.Value = append(doc.Value, thread{ID: 2419 + i, Comments: []comment{{
			ID: 1, Content: c.content, PublishedDate: c.published, CommentType: c.kind, IsDeleted: c.deleted,
			Author: author{UniqueName: c.author, DisplayName: c.author},
		}}})
	}
	b, _ := json.Marshal(doc)
	return string(b)
}

var azSystem = azComment{"The reference refs/heads/dev was updated.", "2026-11-17T16:28:03.903Z", "system", "Microsoft.VisualStudio.Services.TFS", false}

func azApproval(sha, published string) azComment {
	return azComment{record(sha), published, "text", "po@example.com", false}
}

func newAzGate(f *fakeAZ) Gate {
	return Gate{
		Backend: "azure", Org: "acme", Project: "Widgets", Repo: "Widgets",
		Dev: "dev", Release: "release", Run: f.runner,
		PAT: "test-pat", PATRef: "AZURE_DEVOPS_PAT", HTTP: f.http,
		Sleep: func(time.Duration) {}, PollAttempts: 3,
	}
}

// TestAzureVerifyPromotes is the one promoting path on Azure: a record naming the
// current head completes the PR through one PATCH pinned to that commit, and the
// gate reports promoted only once Azure reports the PR completed at it.
func TestAzureVerifyPromotes(t *testing.T) {
	f := &fakeAZ{
		list: `[{"pullRequestId":1457}]`,
		show: []string{
			azShow("active", "succeeded", shaA), // view read
			azShow("active", "succeeded", shaA), // re-read before completing
			azShow("active", "queued", shaA),    // poll: the PATCH answered, merge still queued
			azShow("completed", "succeeded", shaA),
		},
		threads: azThreadsJSON(azSystem, azApproval(shaA, "2026-11-17T16:35:02.993Z")),
	}
	res := newAzGate(f).Verify()

	if res.Verdict != Promoted {
		t.Fatalf("verdict = %q (%s), want promoted: %s", res.Verdict, res.Reason, res.Message)
	}
	if res.PR != 1457 || res.Head != shaA || res.Approved != shaA {
		t.Errorf("pr=%d head=%q approved=%q", res.PR, res.Head, res.Approved)
	}
	if len(res.Records) != 1 || res.Records[0].Author != "po@example.com" {
		t.Errorf("records = %+v, want the one text-comment record (system comments skipped)", res.Records)
	}

	if len(f.httpCalls) != 1 {
		t.Fatalf("made %d completion request(s), want exactly 1", len(f.httpCalls))
	}
	c := f.httpCalls[0]
	if c.method != "PATCH" || c.pat != "test-pat" {
		t.Errorf("method=%q pat=%q", c.method, c.pat)
	}
	wantURL := "https://dev.azure.com/acme/Widgets/_apis/git/repositories/Widgets/pullrequests/1457?api-version=7.1"
	if c.url != wantURL {
		t.Errorf("url:\n got %q\nwant %q", c.url, wantURL)
	}
	var body map[string]any
	if err := json.Unmarshal(c.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "completed" {
		t.Errorf("status = %v", body["status"])
	}
	if pin := body["lastMergeSourceCommit"].(map[string]any)["commitId"]; pin != shaA {
		t.Errorf("completion pinned to %v, want the verified %s", pin, shaA)
	}
	opts := body["completionOptions"].(map[string]any)
	if opts["mergeStrategy"] != "noFastForward" || opts["deleteSourceBranch"] != false || opts["transitionWorkItems"] != false {
		t.Errorf("completionOptions = %v", opts)
	}
	if _, ok := body["autoCompleteSetBy"]; ok {
		t.Error("the completion must never set autoComplete")
	}
}

// TestAzureReadArgv pins the az reads byte for byte. --org must be explicit and
// --detect off: on a machine whose az defaults name another organization, an
// implicit org would read (and gate) the wrong project with nothing to notice.
func TestAzureReadArgv(t *testing.T) {
	f := &fakeAZ{
		list:    `[{"pullRequestId":1457}]`,
		show:    []string{azShow("active", "succeeded", shaA), azShow("active", "succeeded", shaA), azShow("completed", "succeeded", shaA)},
		threads: azThreadsJSON(azApproval(shaA, "2026-11-17T16:35:02.993Z")),
	}
	g := newAzGate(f)
	g.Dev, g.Release = "trunk", "prod"
	if res := g.Verify(); res.Verdict != Promoted {
		t.Fatalf("verdict=%q: %s", res.Verdict, res.Message)
	}
	want := []string{
		"az repos pr list --org https://dev.azure.com/acme --project Widgets --repository Widgets --source-branch trunk --target-branch prod --status active --detect false --only-show-errors -o json",
		"az repos pr show --id 1457 --org https://dev.azure.com/acme --detect false --only-show-errors -o json",
		"az devops invoke --org https://dev.azure.com/acme --area git --resource pullRequestThreads --route-parameters project=Widgets repositoryId=Widgets pullRequestId=1457 --api-version 7.1 --only-show-errors -o json",
	}
	for i, w := range want {
		if got := strings.Join(f.calls[i], " "); got != w {
			t.Errorf("az call %d:\n got %q\nwant %q", i, got, w)
		}
	}
	for _, c := range f.calls {
		if strings.Contains(strings.Join(c, " "), "test-pat") {
			t.Fatal("the PAT appeared in an az argv")
		}
	}
}

// TestAzureVerifyHolds covers every Azure non-promoting outcome.
func TestAzureVerifyHolds(t *testing.T) {
	active := func(head string) []string { return []string{azShow("active", "succeeded", head)} }
	cases := []struct {
		name       string
		fake       *fakeAZ
		gate       func(*Gate)
		wantReason string
		wantIn     string
		wantPatch  bool
	}{
		{
			name:       "no open promotion PR",
			fake:       &fakeAZ{list: `[]`},
			wantReason: ReasonNoOpenPR,
			wantIn:     "no open promotion PR",
		},
		{
			name:       "two open promotion PRs",
			fake:       &fakeAZ{list: `[{"pullRequestId":1457},{"pullRequestId":1462}]`},
			wantReason: ReasonAmbiguousPR,
			wantIn:     "1457, 1462",
		},
		{
			name:       "PR list read failed",
			fake:       &fakeAZ{list: "ERROR: TF400813", listErr: errors.New("exit status 1")},
			wantReason: ReasonReadFailed,
			wantIn:     "Unverified is not approved",
		},
		{
			name:       "PR show read failed",
			fake:       &fakeAZ{list: `[{"pullRequestId":1457}]`, showErr: errors.New("exit status 1")},
			wantReason: ReasonReadFailed,
			wantIn:     "Unverified is not approved",
		},
		{
			name:       "named PR is not active",
			fake:       &fakeAZ{show: []string{azShow("completed", "succeeded", shaA)}},
			gate:       func(g *Gate) { g.PR = 1460 },
			wantReason: ReasonReadFailed,
			wantIn:     "not active",
		},
		{
			name:       "no approval record — only system comments and a conversational OK",
			fake:       &fakeAZ{list: `[{"pullRequestId":1457}]`, show: active(shaA), threads: azThreadsJSON(azSystem, azComment{"LGTM, ship it", "2026-11-17T16:40:00Z", "text", "po@example.com", false})},
			wantReason: ReasonNoRecord,
			wantIn:     "I will not promote on a conversational approval",
		},
		{
			name: "a deleted record does not count",
			fake: &fakeAZ{list: `[{"pullRequestId":1457}]`, show: active(shaA), threads: azThreadsJSON(
				azComment{record(shaA), "2026-11-17T16:35:02.993Z", "text", "po@example.com", true})},
			wantReason: ReasonNoRecord,
			wantIn:     "no promotion approval on record",
		},
		{
			name:       "record names a superseded commit — dev advanced",
			fake:       &fakeAZ{list: `[{"pullRequestId":1457}]`, show: active(shaB), threads: azThreadsJSON(azApproval(shaA, "2026-11-17T16:35:02.993Z"))},
			wantReason: ReasonSuperseded,
			wantIn:     shaA,
		},
		{
			name: "newest stale record is quoted even when threads come back out of date order",
			fake: &fakeAZ{list: `[{"pullRequestId":1457}]`, show: active("cccccccccccccccccccccccccccccccccccccccc"), threads: azThreadsJSON(
				azApproval(shaB, "2026-11-17T16:40:00.5Z"), azApproval(shaA, "2026-11-17T16:35:02.993Z"))},
			wantReason: ReasonSuperseded,
			wantIn:     shaB,
		},
		{
			name:       "no PAT — verified, but nothing is completed",
			fake:       &fakeAZ{list: `[{"pullRequestId":1457}]`, show: active(shaA), threads: azThreadsJSON(azApproval(shaA, "2026-11-17T16:35:02.993Z"))},
			gate:       func(g *Gate) { g.PAT = "" },
			wantReason: ReasonNoCredential,
			wantIn:     "$AZURE_DEVOPS_PAT",
		},
		{
			name: "head moved between the check and the completion",
			fake: &fakeAZ{list: `[{"pullRequestId":1457}]`,
				show:    []string{azShow("active", "succeeded", shaA), azShow("active", "succeeded", shaB)},
				threads: azThreadsJSON(azApproval(shaA, "2026-11-17T16:35:02.993Z"))},
			wantReason: ReasonMergeRefused,
			wantIn:     "head moved to " + shaB,
		},
		{
			name: "completion refused by Azure — the stale-pin 409 a live server returned",
			fake: &fakeAZ{list: `[{"pullRequestId":1457}]`,
				show:     active(shaA),
				threads:  azThreadsJSON(azApproval(shaA, "2026-11-17T16:35:02.993Z")),
				httpCode: 409, httpBody: azStaleBody},
			wantReason: ReasonMergeRefused,
			wantIn:     "HTTP 409: TF401192: The source branch has been modified since the last merge attempt.",
			wantPatch:  true,
		},
		{
			name: "completion accepted but never confirmed",
			fake: &fakeAZ{list: `[{"pullRequestId":1457}]`,
				show:    []string{azShow("active", "succeeded", shaA), azShow("active", "succeeded", shaA), azShow("active", "queued", shaA)},
				threads: azThreadsJSON(azApproval(shaA, "2026-11-17T16:35:02.993Z"))},
			wantReason: ReasonMergeUnconfirmed,
			wantIn:     "I am not reporting a promotion I have not seen",
			wantPatch:  true,
		},
		{
			name: "completion rejected by policy",
			fake: &fakeAZ{list: `[{"pullRequestId":1457}]`,
				show:    []string{azShow("active", "succeeded", shaA), azShow("active", "succeeded", shaA), azShow("active", "rejectedByPolicy", shaA)},
				threads: azThreadsJSON(azApproval(shaA, "2026-11-17T16:35:02.993Z"))},
			wantReason: ReasonMergeRefused,
			wantIn:     "rejectedByPolicy",
			wantPatch:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := newAzGate(tc.fake)
			if tc.gate != nil {
				tc.gate(&g)
			}
			res := g.Verify()
			if res.Verdict != Hold {
				t.Fatalf("verdict = %q, want hold: %s", res.Verdict, res.Message)
			}
			if res.Reason != tc.wantReason {
				t.Errorf("reason = %q, want %q (message: %s)", res.Reason, tc.wantReason, res.Message)
			}
			if !strings.Contains(res.Message, tc.wantIn) {
				t.Errorf("message does not carry %q:\n%s", tc.wantIn, res.Message)
			}
			if patched := len(tc.fake.httpCalls) > 0; patched != tc.wantPatch {
				t.Errorf("completion sent = %v, want %v", patched, tc.wantPatch)
			}
			if strings.Contains(res.Message, "test-pat") {
				t.Error("the PAT leaked into the message")
			}
		})
	}
}

func TestAzureOrgURL(t *testing.T) {
	for in, want := range map[string]string{
		"acme":                          "https://dev.azure.com/acme",
		"https://dev.azure.com/acme/":   "https://dev.azure.com/acme",
		"https://acme.visualstudio.com": "https://acme.visualstudio.com",
	} {
		if got := (Gate{Org: in}).orgURL(); got != want {
			t.Errorf("orgURL(%q) = %q, want %q", in, got, want)
		}
	}
}
