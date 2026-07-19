package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/settings"
)

// hooksCheck is the doctor check that verifies ATL's automation hooks are bound
// in ~/.claude/settings.json and re-binds them if any went missing. Automation is
// mandatory (decision D-3), so an unbound hook silently kills the whole loop
// (drain, doctor, guard stop firing) — the decision's own named blind spot. This
// is a genuinely safe deterministic self-heal: InstallHooks is idempotent and
// preserves the user's own hooks, so re-binding never loses anything.
func hooksCheck() doctor.Check {
	return func() doctor.Result {
		want := defaultHooks()
		// Preserve a user-customized `atl tick --throttle=…`. defaultHooks hardcodes
		// the 10m default, but setup-hooks lets the user pick any interval; since this
		// self-heal runs every session, re-binding the default would silently rewrite a
		// customized throttle back to 10m (and print a bogus "re-bound" message) forever.
		// Re-bind the tick hook with the user's own interval instead.
		if thr := currentTickThrottle(); thr != "" {
			for i := range want {
				if strings.HasPrefix(want[i].Command, tickThrottlePrefix) {
					want[i].Command = tickThrottlePrefix + thr
				}
			}
		}
		missing, err := missingHooks(want)
		if err != nil {
			return doctor.Result{Name: "hooks-bound", Status: doctor.Warn,
				Detail: "could not read settings.json: " + err.Error()}
		}
		if len(missing) == 0 {
			return doctor.Result{Name: "hooks-bound", Status: doctor.OK, Detail: "all automation hooks bound"}
		}
		if _, herr := settings.InstallHooks(want); herr != nil {
			return doctor.Result{Name: "hooks-bound", Status: doctor.Warn,
				Detail: fmt.Sprintf("%d automation hook(s) unbound; re-bind failed: %v", len(missing), herr)}
		}
		return doctor.Result{Name: "hooks-bound", Status: doctor.OK, Healed: true,
			Detail: fmt.Sprintf("re-bound %d dropped automation hook(s): %s", len(missing), strings.Join(missing, ", "))}
	}
}

// missingHooks returns the events whose atl command is not present in settings.json.
// A missing/empty settings file means every hook is missing (a fresh heal target).
func missingHooks(want []settings.Hook) ([]string, error) {
	path, err := settings.Path()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return eventNames(want), nil
		}
		return nil, err
	}
	var obj struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		// Unparseable settings — don't guess; report all wanted as missing so the
		// idempotent InstallHooks re-asserts them (it tolerates a corrupt file).
		return eventNames(want), nil
	}
	var missing []string
	for _, h := range want {
		if !hasCommand(obj.Hooks[h.Event], h.Command) {
			missing = append(missing, h.Event)
		}
	}
	return missing, nil
}

func hasCommand(groups []struct {
	Hooks []struct {
		Command string `json:"command"`
	} `json:"hooks"`
}, command string) bool {
	for _, g := range groups {
		for _, h := range g.Hooks {
			if h.Command == command {
				return true
			}
		}
	}
	return false
}

func eventNames(hooks []settings.Hook) []string {
	names := make([]string, 0, len(hooks))
	for _, h := range hooks {
		names = append(names, h.Event)
	}
	return names
}

// tickThrottlePrefix is the stable prefix of the UserPromptSubmit tick hook; the
// suffix is the user-chosen throttle interval.
const tickThrottlePrefix = "atl tick --throttle="

// currentTickThrottle returns the throttle the user's installed `atl tick`
// UserPromptSubmit hook uses (e.g. "5m"), or "" if there is no such hook or the
// settings file is absent/unreadable. It lets the doctor re-bind the tick hook
// with the user's own interval instead of the hardcoded default.
func currentTickThrottle() string {
	path, err := settings.Path()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var obj struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return ""
	}
	for _, g := range obj.Hooks["UserPromptSubmit"] {
		for _, h := range g.Hooks {
			if strings.HasPrefix(h.Command, tickThrottlePrefix) {
				return strings.TrimPrefix(h.Command, tickThrottlePrefix)
			}
		}
	}
	return ""
}
