package sandbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"time"
)

// DockerSandbox runs commands in an ephemeral, network-isolated container with
// the working tree mounted read-only. Implements the ai.Sandbox interface.
type DockerSandbox struct {
	image   string
	cpus    string
	memory  string
	timeout time.Duration
	log     *slog.Logger
}

func NewDockerSandbox(image, cpus, memory string, timeout time.Duration, log *slog.Logger) *DockerSandbox {
	return &DockerSandbox{image: image, cpus: cpus, memory: memory, timeout: timeout, log: log}
}

// buildDockerArgs assembles the `docker run` argv. Kept pure for testing.
//
// The /tmp tmpfs must allow exec: Docker's default tmpfs mount options are
// noexec, which breaks the common case of `go test`/`go build` compiling a
// binary into a temp dir and then running it ("permission denied"). It's
// capped at the same size as --memory, since tmpfs content is backed by the
// container's memory cgroup anyway — a larger cap wouldn't grant more room.
//
// HOME=/tmp redirects the cache/config directories most build tools default
// to (go build cache, npm, pip, ...) away from the read-only root filesystem,
// which would otherwise fail on the first write (e.g. "mkdir /root/.cache:
// read-only file system").
func buildDockerArgs(image, cpus, memory, workdir, command string) []string {
	return []string{
		"run", "--rm",
		"--network=none",
		"--read-only", "--tmpfs", "/tmp:rw,exec,nosuid,size=" + memory,
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--pids-limit=512",
		"--cpus", cpus, "--memory", memory,
		"-e", "HOME=/tmp",
		"-v", workdir + ":/src:ro", "-w", "/src",
		image, "sh", "-c", command,
	}
}

// Run executes command in the sandbox. exitCode is the command's exit status;
// a non-zero exit is NOT a Go error (the AI should see the output). err is
// returned only for sandbox/infra failures (docker missing, timeout).
func (s *DockerSandbox) Run(workdir, command string) (string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// G204: the sandbox deliberately runs AI-provided commands; isolation is
	// enforced by the container (--network=none, --read-only, no host env).
	//nolint:gosec // intentional: sandboxed execution of untrusted commands
	cmd := exec.CommandContext(ctx, "docker", buildDockerArgs(s.image, s.cpus, s.memory, workdir, command)...)
	cmd.Env = []string{} // no host env / secrets leak into the container

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), -1, fmt.Errorf("sandbox: command timed out after %s", s.timeout)
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return string(out), ee.ExitCode(), nil // command failed — surface output, not a Go error
	}
	if err != nil {
		return string(out), -1, fmt.Errorf("sandbox: running docker: %w", err)
	}
	return string(out), 0, nil
}
