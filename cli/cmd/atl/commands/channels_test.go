package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/queue"
)

// installTeam writes an install manifest for a team at one layer root.
func installTeam(t *testing.T, layer string, m *manifest.Manifest) {
	t.Helper()
	if err := m.Write(layer); err != nil {
		t.Fatal(err)
	}
}

// dryTranscript writes a session transcript with no capture markers at all and
// enough turns + user input to cross both watchdog thresholds.
func dryTranscript(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "sess.jsonl")
	body := tickLine("user", strings.Repeat("a", 700)) +
		tickLine("assistant", "dry reply one") +
		tickLine("user", strings.Repeat("b", 700)) +
		tickLine("assistant", "dry reply two")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func openTestQueue(t *testing.T, dir string) *queue.Store {
	t.Helper()
	st, err := queue.Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// TestNoDeclarationNoSignal is the regression for the ungated watchdog: before
// channels came from declarations, `atl tick` on a machine with NO profile-team
// installed still told the agent to "spawn ONE background /profile-drain
// subagent" — naming a skill and a rule that machine does not have.
//
// A machine whose installed teams declare no channel must be measured for core's
// channel and nothing else, and no output may name any drain skill but /drain.
func TestNoDeclarationNoSignal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := t.TempDir()

	// A team IS installed — it simply owns no capture channel. This is the common
	// case, and it must be indistinguishable from "no teams at all".
	installTeam(t, filepath.Join(home, ".atl"), &manifest.Manifest{
		Handle: "acme", Name: "plain-team", Stores: []string{"~/.atl/notes"},
	})

	chans := declaredChannels(project)
	if len(chans) != 1 || chans[0].Name != string(queue.ChannelLearning) {
		t.Fatalf("declaredChannels = %+v, want exactly core's own channel", chans)
	}

	// The signal side: the auto-drain notice can only be produced for a channel in
	// the set, so a pending count on an undeclared channel reaches no sentence.
	counts := map[queue.Channel]int{queue.ChannelLearning: 3, queue.Channel("profile-fact"): 7}
	var signals []string
	for _, ch := range chans {
		if msg := autoDrainNotice(ch, counts[queue.Channel(ch.Name)]); msg != "" {
			signals = append(signals, msg)
		}
	}
	if len(signals) != 1 || !strings.Contains(signals[0], "3 learning(s)") {
		t.Fatalf("auto-drain signals = %q, want only the learning one", signals)
	}

	// The watchdog side: a fully dry session must nudge for learnings only.
	dir := t.TempDir()
	path := dryTranscript(t, dir)
	st := openTestQueue(t, dir)
	out := captureStdout(t, func() { captureWatchdogNotice(st, project, path, chans) })

	if !strings.Contains(out, "capture-watchdog (learning)") {
		t.Errorf("core's own channel must still be watched: %q", out)
	}
	for _, forbidden := range []string{"/profile-drain", "profile-capture", "profile-fact"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("output named %q on a machine with no team declaring it — the ungated-watchdog bug:\n%s", forbidden, out)
		}
	}
}

// The mirror of the test above: a declaration activates the channel, and every
// word of its signals comes from the declaration.
//
// The fixture deliberately uses values that appear NOWHERE in the source — not
// profile-fact/profile-drain/profile-capture — so a leftover hardcoded constant
// fails here instead of passing by coincidence.
func TestDeclarationActivatesTheChannelVerbatim(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := t.TempDir()

	declared := manifest.Channel{
		Name: "audit-note", Drain: "/audit-drain",
		Rule: "audit-capture", Describes: "audit notes",
	}
	installTeam(t, filepath.Join(home, ".atl"), &manifest.Manifest{
		Handle: "acme", Name: "audit-team", Channels: []manifest.Channel{declared},
	})

	chans := declaredChannels(project)
	if len(chans) != 2 || chans[0] != coreChannel || chans[1] != declared {
		t.Fatalf("declaredChannels = %+v, want [core, declared] in that order", chans)
	}

	// The auto-drain sentence takes its noun from Name and its rule from Rule.
	msg := autoDrainNotice(declared, 4)
	want := "atl: 4 audit-note(s) pending — auto-drain them now in a background subagent (per the audit-capture rule)"
	if msg != want {
		t.Errorf("auto-drain signal:\n got %q\nwant %q", msg, want)
	}

	// The watchdog sentence takes Describes, Drain and Rule from the same record.
	dir := t.TempDir()
	path := dryTranscript(t, dir)
	st := openTestQueue(t, dir)
	out := captureStdout(t, func() { captureWatchdogNotice(st, project, path, chans) })

	if !strings.Contains(out, "capture-watchdog (audit-note)") {
		t.Errorf("declared channel not watched: %q", out)
	}
	for _, want := range []string{"missed audit notes", "background /audit-drain subagent", "per the audit-capture rule"} {
		if !strings.Contains(out, want) {
			t.Errorf("watchdog sentence missing %q — a hardcoded value survived:\n%s", want, out)
		}
	}
}

// The read is the half that fails CLOSED: a channel this function does not find
// is silently inert, indistinguishable from a team that declares none. So assert
// where it reads, not only what it concludes.
//
// A globally-installed team's channel must be found from inside any project —
// the case that matters, since profile-team is global-only — and a
// project-installed team's channel must be found too.
func TestDeclaredChannelsReadsBothLayers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := t.TempDir()

	installTeam(t, filepath.Join(home, ".atl"), &manifest.Manifest{
		Handle: "acme", Name: "global-team",
		Channels: []manifest.Channel{{Name: "g-chan", Drain: "/g", Rule: "g-rule", Describes: "g things"}},
	})
	installTeam(t, filepath.Join(project, ".atl"), &manifest.Manifest{
		Handle: "acme", Name: "project-team",
		Channels: []manifest.Channel{{Name: "p-chan", Drain: "/p", Rule: "p-rule", Describes: "p things"}},
	})

	got := channelNames(declaredChannels(project))
	want := []string{"learning", "g-chan", "p-chan"} // core, then global, then project
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("declaredChannels = %v, want %v", got, want)
	}
}

// With no project root resolved, the project layer must be skipped entirely:
// LayerDir would otherwise build a RELATIVE ".atl" and read whatever happens to
// sit under the current working directory — activating a channel from a
// directory that is not this project.
func TestDeclaredChannelsSkipsAnUnresolvedProject(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cwd := t.TempDir()
	installTeam(t, filepath.Join(cwd, ".atl"), &manifest.Manifest{
		Handle: "acme", Name: "stray",
		Channels: []manifest.Channel{{Name: "stray-chan", Drain: "/s", Rule: "s-rule", Describes: "s things"}},
	})

	restore, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(restore)

	got := declaredChannels("")
	if len(got) != 1 || got[0] != coreChannel {
		t.Fatalf("declaredChannels(\"\") = %+v, want only core's channel — it read a relative project layer", got)
	}
}

// An install written before schema v3 carries no channels, exactly like a team
// that declares none: its markers are refused until `atl update` backfills the
// field. A bounded, self-resolving window — but it must be silent, not broken.
func TestAPreV3InstallContributesNoChannel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := t.TempDir()

	installTeam(t, filepath.Join(home, ".atl"), &manifest.Manifest{
		SchemaVersion: 2, Handle: "acme", Name: "profile-team", Stores: []string{"~/.atl/profiles"},
	})

	chans := declaredChannels(project)
	if len(chans) != 1 || chans[0] != coreChannel {
		t.Fatalf("declaredChannels = %+v, want only core's channel", chans)
	}
	// And no doctor warning: nothing was declared, so nothing was refused.
	if r := channelsCheck(project)(); r.Status != doctor.OK {
		t.Errorf("a pre-v3 install must not warn (it declared nothing): %+v", r)
	}
}

// A declaration the platform had to refuse is a SILENT loss otherwise — markers
// on that channel are never captured and nothing says why. Both refusal reasons
// must reach the doctor, and neither may make `atl doctor` exit non-zero.
func TestChannelsCheckReportsRefusedDeclarations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := t.TempDir()
	layer := filepath.Join(home, ".atl")

	installTeam(t, layer, &manifest.Manifest{
		Handle: "acme", Name: "broken-team",
		Channels: []manifest.Channel{{Name: "half-declared", Drain: "/half"}}, // no rule, no describes
	})
	installTeam(t, layer, &manifest.Manifest{
		Handle: "acme", Name: "collider-team",
		Channels: []manifest.Channel{{Name: "learning", Drain: "/x", Rule: "y", Describes: "z"}},
	})

	if got := channelNames(declaredChannels(project)); len(got) != 1 || got[0] != "learning" {
		t.Fatalf("refused declarations must not become active: %v", got)
	}

	r := channelsCheck(project)()
	if r.Status != doctor.Warn {
		t.Fatalf("status = %v, want Warn (a third-party declaration must never fail the doctor)", r.Status)
	}
	for _, want := range []string{"broken-team", `"half-declared"`, `"rule"`, `"describes"`, "collider-team", "core owns it"} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("detail missing %q — the refusal is unexplained:\n%s", want, r.Detail)
		}
	}
}

// Two teams claiming the same channel: first wins, and the loser is reported
// rather than silently shadowed.
func TestChannelsCheckReportsATeamOnTeamCollision(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := t.TempDir()
	layer := filepath.Join(home, ".atl")

	ch := manifest.Channel{Name: "shared", Drain: "/a", Rule: "a-rule", Describes: "a things"}
	installTeam(t, layer, &manifest.Manifest{Handle: "acme", Name: "a-team", Channels: []manifest.Channel{ch}})
	ch2 := ch
	ch2.Drain = "/b"
	installTeam(t, layer, &manifest.Manifest{Handle: "acme", Name: "b-team", Channels: []manifest.Channel{ch2}})

	chans := declaredChannels(project)
	if len(chans) != 2 || chans[1].Drain != "/a" {
		t.Fatalf("first declaration must win: %+v", chans)
	}
	r := channelsCheck(project)()
	if r.Status != doctor.Warn || !strings.Contains(r.Detail, "b-team") ||
		!strings.Contains(r.Detail, "another installed team already claimed it") {
		t.Errorf("a team-on-team collision must be reported: %+v", r)
	}
}

// isDeclaredChannel is the gate on every surface where a channel name arrives as
// free-form input (`peek --channel`, `_enqueue`). A typo must fail loudly rather
// than open a phantom channel whose items no drain will ever claim.
func TestIsDeclaredChannel(t *testing.T) {
	chans := []manifest.Channel{coreChannel, {Name: "audit-note"}}
	for _, ok := range []string{"learning", "audit-note"} {
		if !isDeclaredChannel(chans, ok) {
			t.Errorf("isDeclaredChannel(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"learnings", "profile-fact", "", "LEARNING"} {
		if isDeclaredChannel(chans, bad) {
			t.Errorf("isDeclaredChannel(%q) = true — a phantom channel is exactly what this gate exists to refuse", bad)
		}
	}
}

// TestEnqueueSeamRefusesAnUndeclaredChannel guards the second of the queue's two
// write seams. `atl learnings _enqueue` takes a channel name as free-form input
// (the /drain skill's mining step calls it), and before the declaration contract
// it performed no validation at all: `_enqueue profile-fct "…"` succeeded on any
// machine and left a pending item in the project's bucket that no drain would
// ever claim — the marker LOOKS captured while nothing can process it.
//
// The sibling seam (drain.Drain) has its own refusal test. Both are asserted
// because a future caller reintroducing the hole at either one compiles clean
// and passes every other gate.
func TestEnqueueSeamRefusesAnUndeclaredChannel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // windows
	t.Chdir(t.TempDir())          // the project key openQueue derives from cwd

	// No team is installed, so only core's own channel is active.
	err := learningsEnqueueCmd.RunE(learningsEnqueueCmd, []string{"profile-fact", "entity: alex"})
	if err == nil {
		t.Fatal("_enqueue accepted an undeclared channel — a phantom item nothing will ever drain")
	}
	for _, want := range []string{"profile-fact", "learning"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should name the rejected channel and the active set, missing %q: %v", want, err)
		}
	}

	// The active channel still goes through — the gate rejects the undeclared
	// name, it does not break capture.
	if err := learningsEnqueueCmd.RunE(learningsEnqueueCmd, []string{"learning", "pools are shared"}); err != nil {
		t.Fatalf("core's own channel must still enqueue: %v", err)
	}

	// Assert on the durable end-state, not on the two exit codes: the refusal has
	// to happen BEFORE the write, and only the queue can say whether it did. The
	// command holds an exclusive lock, so this reads after both calls returned.
	counts := pendingCounts(t, home)
	if _, phantom := counts[queue.Channel("profile-fact")]; phantom {
		t.Errorf("the refused channel still reached the queue: %v", counts)
	}
	if counts[queue.ChannelLearning] != 1 {
		t.Errorf("the active channel's item did not land: %v", counts)
	}
}

// The peek gate is a READ, so it must refuse a typo without refusing data. Items
// already queued on a channel that is no longer active — the owning team was
// uninstalled, or its declaration has not been backfilled onto the install
// manifest yet — are real: `learnings status` lists them precisely so a count
// cannot vanish, and a filter that then refuses to show them would strand the one
// case where there is something to drain.
func TestPeekReadsAnInactiveChannelThatHasItems(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // windows
	project := t.TempDir()
	t.Chdir(project)

	// No team is installed, so "orphan-chan" is inactive — but it has an item,
	// written straight to the store the way one would have been left behind.
	if err := os.MkdirAll(filepath.Join(home, ".atl"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := queue.Open(filepath.Join(home, ".atl", "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	ch := queue.Channel("orphan-chan")
	if _, err := st.Enqueue(project, queue.Item{
		ID: queue.NewID(ch, "stranded"), Channel: ch, Payload: "stranded",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	// Cobra flags are shared package state, so set it explicitly and put it back.
	setPeekChannel(t, "orphan-chan")
	out := captureStdout(t, func() {
		if err := learningsPeekCmd.RunE(learningsPeekCmd, nil); err != nil {
			t.Fatalf("peek --channel on a channel that HAS items must read them: %v", err)
		}
	})
	if !strings.Contains(out, "stranded") {
		t.Errorf("the stranded item was not listed:\n%s", out)
	}

	// A typo matches nothing, so it still fails loudly rather than answering with
	// an empty list that reads as "nothing pending".
	setPeekChannel(t, "learnin")
	if err := learningsPeekCmd.RunE(learningsPeekCmd, nil); err == nil {
		t.Error("peek accepted a typo'd channel with no items — it would read as 'nothing pending'")
	}
}

// setPeekChannel sets peek's --channel for one test and restores it afterwards;
// a cobra command's flags are package-level state shared across tests.
func setPeekChannel(t *testing.T, value string) {
	t.Helper()
	if err := learningsPeekCmd.Flags().Set("channel", value); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = learningsPeekCmd.Flags().Set("channel", "") })
}

// A layer the pass could not read is the third outcome: not "nothing declared
// here" but "cannot tell". Reporting it is what stops the check from certifying
// every channel active while a whole layer went unexamined.
func TestChannelsCheckReportsAnUnreadableLayer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := t.TempDir()

	dir := filepath.Join(home, ".atl", "installed")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "acme__broken.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := channelsCheck(project)()
	if r.Status != doctor.Warn || !strings.Contains(r.Detail, "could not read the installed teams") {
		t.Fatalf("an unreadable layer must be reported, not certified: %+v", r)
	}
}

// pendingCounts reads the per-channel pending counts straight from the queue the
// commands under test wrote to (HOME-derived, like queue.DefaultPath).
func pendingCounts(t *testing.T, home string) map[queue.Channel]int {
	t.Helper()
	st, err := queue.Open(filepath.Join(home, ".atl", "queue.db"))
	if err != nil {
		t.Fatalf("open queue: %v", err)
	}
	defer func() { _ = st.Close() }()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	counts, err := st.Counts(cwd)
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	return counts
}
