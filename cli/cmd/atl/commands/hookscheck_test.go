package commands

import (
	"testing"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/settings"
)

// fullInstallWithThrottle seeds a complete, valid atl hook install using the given
// tick throttle, then returns nothing — HOME must already be set to a temp dir.
func fullInstallWithThrottle(t *testing.T, throttle string, withRetrieve bool) {
	t.Helper()
	hooks := []settings.Hook{
		{Event: "SessionStart", Command: "atl session-start"},
		{Event: "UserPromptSubmit", Command: "atl tick --throttle=" + throttle},
	}
	if withRetrieve {
		hooks = append(hooks, settings.Hook{Event: "UserPromptSubmit", Command: "atl retrieve"})
	}
	hooks = append(hooks, settings.Hook{Event: "PreToolUse", Matcher: "Bash|Edit|Write", Command: "atl guard"})
	if _, err := settings.InstallHooks(hooks); err != nil {
		t.Fatal(err)
	}
}

// TestHooksCheckPreservesCustomThrottle: a fully-installed setup that uses a
// non-default `atl tick --throttle=5m` must NOT be flagged as missing or rewritten
// back to 10m — otherwise the doctor resets the user's throttle every session.
func TestHooksCheckPreservesCustomThrottle(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	fullInstallWithThrottle(t, "5m", true)

	res := hooksCheck()()
	if res.Status != doctor.OK {
		t.Errorf("want OK, got %v — %s", res.Status, res.Detail)
	}
	if res.Healed {
		t.Errorf("a fully-installed setup should not self-heal; got %q", res.Detail)
	}
	if got := currentTickThrottle(); got != "5m" {
		t.Errorf("the user's throttle was not preserved: got %q, want 5m", got)
	}
}

// TestHooksCheckHealMissingKeepsThrottle: when another hook is genuinely missing
// (here: retrieve), the self-heal must re-bind it WITHOUT resetting the user's
// custom tick throttle.
func TestHooksCheckHealMissingKeepsThrottle(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	fullInstallWithThrottle(t, "5m", false) // no retrieve hook -> a genuine heal target

	res := hooksCheck()()
	if !res.Healed {
		t.Errorf("a missing retrieve hook should heal; got status %v — %s", res.Status, res.Detail)
	}
	if got := currentTickThrottle(); got != "5m" {
		t.Errorf("healing another hook reset the tick throttle: got %q, want 5m", got)
	}
	// the healed retrieve hook is now present
	if missing, _ := missingHooks([]settings.Hook{{Event: "UserPromptSubmit", Command: "atl retrieve"}}); len(missing) != 0 {
		t.Errorf("retrieve hook was not bound by the heal")
	}
}
