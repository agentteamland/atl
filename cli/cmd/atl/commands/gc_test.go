package commands

import "testing"

// gcFlagConflict must reject selecting more than one mutually exclusive gc mode
// — reclaim (--apply, with its optional --include-gains modifier), --undo, and
// --purge — instead of letting the RunE switch silently precedence-order them
// (the dangerous case being `--purge --apply`, which runs the irreversible purge
// and drops --apply). A single mode, or --apply paired with its --include-gains
// modifier, is not a conflict.
func TestGCFlagConflict(t *testing.T) {
	tests := []struct {
		name                             string
		apply, undo, purge, includeGains bool
		wantErr                          bool
	}{
		// no flags: the default dry run — never a conflict.
		{"no flags (dry run)", false, false, false, false, false},
		// single modes: each is fine on its own.
		{"apply only", true, false, false, false, false},
		{"undo only", false, true, false, false, false},
		{"purge only", false, false, true, false, false},
		{"include-gains only", false, false, false, true, false},
		// --include-gains is a modifier of the reclaim path, not a separate mode.
		{"apply + include-gains", true, false, false, true, false},
		// the three conflicting pairs the ticket calls out.
		{"undo + purge", false, true, true, false, true},
		{"apply + undo", true, true, false, false, true},
		{"purge + apply", true, false, true, false, true},
		// --include-gains combined with a non-reclaim mode: contradictory intent.
		{"include-gains + undo", false, true, false, true, true},
		{"include-gains + purge", false, false, true, true, true},
		// all three modes at once.
		{"apply + undo + purge", true, true, true, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gcFlagConflict(tt.apply, tt.undo, tt.purge, tt.includeGains)
			if tt.wantErr && err == nil {
				t.Errorf("gcFlagConflict(apply=%v, undo=%v, purge=%v, includeGains=%v) = nil, want error",
					tt.apply, tt.undo, tt.purge, tt.includeGains)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("gcFlagConflict(apply=%v, undo=%v, purge=%v, includeGains=%v) = %v, want nil",
					tt.apply, tt.undo, tt.purge, tt.includeGains, err)
			}
		})
	}
}
