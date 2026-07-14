package selfupdate

import (
	"context"
	"testing"
)

// AutoApply's short-circuits must not touch the network (or the throttle stamp) —
// a dev build, an empty version, or the env brake all return "" immediately.
func TestAutoApplyShortCircuits(t *testing.T) {
	ctx := context.Background()

	if got := AutoApply(ctx, devVersion); got != "" {
		t.Errorf("dev build should no-op, got %q", got)
	}
	if got := AutoApply(ctx, ""); got != "" {
		t.Errorf("empty version should no-op, got %q", got)
	}

	t.Setenv(EnvDisable, "1")
	if got := AutoApply(ctx, "2.3.1"); got != "" {
		t.Errorf("%s should no-op, got %q", EnvDisable, got)
	}
}
