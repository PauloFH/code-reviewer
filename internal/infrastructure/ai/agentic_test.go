package ai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// fakeWorkspace is an in-memory CodeWorkspace for tests.
type fakeWorkspace struct {
	files map[string]string
}

func (f *fakeWorkspace) Root() string { return "/fake" }
func (f *fakeWorkspace) ReadFile(path string) ([]byte, error) {
	if c, ok := f.files[path]; ok {
		return []byte(c), nil
	}
	return nil, &notFoundErr{path}
}
func (f *fakeWorkspace) ListDir(_ string) ([]domain.DirEntry, error) {
	return []domain.DirEntry{{Name: "main.go"}, {Name: "internal", IsDir: true}}, nil
}
func (f *fakeWorkspace) Grep(pattern, _ string) ([]domain.GrepHit, error) {
	var hits []domain.GrepHit
	for p, c := range f.files {
		for i, line := range strings.Split(c, "\n") {
			if strings.Contains(line, pattern) {
				hits = append(hits, domain.GrepHit{Path: p, Line: i + 1, Text: line})
			}
		}
	}
	return hits, nil
}

type notFoundErr struct{ p string }

func (e *notFoundErr) Error() string { return "not found: " + e.p }

func TestToolExecutorReadFile(t *testing.T) {
	exec := newToolExecutor(&fakeWorkspace{files: map[string]string{"main.go": "package main\n"}}, nil)
	out := exec.run("read_file", map[string]any{"path": "main.go"})
	if !strings.Contains(out, "package main") {
		t.Fatalf("read_file output = %q", out)
	}
}

func TestToolExecutorUnknownTool(t *testing.T) {
	exec := newToolExecutor(&fakeWorkspace{}, nil)
	out := exec.run("frobnicate", map[string]any{})
	if !strings.Contains(strings.ToLower(out), "unknown tool") {
		t.Fatalf("expected unknown-tool message, got %q", out)
	}
}

func TestToolExecutorReadFileErrorIsReported(t *testing.T) {
	exec := newToolExecutor(&fakeWorkspace{files: map[string]string{}}, nil)
	out := exec.run("read_file", map[string]any{"path": "missing.go"})
	if !strings.Contains(strings.ToLower(out), "error") && !strings.Contains(out, "not found") {
		t.Fatalf("expected error text in output, got %q", out)
	}
}

type fakeSandbox struct {
	out    string
	code   int
	err    error
	called bool
	gotCmd string
}

func (s *fakeSandbox) Run(_ string, command string) (string, int, error) {
	s.called = true
	s.gotCmd = command
	return s.out, s.code, s.err
}

func TestToolExecutorRun(t *testing.T) {
	sb := &fakeSandbox{out: "ok\n", code: 0}
	exec := newToolExecutor(&fakeWorkspace{}, sb)
	out := exec.run("run", map[string]any{"command": "go test ./..."})
	if !sb.called || sb.gotCmd != "go test ./..." {
		t.Fatalf("sandbox not invoked correctly: %+v", sb)
	}
	if !strings.Contains(out, "exit=0") || !strings.Contains(out, "ok") {
		t.Fatalf("run output = %q", out)
	}
}

func TestToolExecutorRunWithoutSandbox(t *testing.T) {
	exec := newToolExecutor(&fakeWorkspace{}, nil)
	out := exec.run("run", map[string]any{"command": "go test"})
	if !strings.Contains(strings.ToLower(out), "habilitada") && !strings.Contains(strings.ToLower(out), "error") {
		t.Fatalf("expected disabled message, got %q", out)
	}
}

func TestToolSpecsFor(t *testing.T) {
	names := func(specs []toolSpec) map[string]bool {
		m := map[string]bool{}
		for _, s := range specs {
			m[s.Name] = true
		}
		return m
	}
	if names(toolSpecsFor(false))["run"] {
		t.Error("run tool must NOT be present when execution disabled")
	}
	withRun := names(toolSpecsFor(true))
	if !withRun["run"] || !withRun["read_file"] {
		t.Error("execution=true must include run + read tools")
	}
}

func TestRunAgenticLoopExecutesRunTool(t *testing.T) {
	driver := &scriptedDriver{steps: []agenticStep{
		{Calls: []toolCall{{ID: "r", Name: "run", Input: map[string]any{"command": "go test ./..."}}}},
		{Final: `{"review_markdown":"testes ok","inline_comments":[]}`},
	}}
	sb := &fakeSandbox{out: "PASS\n", code: 0}
	res, err := runAgenticLoop(driver, newToolExecutor(&fakeWorkspace{}, sb), 12)
	if err != nil {
		t.Fatalf("loop: %v", err)
	}
	if !sb.called {
		t.Fatal("expected sandbox to be invoked via run tool")
	}
	if res.Body != "testes ok" {
		t.Fatalf("body = %q", res.Body)
	}
}

func TestToolSpecsCoverReadTools(t *testing.T) {
	names := map[string]bool{}
	for _, s := range readToolSpecs() {
		names[s.Name] = true
	}
	for _, want := range []string{"read_file", "list_dir", "grep"} {
		if !names[want] {
			t.Errorf("missing tool spec %q", want)
		}
	}
}

// scriptedDriver returns a predefined sequence of steps, recording the tool
// results it receives so tests can assert the loop fed them back.
type scriptedDriver struct {
	steps    []agenticStep
	idx      int
	gotInput [][]toolResult
}

func (d *scriptedDriver) step(results []toolResult) (agenticStep, error) {
	d.gotInput = append(d.gotInput, results)
	st := d.steps[d.idx]
	d.idx++
	return st, nil
}

func TestRunAgenticLoopExecutesToolsThenFinal(t *testing.T) {
	driver := &scriptedDriver{steps: []agenticStep{
		{Calls: []toolCall{{ID: "a", Name: "read_file", Input: map[string]any{"path": "main.go"}}}, Usage: domain.TokenUsage{TotalTokens: 10}},
		{Final: `{"review_markdown":"tudo certo","inline_comments":[]}`, Usage: domain.TokenUsage{TotalTokens: 5}},
	}}
	exec := newToolExecutor(&fakeWorkspace{files: map[string]string{"main.go": "package main\n"}}, nil)

	res, err := runAgenticLoop(driver, exec, 12)
	if err != nil {
		t.Fatalf("runAgenticLoop: %v", err)
	}
	if res.Body != "tudo certo" {
		t.Fatalf("body = %q", res.Body)
	}
	if res.Usage.TotalTokens != 15 {
		t.Fatalf("usage accumulation = %d, want 15", res.Usage.TotalTokens)
	}
	// second step must have received the tool result from step 1
	if len(driver.gotInput) != 2 || len(driver.gotInput[1]) != 1 || driver.gotInput[1][0].ID != "a" {
		t.Fatalf("loop did not feed tool results back: %+v", driver.gotInput)
	}
}

func TestRunAgenticLoopExceedsIterations(t *testing.T) {
	// always returns a tool call, never finalizes
	loopStep := agenticStep{Calls: []toolCall{{ID: "x", Name: "list_dir", Input: map[string]any{"path": "."}}}}
	driver := &scriptedDriver{steps: []agenticStep{loopStep, loopStep, loopStep}}
	exec := newToolExecutor(&fakeWorkspace{}, nil)

	if _, err := runAgenticLoop(driver, exec, 2); err == nil {
		t.Fatal("expected error when loop exceeds max iterations")
	}
}

func TestReviewDiffSingleShotWhenNoWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"review_markdown\":\"single shot\",\"inline_comments\":[]}"}}],"usage":{"total_tokens":7}}`))
	}))
	defer srv.Close()
	c := &Client{transportKind: providerChatCompletions, mode: domain.AIModeAPI, baseURL: srv.URL, token: "k", model: "gpt-4o", http: &http.Client{Timeout: 10 * time.Second}}
	res, err := c.ReviewDiff(domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "t"},
		Diff:         "--- a/x\n+++ b/x\n@@ -1 +1 @@\n+x",
		Profile:      &domain.ReviewProfile{Agentic: true}, // agentic, mas sem Workspace → single-shot
	})
	if err != nil {
		t.Fatalf("expected single-shot, got error: %v", err)
	}
	if res.Body != "single shot" {
		t.Fatalf("body = %q", res.Body)
	}
}
