package sandbox

import (
	"strings"
	"testing"
)

func TestBuildDockerArgs(t *testing.T) {
	args := buildDockerArgs("golang:1.25", "2", "2g", "/work/src", "go test ./...")
	joined := strings.Join(args, " ")

	for _, want := range []string{
		"run", "--rm", "--network=none", "--read-only",
		"--tmpfs /tmp:rw,exec,nosuid,size=2g", "--cpus 2", "--memory 2g",
		"--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=512",
		"-e HOME=/tmp",
		"/work/src:/src:ro", "-w /src", "golang:1.25", "go test ./...",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("docker args missing %q\nargs: %s", want, joined)
		}
	}
	// the tmpfs must allow exec — go test/build compiles a binary into /tmp and
	// runs it, which the Docker default (noexec) would reject with "permission
	// denied". See buildDockerArgs' doc comment.
	if strings.Contains(joined, "noexec") {
		t.Error("tmpfs must not be mounted noexec — breaks compiling and running test/build binaries")
	}
	// command must be passed via sh -c as a single arg
	if args[len(args)-1] != "go test ./..." {
		t.Errorf("last arg should be the command, got %q", args[len(args)-1])
	}
}
