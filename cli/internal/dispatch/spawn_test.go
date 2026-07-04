package dispatch

import (
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

func argvContains(argv []string, want string) bool {
	for _, a := range argv {
		if a == want {
			return true
		}
	}
	return false
}

func TestBuildWorkerArgv(t *testing.T) {
	spec := WorkerSpec{
		Prompt:        "implement work-item 4821",
		WorktreeDir:   "/wt/s1/4821",
		MCPConfigPath: "/cfg/.mcp.json",
		ExtraEnv:      []string{"AZURE_DEVOPS_PAT=super-secret-token"},
	}
	argv := BuildWorkerArgv(spec)

	if argv[0] != "claude" || !argvContains(argv, "-p") {
		t.Fatalf("argv should start `claude -p`: %v", argv)
	}
	if !argvContains(argv, "implement work-item 4821") {
		t.Error("prompt missing from argv")
	}
	for _, flag := range []string{"--dangerously-skip-permissions", "--output-format", "json"} {
		if !argvContains(argv, flag) {
			t.Errorf("headless flag %q missing (must match the e2e-proven invocation)", flag)
		}
	}
	if !argvContains(argv, "--mcp-config") || !argvContains(argv, "/cfg/.mcp.json") {
		t.Error("--mcp-config path should be appended when set (#17)")
	}
	// The Azure PAT must ride the env, never the argv — so it is never logged.
	if strings.Contains(strings.Join(argv, " "), "super-secret-token") {
		t.Error("secret leaked into argv — the PAT must stay in ExtraEnv only")
	}
}

func TestBuildWorkerArgvNoMCPConfig(t *testing.T) {
	argv := BuildWorkerArgv(WorkerSpec{Prompt: "x"})
	if argvContains(argv, "--mcp-config") {
		t.Error("--mcp-config should be omitted when no path is given (worktree .mcp.json is inherited)")
	}
}

func TestTailWriterCapsAndKeepsTail(t *testing.T) {
	w := &tailWriter{}
	// Write more than the cap in chunks; only the tail should survive.
	big := strings.Repeat("A", stderrTailMax)
	if _, err := w.Write([]byte(big)); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("TAIL")); err != nil {
		t.Fatal(err)
	}
	got := w.String()
	if len(got) != stderrTailMax {
		t.Errorf("tail len = %d, want capped at %d", len(got), stderrTailMax)
	}
	if !strings.HasSuffix(got, "TAIL") {
		t.Error("tail buffer should keep the most recent bytes")
	}
}

func TestNewSpawnerNonNil(t *testing.T) {
	if NewSpawner() == nil {
		t.Error("NewSpawner should return a real Spawner")
	}
}

// startReaped runs a benign shell command through the real reaper so the
// non-blocking Exited() poll is proven against an actual process (execSpawner
// hardcodes `claude`, so it can't be unit-tested directly).
func startReaped(t *testing.T, script string) *execHandle {
	t.Helper()
	cmd := exec.Command("sh", "-c", script)
	tail := &tailWriter{}
	cmd.Stderr = tail
	if err := cmd.Start(); err != nil {
		t.Skipf("sh unavailable: %v", err)
	}
	return newReapedHandle(cmd, tail)
}

func TestExecHandleExitedAfterReap(t *testing.T) {
	h := startReaped(t, "exit 7")
	if err := h.Wait(); err == nil {
		t.Error("a non-zero exit should surface a Wait error")
	}
	exited, code := h.Exited()
	if !exited || code != 7 {
		t.Errorf("Exited() = (%v, %d) after reap, want (true, 7)", exited, code)
	}
	if h.ExitCode() != 7 {
		t.Errorf("ExitCode() = %d, want 7", h.ExitCode())
	}
}

func TestExecHandleExitedIsNonBlockingWhileRunning(t *testing.T) {
	h := startReaped(t, "sleep 3")
	// The poll must return immediately with (false, -1) — never block on Wait.
	if exited, code := h.Exited(); exited || code != -1 {
		t.Errorf("a still-running worker: Exited() = (%v, %d), want (false, -1)", exited, code)
	}
	if h.ExitCode() != -1 {
		t.Errorf("ExitCode() while running = %d, want -1", h.ExitCode())
	}
	_ = h.Signal(syscall.SIGKILL) // don't leak the process
	_ = h.Wait()
}
