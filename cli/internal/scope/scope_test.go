package scope

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	cases := map[string]Scope{
		"":        Project,
		"project": Project,
		"global":  Global,
		"GLOBAL":  Global,
		" both ":  Both,
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", in, err)
		}
		if got != want {
			t.Errorf("Parse(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := Parse("nonsense"); err == nil {
		t.Error("Parse(nonsense) should error")
	}
}

func TestResolveOverrideWins(t *testing.T) {
	cases := []struct {
		declared Scope
		override Override
		want     Scope
	}{
		{Project, NoOverride, Project},
		{Global, NoOverride, Global},
		{Both, NoOverride, Both},
		{Project, ForceGlobal, Global},  // override beats project default
		{Global, ForceProject, Project}, // override beats global default
		{Both, ForceProject, Project},
	}
	for _, c := range cases {
		if got := Resolve(c.declared, c.override); got != c.want {
			t.Errorf("Resolve(%v, %v) = %v, want %v", c.declared, c.override, got, c.want)
		}
	}
}

func TestEffectiveProjectShadowsGlobal(t *testing.T) {
	// present at both → project shadows global
	if s, ok := Effective(true, true); !ok || s != Project {
		t.Errorf("both present: got %v ok=%v, want project/true", s, ok)
	}
	// global only
	if s, ok := Effective(true, false); !ok || s != Global {
		t.Errorf("global only: got %v ok=%v, want global/true", s, ok)
	}
	// project only
	if s, ok := Effective(false, true); !ok || s != Project {
		t.Errorf("project only: got %v ok=%v, want project/true", s, ok)
	}
	// neither
	if _, ok := Effective(false, false); ok {
		t.Error("neither present: ok should be false")
	}
}

func TestLayerDir(t *testing.T) {
	proj, err := LayerDir(Project, "/work/myapp")
	if err != nil {
		t.Fatal(err)
	}
	if proj != filepath.Join("/work/myapp", ".atl") {
		t.Errorf("project layer: got %q", proj)
	}

	home, _ := os.UserHomeDir()
	glob, err := LayerDir(Global, "/work/myapp")
	if err != nil {
		t.Fatal(err)
	}
	if glob != filepath.Join(home, ".atl") {
		t.Errorf("global layer: got %q", glob)
	}

	if _, err := LayerDir(Both, "/work/myapp"); err == nil {
		t.Error("Both should have no single layer dir")
	}
}
