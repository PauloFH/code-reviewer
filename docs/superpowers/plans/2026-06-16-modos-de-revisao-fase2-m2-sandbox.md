# Modos de Revisão — Fase 2 / Milestone 2: Sandbox de execução (Docker) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Habilitar a camada `execution`: dar ao loop agêntico uma tool `run` que executa comandos (testes/build/lint) num sandbox Docker isolado (sem rede, sem segredos, código read-only), ativada apenas quando `profile.Execution` e um sandbox está configurado.

**Architecture:** Interface `Sandbox` (definida no pacote `ai`, ao lado de `transport`) implementada pelo novo pacote `internal/infrastructure/sandbox` (Docker). O `toolExecutor` ganha um `sandbox` opcional e a tool `run`; `toolSpecsFor(execution)` inclui `run` quando ligado. O `ai.Client` recebe o sandbox via `ResolvingReviewer`, e só expõe `run` quando `profile.Execution`. Sem sandbox configurado (`SANDBOX_IMAGE` vazio), nada muda.

**Tech Stack:** Go (stdlib `os/exec`, `context`). Testes com fake sandbox + função pura de montagem de args (Docker não é executado nos testes).

**Pré-requisitos:** M4b (loop + executor + `agenticDriver`), M4c (`profile.Execution` já existe na entidade).

**Decisão de design aprovada (brainstorming):** Docker com `--rm --network=none --read-only --tmpfs /tmp --cpus --memory -v <ws>:/src:ro`. O mount read-only do código é proposital (código não-confiável); builds que escrevem na árvore têm essa limitação conhecida — refinável depois. Interface `Sandbox` permite Podman/gVisor no futuro.

**Referências (ler antes):**
- `internal/infrastructure/ai/agentic.go` — `toolSpec`, `toolExecutor`, `newToolExecutor`, `readToolSpecs`, `runAgenticLoop`, `agenticStep`/`scriptedDriver` (nos testes).
- `internal/infrastructure/ai/client.go` — `Client`, `tryAgentic`, `agenticDriver`, `newClientFromConfig`, `New`.
- `internal/infrastructure/ai/resolver.go` — `ResolvingReviewer`, `NewResolvingReviewer`, `ReviewDiff`, `TestConfig`.
- `internal/config/config.go` — `AgenticConfig`/`Load`.
- `cmd/server/main.go:137-143` — construção do `envClient` e `NewResolvingReviewer`.

---

## File Structure

- **Modify:** `internal/infrastructure/ai/agentic.go` — `Sandbox` interface, `run` tool, `toolSpecsFor`, executor com sandbox.
- **Modify:** `internal/infrastructure/ai/agentic_test.go` — atualizar call sites de `newToolExecutor`; testes de `run`/`toolSpecsFor`.
- **Modify:** `internal/infrastructure/ai/{anthropic,chat_completions,gemini}_agentic_test.go` — atualizar `newToolExecutor(ws)` → `newToolExecutor(ws, nil)`.
- **Modify:** `internal/infrastructure/ai/client.go` — campo `sandbox`, ativar `run` quando `profile.Execution`.
- **Modify:** `internal/infrastructure/ai/resolver.go` — campo `sandbox` + param; setar no client.
- **Modify:** `internal/infrastructure/ai/resolver_test.go` — atualizar call site de `NewResolvingReviewer`.
- **Create:** `internal/infrastructure/sandbox/docker.go` — `DockerSandbox` + `buildDockerArgs`.
- **Create:** `internal/infrastructure/sandbox/docker_test.go` — teste de `buildDockerArgs`.
- **Modify:** `internal/config/config.go` — `SandboxConfig`.
- **Modify:** `cmd/server/main.go` — construir o sandbox e injetar.

---

## Task 1: ai — Sandbox interface, tool `run`, executor

**Files:**
- Modify: `internal/infrastructure/ai/agentic.go`
- Modify: `internal/infrastructure/ai/agentic_test.go`
- Modify: `internal/infrastructure/ai/anthropic_agentic_test.go`, `chat_completions_agentic_test.go`, `gemini_agentic_test.go`

- [ ] **Step 1: Atualizar call sites + escrever testes**

(a) Em TODOS os arquivos de teste do pacote `ai` que chamam `newToolExecutor(<ws>)` com UM argumento, trocar para `newToolExecutor(<ws>, nil)`. (Use grep: `grep -rn "newToolExecutor(" internal/infrastructure/ai/`.)

(b) Em `internal/infrastructure/ai/agentic_test.go`, adicionar:

```go
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
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/ai/ -run 'TestToolExecutorRun|TestToolSpecsFor|TestRunAgenticLoopExecutesRun' -v`
Expected: FAIL — `Sandbox`/`toolSpecsFor`/segundo arg de `newToolExecutor` indefinidos.

- [ ] **Step 3: Implementar**

Em `internal/infrastructure/ai/agentic.go`:

(a) Adicionar a interface `Sandbox` (perto do topo, após os consts):

```go
// Sandbox runs a shell command against the checked-out working tree in isolation.
// Implemented by internal/infrastructure/sandbox. exitCode is the command's exit
// status; err is non-nil only for sandbox/infra failures (not command failure).
type Sandbox interface {
	Run(workdir, command string) (output string, exitCode int, err error)
}
```

(b) Adicionar `runToolSpec` e `toolSpecsFor`:

```go
func runToolSpec() toolSpec {
	return toolSpec{
		Name:        "run",
		Description: "Executa um comando shell no repositório dentro de um sandbox isolado (sem rede, sem segredos, código somente-leitura). Use para rodar testes, build ou linters.",
		Schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"command": map[string]any{"type": "string", "description": "Comando shell, ex.: 'go test ./...'"}},
			"required":   []string{"command"},
		},
	}
}

// toolSpecsFor returns the tools available to the agentic reviewer; the run tool
// is included only when sandboxed execution is enabled.
func toolSpecsFor(execution bool) []toolSpec {
	specs := readToolSpecs()
	if execution {
		specs = append(specs, runToolSpec())
	}
	return specs
}
```

(c) Dar ao `toolExecutor` um campo `sandbox` e atualizar `newToolExecutor`:

```go
type toolExecutor struct {
	ws      domain.CodeWorkspace
	sandbox Sandbox
}

func newToolExecutor(ws domain.CodeWorkspace, sandbox Sandbox) *toolExecutor {
	return &toolExecutor{ws: ws, sandbox: sandbox}
}
```

(d) Adicionar o case `"run"` no `switch` de `(*toolExecutor).run`, antes do `default`:

```go
	case "run":
		if e.sandbox == nil {
			return "error: execução em sandbox não habilitada"
		}
		out, code, err := e.sandbox.Run(e.ws.Root(), asString(input["command"]))
		if err != nil {
			return "error: " + err.Error()
		}
		return truncate(fmt.Sprintf("exit=%d\n%s", code, out))
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/ai/ -v`
Expected: PASS (todos — os call sites atualizados + os novos).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/ai/agentic.go internal/infrastructure/ai/agentic_test.go internal/infrastructure/ai/anthropic_agentic_test.go internal/infrastructure/ai/chat_completions_agentic_test.go internal/infrastructure/ai/gemini_agentic_test.go
git commit -m "feat(ai): tool run + Sandbox interface + toolSpecsFor(execution)"
```

---

## Task 2: ai — ativar `run` no Client quando profile.Execution

**Files:**
- Modify: `internal/infrastructure/ai/client.go`
- Modify: `internal/infrastructure/ai/resolver.go`
- Modify: `internal/infrastructure/ai/resolver_test.go`

- [ ] **Step 1: Implementar (sem teste novo — coberto pela Task 1 + build)**

(a) Em `internal/infrastructure/ai/client.go`, adicionar o campo `sandbox Sandbox` ao struct `Client` (após `promptRepo`):

```go
	promptRepo    domain.PromptRepository
	sandbox       Sandbox
```

(b) Atualizar `tryAgentic` para escolher as tools e o executor conforme `profile.Execution`:

```go
func (c *Client) tryAgentic(input domain.ReviewInput, sysPrompt, userPrompt string) (*domain.ReviewResult, error, bool) {
	execution := input.Profile != nil && input.Profile.Execution && c.sandbox != nil
	driver, err := c.agenticDriver(sysPrompt, userPrompt, toolSpecsFor(execution))
	if err != nil {
		return nil, nil, false
	}
	var sb Sandbox
	if execution {
		sb = c.sandbox
	}
	maxIter := 0
	if input.Profile != nil {
		maxIter = input.Profile.MaxIterations
	}
	result, runErr := runAgenticLoop(driver, newToolExecutor(input.Workspace, sb), maxIter)
	return result, runErr, true
}
```

(c) Em `internal/infrastructure/ai/resolver.go`, adicionar o campo `sandbox Sandbox` ao `ResolvingReviewer` e o parâmetro ao `NewResolvingReviewer` (no fim, após `envPriceOut`/antes de `log` — escolha uma posição e seja consistente). Atribua no construtor.

No `ReviewDiff`, após construir o client (nos dois caminhos: env e cfg), setar o sandbox antes de chamar `ReviewDiff` do client:

- caminho env (`cfg == nil`): antes de `r.envClient.ReviewDiff(input)`, fazer `r.envClient.sandbox = r.sandbox`.
- caminho cfg: após `client := newClientFromConfig(*cfg, r.promptRepo)`, fazer `client.sandbox = r.sandbox`.

No `TestConfig`, após `client := newClientFromConfig(...)`, também `client.sandbox = r.sandbox` (inócuo; o diff de teste não é agêntico).

(d) Em `internal/infrastructure/ai/resolver_test.go`, atualizar a(s) chamada(s) de `NewResolvingReviewer` adicionando `nil` na posição do novo parâmetro `sandbox`.

- [ ] **Step 2: Build + testes**

Run: `go test ./internal/infrastructure/ai/...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/ai/client.go internal/infrastructure/ai/resolver.go internal/infrastructure/ai/resolver_test.go
git commit -m "feat(ai): Client/ResolvingReviewer propagam sandbox; run só quando profile.Execution"
```

---

## Task 3: Pacote sandbox — DockerSandbox

**Files:**
- Create: `internal/infrastructure/sandbox/docker.go`
- Create: `internal/infrastructure/sandbox/docker_test.go`

- [ ] **Step 1: Escrever o teste que falha**

Criar `internal/infrastructure/sandbox/docker_test.go`:

```go
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
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/sandbox/ -v`
Expected: FAIL — pacote/func inexistente.

- [ ] **Step 3: Implementar**

Criar `internal/infrastructure/sandbox/docker.go`:

```go
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
func buildDockerArgs(image, cpus, memory, workdir, command string) []string {
	return []string{
		"run", "--rm",
		"--network=none",
		"--read-only", "--tmpfs", "/tmp",
		"--cpus", cpus, "--memory", memory,
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
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/sandbox/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/sandbox/docker.go internal/infrastructure/sandbox/docker_test.go
git commit -m "feat(sandbox): DockerSandbox isolado (network=none, read-only, sem env)"
```

---

## Task 4: Config — SandboxConfig

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Implementar**

(a) Adicionar o campo em `Config` (após `Agentic AgenticConfig`):

```go
	Agentic           AgenticConfig
	Sandbox           SandboxConfig
```

(b) Adicionar a struct:

```go
// SandboxConfig tunes the execution sandbox. Image empty = sandbox disabled.
type SandboxConfig struct {
	Image          string
	CPUs           string
	Memory         string
	TimeoutSeconds int
}
```

(c) No `Load`, adicionar ao literal `cfg`:

```go
		Sandbox: SandboxConfig{
			Image:          getEnv("SANDBOX_IMAGE", ""),
			CPUs:           getEnv("SANDBOX_CPUS", "2"),
			Memory:         getEnv("SANDBOX_MEMORY", "2g"),
			TimeoutSeconds: getEnvInt("SANDBOX_TIMEOUT_SECONDS", 120),
		},
```

- [ ] **Step 2: Build + commit**

Run: `go build ./internal/config/ && go test ./internal/config/...`

```bash
git add internal/config/config.go
git commit -m "feat(config): SandboxConfig (SANDBOX_IMAGE/CPUS/MEMORY/TIMEOUT)"
```

---

## Task 5: Wiring no main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Construir e injetar**

Em `cmd/server/main.go`:

(a) Adicionar o import:

```go
	"github.com/paulofh/code-reviewer/internal/infrastructure/sandbox"
```

(b) Antes de `reviewer := ai.NewResolvingReviewer(...)`, construir o sandbox (nil quando desabilitado):

```go
	var execSandbox ai.Sandbox
	if cfg.Sandbox.Image != "" {
		execSandbox = sandbox.NewDockerSandbox(
			cfg.Sandbox.Image, cfg.Sandbox.CPUs, cfg.Sandbox.Memory,
			time.Duration(cfg.Sandbox.TimeoutSeconds)*time.Second, log,
		)
		log.Info("execution sandbox enabled", slog.String("image", cfg.Sandbox.Image))
	}
```

(c) Passar `execSandbox` como o novo parâmetro de `NewResolvingReviewer` (na posição definida na Task 2c):

```go
	reviewer := ai.NewResolvingReviewer(
		repo, repo, claudeOAuth, envClient,
		cfg.AI.PricePerMInputTokens, cfg.AI.PricePerMOutputTokens,
		execSandbox, log,
	)
```

- [ ] **Step 2: Build + testes**

Run: `go build ./... && go test ./...`
Expected: sucesso, tudo verde.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(main): injeta DockerSandbox no ResolvingReviewer (SANDBOX_IMAGE)"
```

---

## Verificação final do milestone

- [ ] **Build:** `go build ./...` → sucesso.
- [ ] **Testes:** `go test ./...` → tudo verde, sem regressão.
- [ ] **Sanidade:** com `SANDBOX_IMAGE` setado e um perfil com `execution`, o loop agêntico ganha a tool `run` que executa comandos no container isolado; sem `SANDBOX_IMAGE` (default), `run` não é exposta e nada muda. `buildDockerArgs` garante `--network=none`, `--read-only`, `:ro`, sem env.

## Notas para o executor

- **Docker não é executado nos testes** — só `buildDockerArgs` (pura) e o executor com `fakeSandbox`. Isso cobre a lógica sem exigir daemon/imagem.
- **Exit ≠ erro:** comando que falha (exit≠0) NÃO é erro Go — devolva o output + exitCode para a IA ver. Erro Go só para falha de infra (docker ausente, timeout).
- **`newToolExecutor` agora tem 2 args** — atualize TODOS os call sites de teste (`grep -rn "newToolExecutor(" internal/infrastructure/ai/`) para passar `nil` onde não há sandbox.
- **Sem env/segredos no container:** `cmd.Env = []string{}` + `--network=none`. Não remova esses dois — são o núcleo do isolamento.
- **Limitação conhecida:** `-v <ws>:/src:ro` é read-only; builds que escrevem na árvore do projeto falharão. É o trade-off de segurança aprovado; refinar (ex.: cópia gravável) é trabalho futuro.
- **Posição do parâmetro `sandbox`** em `NewResolvingReviewer`: escolha (sugiro antes de `log`) e propague a TODOS os call sites (main.go + resolver_test.go).
