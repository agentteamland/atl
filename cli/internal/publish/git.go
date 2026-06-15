package publish

import (
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes an external command and returns its combined output. It's the
// seam that keeps apply testable: real apply shells out via execRunner, while
// tests inject a fake to assert the exact git/gh argv without touching the
// network or mutating any repo.
type Runner func(name string, args ...string) ([]byte, error)

func execRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// GH wraps the git + gh commands publish's apply paths need. NewGH() uses the
// real binaries; tests set Run to a fake Runner.
type GH struct {
	Run Runner
}

// NewGH returns a GH backed by the real git/gh binaries.
func NewGH() GH { return GH{Run: execRunner} }

func (g GH) out(name string, args ...string) (string, error) {
	b, err := g.Run(name, args...)
	if err != nil {
		return "", fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(b)))
	}
	return strings.TrimSpace(string(b)), nil
}

// Login returns the authenticated GitHub login.
func (g GH) Login() (string, error) { return g.out("gh", "api", "user", "-q", ".login") }

// DefaultBranch returns repo's default branch (e.g. "main").
func (g GH) DefaultBranch(repo string) (string, error) {
	return g.out("gh", "repo", "view", repo, "--json", "defaultBranchRef", "-q", ".defaultBranchRef.name")
}

// Fork forks repo to the authenticated user without cloning. Idempotent: gh
// no-ops if the fork already exists.
func (g GH) Fork(repo string) error {
	_, err := g.out("gh", "repo", "fork", repo, "--clone=false")
	return err
}

// Clone clones repo into dir.
func (g GH) Clone(repo, dir string) error {
	_, err := g.out("gh", "repo", "clone", repo, dir)
	return err
}

// Git runs `git -C dir args...`.
func (g GH) Git(dir string, args ...string) (string, error) {
	return g.out("git", append([]string{"-C", dir}, args...)...)
}

// PRCreate opens a PR on baseRepo (base branch) from head ("login:branch") with
// the given title and a body read from bodyFile. Returns the PR URL gh prints.
// The cross-repo form (--repo upstream + --head login:branch) is what makes the
// PR land on the source repo rather than the fork.
func (g GH) PRCreate(baseRepo, base, head, title, bodyFile string) (string, error) {
	return g.out("gh", "pr", "create",
		"--repo", baseRepo, "--base", base, "--head", head,
		"--title", title, "--body-file", bodyFile)
}

// EnsureTopic adds topic to repo's GitHub topics (idempotent). The atl-team
// topic is what the index CI's topic-discovery scan keys off.
func (g GH) EnsureTopic(repo, topic string) error {
	_, err := g.out("gh", "repo", "edit", repo, "--add-topic", topic)
	return err
}
