package teampkg

import (
	"testing"
)

// The declared shape: a field inside a named capability object, like `store` and
// `channel` — not a bare-string capability the way `review` is.
func TestDeclaredSessionScripts(t *testing.T) {
	tm := withCapabilities(t, `{"capabilities":{
		"briefing": {"sessionScript": "scripts/session-brief.sh"}
	}}`)
	got := tm.DeclaredSessionScripts()
	if len(got) != 1 || got[0] != "scripts/session-brief.sh" {
		t.Errorf("DeclaredSessionScripts() = %v, want [scripts/session-brief.sh]", got)
	}
}

// Sorted capability order, so a manifest with several declarations produces the
// same list every read — the same determinism DeclaredStores/DeclaredChannels give.
func TestDeclaredSessionScriptsIsInSortedCapabilityOrder(t *testing.T) {
	tm := withCapabilities(t, `{"capabilities":{
		"zeta":  {"sessionScript": "scripts/z.sh"},
		"alpha": {"sessionScript": "scripts/a.sh"}
	}}`)
	got := tm.DeclaredSessionScripts()
	if len(got) != 2 || got[0] != "scripts/a.sh" || got[1] != "scripts/z.sh" {
		t.Errorf("DeclaredSessionScripts() = %v, want [scripts/a.sh scripts/z.sh]", got)
	}
}

// Declaring none is the common case: silence, never an error.
func TestDeclaredSessionScriptsEmptyWhenNoneDeclared(t *testing.T) {
	for name, raw := range map[string]string{
		"no capabilities at all":     `{}`,
		"empty capabilities":         `{"capabilities":{}}`,
		"a different capability":     `{"capabilities":{"profile":{"store":"~/.atl/profiles"}}}`,
		"a bare-string capability":   `{"capabilities":{"review":"tech-lead"}}`,
		"the field is not a string":  `{"capabilities":{"delivery":{"sessionScript":42}}}`,
		"the capability is an array": `{"capabilities":{"delivery":["scripts/x.sh"]}}`,
	} {
		if got := withCapabilities(t, raw).DeclaredSessionScripts(); len(got) != 0 {
			t.Errorf("%s: DeclaredSessionScripts() = %v, want none", name, got)
		}
	}
}

// The vetting happens HERE, where the value enters the manifest, rather than at
// exec time — a value that escapes the team's assets must never be RECORDED as a
// legitimate declaration, or it reads later as a broken script rather than a
// refused one.
func TestSessionScriptRelRefusesWhatCannotBeATeamAsset(t *testing.T) {
	for name, decl := range map[string]string{
		"empty":                  "",
		"blank":                  "   ",
		"absolute":               "/etc/payload.sh",
		"parent escape":          "../../../etc/payload.sh",
		"escape after a segment": "scripts/../../../etc/payload.sh",
		"bare filename":          "brief.sh",
		"current dir":            ".",
	} {
		if got, ok := SessionScriptRel(decl); ok {
			t.Errorf("%s: SessionScriptRel(%q) = %q, true — want refused", name, decl, got)
		}
	}
}

// A refused declaration must be DROPPED from the recorded list, not recorded and
// filtered later: the manifest is what a consumer sees, and an escaping path in
// it is a claim the platform never intends to honour.
func TestDeclaredSessionScriptsDropsARefusedPath(t *testing.T) {
	tm := withCapabilities(t, `{"capabilities":{
		"bad":  {"sessionScript": "../../../etc/payload.sh"},
		"good": {"sessionScript": "scripts/ok.sh"}
	}}`)
	got := tm.DeclaredSessionScripts()
	if len(got) != 1 || got[0] != "scripts/ok.sh" {
		t.Errorf("DeclaredSessionScripts() = %v, want only [scripts/ok.sh]", got)
	}
}

func TestSessionScriptRelNormalises(t *testing.T) {
	for decl, want := range map[string]string{
		"scripts/brief.sh":         "scripts/brief.sh",
		"  scripts/brief.sh  ":     "scripts/brief.sh",
		"./scripts/brief.sh":       "scripts/brief.sh",
		"scripts/./brief.sh":       "scripts/brief.sh",
		"scripts/sub/../brief.sh":  "scripts/brief.sh",
		"knowledge/hooks/brief.sh": "knowledge/hooks/brief.sh",
	} {
		got, ok := SessionScriptRel(decl)
		if !ok || got != want {
			t.Errorf("SessionScriptRel(%q) = %q, %v; want %q, true", decl, got, ok, want)
		}
	}
}

