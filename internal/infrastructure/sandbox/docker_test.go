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
		"--tmpfs /tmp", "--cpus 2", "--memory 2g",
		"/work/src:/src:ro", "-w /src", "golang:1.25", "go test ./...",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("docker args missing %q\nargs: %s", want, joined)
		}
	}
	// command must be passed via sh -c as a single arg
	if args[len(args)-1] != "go test ./..." {
		t.Errorf("last arg should be the command, got %q", args[len(args)-1])
	}
}
