package dispatch

import (
	"os"
	"os/exec"
	"sync"
	"time"
)

// workerWaitDelay bounds how long the reaper's cmd.Wait() waits for the worker's
// stderr pipe to close AFTER the process has exited. os/exec's Wait blocks on the
// stderr copier goroutine, which only returns once EVERY holder of the inherited
// pipe closes it — a surviving grandchild (e.g. an Azure MCP server that outlives
// a SIGKILL'd worker) would otherwise hang Wait() forever and deadlock the
// single-threaded supervisor. WaitDelay force-closes the pipe and lets Wait return.
const workerWaitDelay = 10 * time.Second

// Handle is a spawned worker process the supervisor can wait on and signal. It
// is the observable surface of a running `claude -p` worker; the scheduler polls
// the worker's status.json for progress and uses this handle for the recovery
// ladder (SIGTERM → SIGKILL) and for post-mortem diagnostics.
type Handle interface {
	Wait() error                // blocks until the process exits
	Exited() (bool, int)        // non-blocking: (has-exited, exit-code); code is -1 while still running — the scheduler poll seam
	Signal(sig os.Signal) error // deliver a signal (SIGTERM, then os.Kill)
	ExitCode() int              // exit code after exit; -1 if not yet exited
	StderrTail() string         // last stderr, for a mark-blocked diagnostic
	PID() int
}

// Spawner starts a worker from a WorkerSpec and returns a Handle. It is the seam
// (mirrors internal/publish's Runner) that keeps the scheduler testable: real
// dispatch spawns a `claude -p` process via NewSpawner(); tests inject a fake
// that fakes exit code + stderr without spawning anything.
type Spawner func(spec WorkerSpec) (Handle, error)

// WorkerSpec describes one `claude -p` worker invocation.
type WorkerSpec struct {
	Prompt        string   // the headless task prompt (the delivery-team builds this)
	WorktreeDir   string   // the worker's cwd — its isolated git worktree
	MCPConfigPath string   // optional --mcp-config path; empty inherits the worktree's .mcp.json (#17)
	ExtraEnv      []string // extra KEY=VALUE appended to the inherited env (e.g. the Azure PAT var)
}

// BuildWorkerArgv builds the `claude -p` argv for a worker. The flags match the
// e2e-proven headless invocation (`claude -p <prompt> --dangerously-skip-permissions
// --output-format json`); --mcp-config is appended when a path is given so the
// worker inherits the parent's Azure MCP surface (wit_*/repo_*/wiki_*) with zero
// new adapter code (#17). The Azure PAT never appears here — it rides the env
// (WorkerSpec.ExtraEnv), never the argv, so it is never logged.
func BuildWorkerArgv(spec WorkerSpec) []string {
	argv := []string{"claude", "-p", spec.Prompt, "--dangerously-skip-permissions", "--output-format", "json"}
	if spec.MCPConfigPath != "" {
		argv = append(argv, "--mcp-config", spec.MCPConfigPath)
	}
	return argv
}

// NewSpawner returns a Spawner backed by the real `claude` binary.
func NewSpawner() Spawner { return execSpawner }

func execSpawner(spec WorkerSpec) (Handle, error) {
	argv := BuildWorkerArgv(spec)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = spec.WorktreeDir
	cmd.Env = append(os.Environ(), spec.ExtraEnv...)
	tail := &tailWriter{}
	cmd.Stderr = tail
	cmd.WaitDelay = workerWaitDelay // bound Wait() so a lingering grandchild can't hang the supervisor
	configureProcAttr(cmd)          // own process group → the recovery ladder can kill the whole tree
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return newReapedHandle(cmd, tail), nil
}

// newReapedHandle wraps a started command in a Handle with a background reaper so
// Exited() is a non-blocking poll: os/exec only populates ProcessState after Wait
// returns, so a single reaper waits once and every Exited()/Wait()/ExitCode()
// reads the reaped result. Closing done happens-after the ProcessState + waitErr
// writes, so the poll reads are race-free.
func newReapedHandle(cmd *exec.Cmd, tail *tailWriter) *execHandle {
	h := &execHandle{cmd: cmd, stderr: tail, done: make(chan struct{})}
	go func() {
		h.waitErr = cmd.Wait()
		close(h.done)
	}()
	return h
}

type execHandle struct {
	cmd     *exec.Cmd
	stderr  *tailWriter
	done    chan struct{}
	waitErr error
}

func (h *execHandle) Wait() error {
	<-h.done
	return h.waitErr
}

// Exited is the scheduler's non-blocking liveness poll: it never blocks on the
// process, returning (false, -1) while the worker still runs and (true, code)
// once the reaper has reaped it.
func (h *execHandle) Exited() (bool, int) {
	select {
	case <-h.done:
		return true, h.cmd.ProcessState.ExitCode()
	default:
		return false, -1
	}
}

func (h *execHandle) Signal(sig os.Signal) error { return signalProcess(h.cmd.Process, sig) }
func (h *execHandle) StderrTail() string         { return h.stderr.String() }
func (h *execHandle) PID() int                   { return h.cmd.Process.Pid }

func (h *execHandle) ExitCode() int {
	select {
	case <-h.done:
		return h.cmd.ProcessState.ExitCode()
	default:
		return -1
	}
}

// stderrTailMax caps the retained stderr so a chatty worker can't grow the
// buffer without bound; only the tail matters for a diagnostic.
const stderrTailMax = 4096

type tailWriter struct {
	mu  sync.Mutex
	buf []byte
}

func (w *tailWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	if len(w.buf) > stderrTailMax {
		w.buf = w.buf[len(w.buf)-stderrTailMax:]
	}
	return len(p), nil
}

func (w *tailWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return string(w.buf)
}
