# Go Conventions

Concise reference for anyone writing Go code in this project. Each section links to authoritative sources.

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

Accept interfaces, return structs. Define interfaces where they're consumed, not where they're implemented — this keeps packages decoupled.

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

- [Dave Cheney: Functional options for friendly APIs](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis)
- [Uber Go Style Guide: Dependency Injection](https://github.com/uber-go/guide/blob/master/style.md)

## 7. Code Style

Run `gofmt` (or `goimports`) on every save. Add a doc comment to every exported symbol — `go doc` and IDE tooling depend on it.

```go
// Provider executes AI completions against a configured backend.
type Provider struct { /* ... */ }
```

Comments should explain *why*, not *what*. The code already says what.

- [Effective Go: Commentary](https://go.dev/doc/effective_go#commentary)
- [Go Doc Comments](https://go.dev/doc/comment)

## 8. CLI (Kong)

Use [Kong](https://github.com/alecthomas/kong) for CLI parsing. Define commands as nested structs with a `Run(...deps) error` method on each leaf command. Keep `main.go` thin — parse, bind, and dispatch.

```go
// cmd/capsule/main.go
type CLI struct {
    Version kong.VersionFlag `help:"Show version."`
    Run     RunCmd           `cmd:"" help:"Run a capsule."`
}

type RunCmd struct {
    Config string `arg:"" type:"existingfile" help:"Path to capsule config."`
}

func (r *RunCmd) Run(orch *capsule.Orchestrator) error {
    return orch.Execute(r.Config)
}

func main() {
    var cli CLI
    ctx := kong.Parse(&cli, kong.Vars{"version": version})
    orch := capsule.NewOrchestrator(/* ... */)
    ctx.Bind(orch)
    ctx.FatalIfErrorf(ctx.Run())
}
```

In tests, use `kong.New` (not `kong.Parse`) to avoid `os.Exit` on error.

```go
func TestRunCmd(t *testing.T) {
    var cli CLI
    k, err := kong.New(&cli)
    if err != nil {
        t.Fatal(err)
    }
    kctx, err := k.Parse([]string{"run", "testdata/config.yaml"})
    if err != nil {
        t.Fatal(err)
    }
    // Bind test doubles, then kctx.Run()
}
```

**Avoid:**
- `ctx.Command()` string switching — use `Run()` methods instead.
- Validation in `BeforeApply` — prefer `AfterApply` (positional arguments aren't populated yet).

- [Kong README](https://github.com/alecthomas/kong)
- [Kong examples](https://github.com/alecthomas/kong/tree/master/_examples)
