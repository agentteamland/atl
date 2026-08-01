package commands

import (
	"fmt"
	"strings"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/queue"
	"github.com/agentteamland/atl/cli/internal/scope"
)

// coreChannel is the platform's OWN capture channel. It is not a declaration:
// core ships the learning-capture rule and the /drain skill, so this channel
// exists on every machine, with or without any team installed.
var coreChannel = manifest.Channel{
	Name:      string(queue.ChannelLearning),
	Drain:     "/drain",
	Rule:      "learning-capture",
	Describes: "learnings",
}

// declaredChannels returns every capture channel active in this project: core's
// own first, then the valid ones installed teams declared, global layer before
// project layer, sorted within a layer by handle/name (manifest.List's order).
//
// Reading a declared value to emit a signal is a MECHANICAL use of the
// declaration; it is not enforcement of who may read or write anything. Access
// control over capabilities is a separate, still-deferred concern (#167) — this
// function neither implements nor begins it.
//
// Never fails: an unreadable layer contributes nothing. A channel whose name is
// already taken (by core, or by an earlier team) is dropped — first wins, core
// always wins — and an invalid declaration is dropped too; channelsCheck is
// what reports both.
func declaredChannels(projectRoot string) []manifest.Channel {
	active, _ := collectChannels(projectRoot)
	return active
}

// collectChannels is declaredChannels plus the reasons a declaration was
// refused, so the doctor check and the signal emitters read the same pass and
// can never disagree about which channels are active.
func collectChannels(projectRoot string) ([]manifest.Channel, []string) {
	active := []manifest.Channel{coreChannel}
	taken := map[string]bool{coreChannel.Name: true}
	var problems []string

	layers := []scope.Scope{scope.Global}
	if projectRoot != "" {
		// An unresolved project root would make LayerDir return a RELATIVE ".atl",
		// silently reading whatever happens to sit under the current directory.
		layers = append(layers, scope.Project)
	}
	for _, s := range layers {
		// A layer this pass could not read is the third outcome — not "no channel
		// declared here" but "cannot tell". Skipping it silently would let the check
		// below report that every declared channel is active while a whole layer went
		// unexamined, so one unparseable install manifest could disable a channel with
		// nothing saying why. Report it instead; the channels stay inactive either way.
		layer, err := scope.LayerDir(s, projectRoot)
		if err != nil {
			problems = append(problems, fmt.Sprintf(
				"could not resolve the %s layer (%v); a capture channel declared there is inactive", s, err))
			continue
		}
		manifests, err := manifest.List(layer)
		if err != nil {
			problems = append(problems, fmt.Sprintf(
				"could not read the installed teams at the %s layer (%v); a capture channel declared there is inactive", s, err))
			continue
		}
		for _, m := range manifests {
			for _, ch := range m.Channels {
				name := strings.TrimSpace(ch.Name)
				if name == "" {
					continue // nothing addressable — not recorded, so not reportable
				}
				if missing := ch.MissingFields(); len(missing) > 0 {
					problems = append(problems, fmt.Sprintf(
						"team %s declares capture channel %q without %s; markers on it are not captured",
						m.Name, name, quoteJoin(missing)))
					continue
				}
				if taken[name] {
					owner := "another installed team already claimed it"
					if name == coreChannel.Name {
						owner = "core owns it"
					}
					problems = append(problems, fmt.Sprintf(
						"team %s declares capture channel %q, but %s; its declaration is ignored",
						m.Name, name, owner))
					continue
				}
				taken[name] = true
				active = append(active, ch)
			}
		}
	}
	return active, problems
}

// channelsCheck reports capture channels an installed team declared that the
// platform had to refuse: a declaration missing a field a signal sentence needs,
// or a name already taken by core or another team. It also reports a layer the
// pass could not read, so it never claims everything is active when it could not
// look. All three are silent losses otherwise — markers on such a channel are
// never captured and nothing says why.
//
// Warn, never Fail: a broken third-party declaration must not make `atl doctor`
// exit non-zero.
func channelsCheck(projectRoot string) doctor.Check {
	return func() doctor.Result {
		_, problems := collectChannels(projectRoot)
		if len(problems) == 0 {
			return doctor.Result{Name: "channels", Status: doctor.OK,
				Detail: "every declared capture channel is active"}
		}
		return doctor.Result{Name: "channels", Status: doctor.Warn,
			Detail: strings.Join(problems, "; ")}
	}
}

// channelNames is the list of active channel names, for a message or a queue
// lookup.
func channelNames(chans []manifest.Channel) []string {
	out := make([]string, 0, len(chans))
	for _, ch := range chans {
		out = append(out, ch.Name)
	}
	return out
}

// isDeclaredChannel reports whether name is one of the active channels — the
// gate on every surface where a channel name arrives as free-form input
// (`peek --channel`, `_enqueue`). A typo must fail loudly here rather than open
// a phantom channel whose items no drain will ever claim.
func isDeclaredChannel(chans []manifest.Channel, name string) bool {
	for _, ch := range chans {
		if ch.Name == name {
			return true
		}
	}
	return false
}

// quoteJoin renders a field list as `"a", "b"` for an error/warning detail.
func quoteJoin(fields []string) string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = fmt.Sprintf("%q", f)
	}
	return strings.Join(out, ", ")
}
