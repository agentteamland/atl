// Package scope is the global/project layering primitive — v2's first-class
// scope axis.
//
// There are two layers, isomorphic with Claude Code's own (~/.atl user-global
// vs <project>/.atl). A team's publisher declares a default scope; the user
// may override it at install time. A capability may live at both layers, and
// when it does, the project layer shadows global (nearest wins) — the same
// mental model as Claude Code's CLAUDE.md layering.
package scope

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Scope is a declared default or a chosen install layer.
type Scope int

const (
	Project Scope = iota // project-local (the default)
	Global               // user-global
	Both                 // present at / installable to both layers
)

func (s Scope) String() string {
	switch s {
	case Global:
		return "global"
	case Both:
		return "both"
	default:
		return "project"
	}
}

// Parse reads a publisher's declared scope string. Empty means project.
func Parse(s string) (Scope, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "project":
		return Project, nil
	case "global":
		return Global, nil
	case "both":
		return Both, nil
	default:
		return Project, fmt.Errorf("invalid scope %q (want global|project|both)", s)
	}
}

// Override is the user's install-time choice.
type Override int

const (
	NoOverride   Override = iota // honor the publisher default
	ForceGlobal                  // --global
	ForceProject                 // --project
)

func (o Override) String() string {
	switch o {
	case ForceGlobal:
		return "global"
	case ForceProject:
		return "project"
	default:
		return "none"
	}
}

// Resolve returns the effective install scope from the publisher's declared
// default and the user's override. The user override always wins; otherwise
// the declared default holds.
func Resolve(declared Scope, override Override) Scope {
	switch override {
	case ForceGlobal:
		return Global
	case ForceProject:
		return Project
	default:
		return declared
	}
}

// Effective resolves which layer a capability is used from when it may be
// present at either layer. Project shadows global (nearest wins). The bool is
// false when the capability is present at neither layer.
func Effective(presentGlobal, presentProject bool) (Scope, bool) {
	switch {
	case presentProject:
		return Project, true
	case presentGlobal:
		return Global, true
	default:
		return Project, false
	}
}

// LayerDir returns the on-disk root for a single-layer scope. Global lives
// under ~/.atl; project lives under <projectRoot>/.atl. Both has no single
// directory and is an error.
func LayerDir(s Scope, projectRoot string) (string, error) {
	switch s {
	case Global:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".atl"), nil
	case Project:
		return filepath.Join(projectRoot, ".atl"), nil
	default:
		return "", fmt.Errorf("scope %q has no single layer directory", s)
	}
}
