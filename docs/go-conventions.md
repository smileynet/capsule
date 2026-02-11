# Go Conventions

Concise reference for anyone writing Go code in this project. Covers style, patterns, and tooling conventions. Does not cover deployment, CI/CD, or API design. Long URLs collected in [References](#references); all others inline.

## Contents

1. [Project Structure](#1-project-structure)
2. [Interface Design](#2-interface-design)
3. [Error Handling](#3-error-handling)
4. [Testing](#4-testing)
5. [Package Naming](#5-package-naming)
6. [Dependency Injection](#6-dependency-injection)
7. [Code Style](#7-code-style)
8. [CLI (Kong)](#8-cli-kong)
9. [YAML Configuration](#9-yaml-configuration)
10. [Subprocess Execution](#10-subprocess-execution)

## 1. Project Structure

Thin `cmd/` entry points. All logic in `internal/` packages. No `pkg/` directory — `internal/` enforces encapsulation at the compiler level.

```
cmd/capsule/main.go    # wiring only: parse flags, create deps, call Run()
internal/config/        # one responsibility per package
internal/provider/
```

- [Standard Go Project Layout discussion](https://go.dev/doc/modules/layout)
- [Ben Johnson: Standard Package Layout](https://www.gobeyond.dev/standard-package-layout/)

## 2. Interface Design

Accept interfaces, return structs. Define interfaces where consumed, not where implemented — this keeps packages decoupled.

```go
// In orchestrator package (consumer):
type Provider interface {
    Complete(ctx context.Context, prompt string) (string, error)
}

// In provider package (producer): export the concrete struct
type Claude struct { /* ... */ }
func (c *Claude) Complete(ctx context.Context, prompt string) (string, error) { /* ... */ }
```

Keep interfaces small. A one- or two-method interface is easier to mock and compose.

- [Go Wiki: CodeReviewComments — Interfaces](https://go.dev/wiki/CodeReviewComments#interfaces)
- [Go Proverbs: The bigger the interface, the weaker the abstraction](https://go-proverbs.github.io/)

## 3. Error Handling

Wrap errors with `%w` to preserve the chain. Use sentinel errors for conditions callers need to check.

```go
var ErrNotFound = errors.New("config: not found")

func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("config: reading %s: %w", path, err)
    }
    // ...
}
```

Never discard errors silently. If a function returns an error, handle it or propagate it.

- [Go Blog: Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- [Effective Go: Errors](https://go.dev/doc/effective_go#errors)

## 4. Testing

Table-driven tests for multiple cases. Mock at package boundaries, not internal functions. Use `t.TempDir()` for filesystem tests — cleanup is automatic.

```go
func TestParseDuration(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    time.Duration
        wantErr bool
    }{
        {name: "seconds", input: "30s", want: 30 * time.Second},
        {name: "invalid", input: "xyz", wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseDuration(tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Speed Tiers

Guard slow tests with `testing.Short()` so pre-commit hooks stay fast:

- **Fast** (always run): Pure logic, no I/O, no sleeps. No guard needed.
- **Slow** (skip in `-short` mode): Subprocess execution, file I/O, `time.Sleep`.
  Use `if testing.Short() { t.Skip("...") }`.
- **Smoke** (build-tag gated): End-to-end binary tests. Use `//go:build smoke`.

- [Go Wiki: TableDrivenTests](https://go.dev/wiki/TableDrivenTests)
- [Go Blog: Using Subtests and Sub-benchmarks](https://go.dev/blog/subtests)

## 5. Package Naming

Short, lowercase, singular nouns. No `utils`, `helpers`, `common`, or `misc` — these attract unrelated code. If you can't name it, the abstraction is wrong.

```
config      ✓       configuration  ✗
prompt      ✓       promptUtils    ✗
worktree    ✓       helpers        ✗
```

- [Go Blog: Package Names](https://go.dev/blog/package-names)
- [Effective Go: Package names](https://go.dev/doc/effective_go#package-names)

## 6. Dependency Injection

Pass dependencies as function/constructor parameters. Use functional options for optional configuration. Avoid package-level globals and `init()`.

```go
type Server struct {
    logger *slog.Logger
    store  Store
}

func NewServer(store Store, opts ...Option) *Server {
    s := &Server{store: store, logger: slog.Default()}
    for _, opt := range opts {
        opt(s)
    }
    return s
}

type Option func(*Server)

func WithLogger(l *slog.Logger) Option {
    return func(s *Server) { s.logger = l }
}
```

- [Dave Cheney: Functional options for friendly APIs][functional-options]
- [Uber Go Style Guide: Dependency Injection](https://github.com/uber-go/guide/blob/master/style.md)

## 7. Code Style

Use `goimports` over `gofmt` — it's a superset that formats code and manages imports (adds missing, removes unused). Enforcement is automatic:

**Post-edit:** Every `.go` file edit triggers `goimports -w`, `go build`, and `go vet` scoped to the edited package. See `scripts/hooks/claude-go-check.sh`.

**Pre-commit:** Incremental `golangci-lint` (only new issues via `--new-from-rev=HEAD`) and `go test -short` run on staged packages via bd hook chaining. The canonical hook lives at `scripts/hooks/pre-commit.sh` and must be installed as `.git/hooks/pre-commit.old` for bd to discover it. See `CLAUDE.md` "Hook Setup" for installation and `.golangci.yml` for linter config.

Add a doc comment to every exported symbol — `go doc` and IDE tooling depend on it.

```go
// Provider executes AI completions against a configured backend.
type Provider struct { /* ... */ }
```

Comments should explain *why*, not *what*. The code already says what.

- [Effective Go: Commentary](https://go.dev/doc/effective_go#commentary)
- [Go Doc Comments](https://go.dev/doc/comment)

## 8. CLI (Kong)

Use [Kong](https://github.com/alecthomas/kong) for CLI parsing. Define commands as nested structs with a `Run() error` method on each leaf command. Keep `main()` thin — parse and dispatch.

### Main Structure

```go
type CLI struct {
    Version kong.VersionFlag `help:"Show version." short:"V"`
    Run     RunCmd           `cmd:"" help:"Run a capsule pipeline."`
}

func main() {
    var cli CLI
    ctx := kong.Parse(&cli, kong.Vars{"version": version + " " + commit + " " + date})
    err := ctx.Run()
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: %s\n", err)
        os.Exit(exitCode(err))
    }
}
```

### Testability: Run() / run() Split

Each command's `Run()` method constructs real dependencies, then delegates to a lowercase `run()` that accepts interfaces. Tests call `run()` directly with mocks, bypassing config loading and provider construction.

```go
// pipelineRunner abstracts orchestrator.RunPipeline for testing.
type pipelineRunner interface {
    RunPipeline(ctx context.Context, input orchestrator.PipelineInput) error
}

// Run constructs real deps (config, provider, orchestrator), then calls run().
func (r *RunCmd) Run() error {
    cfg, err := loadConfig()
    // ... build real provider, orchestrator ...
    return r.run(os.Stdout, orch)
}

// run accepts interfaces — testable without real deps.
func (r *RunCmd) run(w io.Writer, runner pipelineRunner) error {
    return runner.RunPipeline(ctx, orchestrator.PipelineInput{BeadID: r.BeadID})
}
```

### Exit Codes

Map errors to exit codes in `main()`, not in command methods. Commands return errors; `main()` translates:

```go
const (
    exitSuccess  = 0 // no error
    exitPipeline = 1 // pipeline phase failure or context cancellation
    exitSetup    = 2 // config, provider, or wiring error
)

func exitCode(err error) int {
    if err == nil {
        return exitSuccess
    }
    var pe *orchestrator.PipelineError
    if errors.As(err, &pe) {
        return exitPipeline
    }
    return exitSetup
}
```

### Testing

Use `kong.New` (not `kong.Parse`) to avoid `os.Exit` on parse errors. For `--version` testing, use `kong.Exit` to replace `os.Exit` with a recoverable panic:

```go
func TestVersionFlag(t *testing.T) {
    var cli CLI
    var buf bytes.Buffer
    k, err := kong.New(&cli,
        kong.Vars{"version": "v1.0.0 abc1234 2026-01-01"},
        kong.Writers(&buf, &buf),
        kong.Exit(func(int) { panic(errExitCalled) }),
    )
    if err != nil {
        t.Fatal(err)
    }
    defer func() {
        r := recover()
        // assert r is errExitCalled, then check buf.String()
    }()
    k.Parse([]string{"--version"})
}
```

### Antipatterns

| Antipattern | Risk | Fix |
|-------------|------|-----|
| `ctx.Command()` string switching | Fragile, bypasses `Run()` dispatch | Define `Run()` method on each command struct |
| Validation in `BeforeApply` | Positional args not populated yet | Use `AfterApply` for validation that needs args |
| `os.Exit` in `Run()` methods | Untestable, skips deferred cleanup | Return errors; let `main()` call `os.Exit` |
| Real deps in testable path | Tests need config files, providers | Split `Run()` / `run()`; pass interfaces to `run()` |
| `kong.Parse` in tests | Calls `os.Exit` on parse error | Use `kong.New` + `k.Parse()` for testable parsing |

- [Kong README](https://github.com/alecthomas/kong)
- [Kong examples](https://github.com/alecthomas/kong/tree/master/_examples)

## 9. YAML Configuration

Use `gopkg.in/yaml.v3` with strict decoding. Key patterns:

### Strict Unmarshaling

Always use `yaml.NewDecoder` with `KnownFields(true)` to catch typos. Without it, `provder: openai` silently produces an empty provider.

```go
dec := yaml.NewDecoder(bytes.NewReader(data))
dec.KnownFields(true) // rejects unknown fields
if err := dec.Decode(&cfg); err != nil {
    return nil, fmt.Errorf("config: parsing %s: %w", path, err)
}
```

**Avoid** `yaml.Unmarshal` for config files — it cannot enable strict mode.

### Layered Config with Pointer Disambiguation

Go zero values make it impossible to distinguish "field not set" from "field set to zero." Use pointer-based intermediate structs for layered merging:

```go
// rawConfig uses pointers to distinguish set vs unset
type rawConfig struct {
    Runtime *rawRuntime `yaml:"runtime"`
}
type rawRuntime struct {
    Timeout *time.Duration `yaml:"timeout"` // nil = not set, &0 = explicitly zero
}

// merge only overwrites fields that were explicitly set
if layer.Runtime.Timeout != nil {
    cfg.Runtime.Timeout = *layer.Runtime.Timeout
}
```

### Config Precedence

Follow 12-factor conventions. Later sources override earlier:

1. **Compiled defaults** (`DefaultConfig()`)
2. **User config** (`~/.config/capsule/config.yaml`)
3. **Project config** (`.capsule/config.yaml`)
4. **Environment variables** (`CAPSULE_*`)
5. **CLI flags** (highest priority)

### Validation

Add a `Validate()` method that checks invariants after all layers are applied. Call it once, after the final config is assembled — not per-layer.

```go
func (c *Config) Validate() error {
    if c.Runtime.Provider == "" {
        return errors.New("config: runtime.provider cannot be empty")
    }
    if c.Runtime.Timeout <= 0 {
        return fmt.Errorf("config: runtime.timeout must be positive, got %v", c.Runtime.Timeout)
    }
    return nil
}
```

### Antipatterns

| Antipattern | Risk | Fix |
|-------------|------|-----|
| `yaml.Unmarshal` for config | Silent typo acceptance | `yaml.NewDecoder` + `KnownFields(true)` |
| Zero-value merge confusion | Layer overwrites with defaults | Pointer-based raw structs |
| No validation after load | Invalid state at runtime | `Validate()` after full assembly |
| Secrets in YAML files | Credential leaks in VCS | Env vars or secret refs (`op://...`) |

- [go-yaml KnownFields](https://pkg.go.dev/gopkg.in/yaml.v3#Decoder.KnownFields)
- [12-Factor Config](https://12factor.net/config)

## 10. Subprocess Execution

Use `exec.Command` with separate arguments — never shell string concatenation. This prevents command injection because arguments are passed directly to the process, not interpreted by a shell.

```go
// Good: arguments are separate strings, safe from injection
cmd := exec.Command("git", "worktree", "add", "-b", branchName, wtPath, baseBranch)
cmd.Dir = repoRoot

// Bad: shell interprets special characters in user data
cmd := exec.Command("sh", "-c", "git worktree add -b "+branchName+" "+wtPath)
```

### Input Validation

Validate any external input before passing to `exec.Command`. Even with argument separation, certain tools interpret arguments as flags or special values:

```go
// Reject values that git might interpret as flags
if strings.HasPrefix(id, "-") {
    return fmt.Errorf("invalid id %q: must not start with -", id)
}

// Reject path traversal
if strings.ContainsAny(id, `/\`) || id == "." || id == ".." {
    return fmt.Errorf("invalid id %q", id)
}
```

### Working Directory

Always set `cmd.Dir` explicitly. Never rely on the process's inherited working directory — it may not be what you expect, especially in tests.

### Error Reporting

Capture stderr for diagnostic context. Always include the command's output in error messages:

```go
if out, err := cmd.CombinedOutput(); err != nil {
    return fmt.Errorf("git worktree add: %w\n%s", err, strings.TrimSpace(string(out)))
}
```

### Timeout Management

For long-running subprocesses, use `exec.CommandContext` with a deadline:

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
defer cancel()
cmd := exec.CommandContext(ctx, "claude", "-p", prompt)
```

### Antipatterns

| Antipattern | Risk | Fix |
|-------------|------|-----|
| `exec.Command("sh", "-c", userInput)` | Command injection | Separate arguments: `exec.Command("git", arg1, arg2)` |
| Accepting `-`-prefixed user input as args | Flag injection | Validate: reject strings starting with `-` |
| No `cmd.Dir` | Wrong working directory | Always set `cmd.Dir` explicitly |
| Discarding stderr | Silent failures | Capture with `CombinedOutput()` or `bytes.Buffer` |
| No timeout on external commands | Hung processes | Use `exec.CommandContext` with deadline |

- [Go Blog: Command PATH Security](https://go.dev/blog/path-security)
- [Snyk: Go Command Injection][snyk-go-injection]
- [Semgrep: Command Injection in Go][semgrep-go-injection]

## References

[functional-options]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
[snyk-go-injection]: https://snyk.io/blog/understanding-go-command-injection-vulnerabilities/
[semgrep-go-injection]: https://semgrep.dev/docs/cheat-sheets/go-command-injection
