# Manual Verification Walkthrough

Step-by-step validation of all Go CLI features against the demo-brownfield template.

## 1. Prerequisites

- Go 1.22+
- git
- `bd` (beads CLI)
- `jq`
- Optional: `claude` CLI (Section 8 only)

Build the binary:

```bash
make build
```

## 2. Demo Project Setup

Create a test environment from the demo-brownfield template:

```bash
scripts/setup-template.sh --template=demo-brownfield /tmp/capsule-demo
```

The Go CLI resolves `prompts/` and `templates/worklog.md.template` relative to CWD
([cmd/capsule/main.go:106-108][main-paths]). The demo template doesn't include these,
so copy them into the demo project:

```bash
cp -r prompts/ /tmp/capsule-demo/prompts/
cp -r templates/ /tmp/capsule-demo/templates/
```

Verify bead fixtures loaded:

```bash
cd /tmp/capsule-demo && bd list
```

Expected: 4 beads listed (`demo-1`, `demo-1.1`, `demo-1.1.1`, `demo-1.1.2`).

## 3. CLI Basics

Run from the capsule repo root (no demo project needed). The binary path assumes
`make build` output at `./capsule`.

### 3.1 Version

```bash
./capsule --version
```

**Expected:** Single line with version, commit hash, and build date
(e.g. `dev unknown unknown` for local builds).

**Verify:** Exit code 0.

### 3.2 No arguments

```bash
./capsule
```

**Expected:** Non-zero exit, usage text listing `run`, `abort`, `clean` commands.

### 3.3 Run without bead-id

```bash
./capsule run
```

**Expected:** Non-zero exit, error mentioning `<bead-id>` as a required argument.

### 3.4 Unknown provider

```bash
./capsule run test-bead --provider=fake
```

**Expected:** Exit code 2, error message:
`error: run: unknown provider "fake" (available: claude, kiro)`.

**Verify:** `echo $?` returns `2`.

### 3.5 Flag parsing

```bash
./capsule run some-bead --timeout=60 2>&1; echo "exit: $?"
```

**Expected:** Fails at provider execution (not at flag parsing). The error message
should reference a pipeline phase, not an unknown flag.

## 4. Infrastructure Verification

Run from the demo project directory. The pipeline creates a worktree and worklog,
then fails at the first provider call since `claude` CLI is not available.

```bash
cd /tmp/capsule-demo
CAPSULE_BIN="<path-to-capsule-repo>/capsule"
```

### 4.1 Run pipeline (expect failure at provider)

To test infrastructure without running the full pipeline, hide the `claude` CLI
from PATH:

```bash
PATH=/usr/bin:/bin $CAPSULE_BIN run demo-1.1.1
```

**Expected:** Exit code 1, status line `[HH:MM:SS] [1/6] test-writer running`
followed by error:
`pipeline: phase "test-writer" attempt 1: executing test-writer: provider: claude: exec: "claude": executable file not found in $PATH:`.

If `claude` is available on your PATH, the pipeline will proceed
(see Section 8 instead). To force the infrastructure-only path, use the
restricted PATH shown above.

### 4.2 Verify worktree created

```bash
ls .capsule/worktrees/demo-1.1.1/
```

**Expected:** Directory exists with worktree contents (at minimum a `.git` file).

### 4.3 Verify worklog created

```bash
cat .capsule/worktrees/demo-1.1.1/worklog.md
```

**Expected:** Worklog with template structure:

- `# Worklog: demo-1.1.1` header with task ID
- `## Mission Briefing` with Epic/Feature/Task sections populated from bead hierarchy
- `## Phase Log` with phases listed as `_Status: pending_`

The worklog resolves the full bead hierarchy: task → feature → epic. When `bd` is
available and the bead exists, all three levels are populated with titles, goals,
and descriptions.

### 4.4 Verify bead context populated

```bash
head -20 .capsule/worktrees/demo-1.1.1/worklog.md
```

**Expected:** Epic, Feature, and Task sections all populated with titles and descriptions
from the bead hierarchy (not empty).

### 4.5 Verify git branch

```bash
git branch | grep capsule-demo-1.1.1
```

**Expected:** Branch `capsule-demo-1.1.1` exists.

## 5. Abort Command

With the worktree from Section 4 still present:

### 5.1 Abort the capsule

```bash
$CAPSULE_BIN abort demo-1.1.1
```

**Expected:** `Aborted capsule demo-1.1.1 (branch preserved)`, exit code 0.

### 5.2 Verify worktree removed

```bash
ls .capsule/worktrees/demo-1.1.1/
```

**Expected:** `No such file or directory`.

### 5.3 Verify branch preserved

```bash
git branch | grep capsule-demo-1.1.1
```

**Expected:** Branch `capsule-demo-1.1.1` still exists.

### 5.4 Error case: abort again

```bash
$CAPSULE_BIN abort demo-1.1.1
```

**Expected:** Exit code 2, error: `abort: no worktree found for "demo-1.1.1"`.

## 6. Clean Command

### 6.1 Re-create worktree

First, delete the branch left by abort so `run` can recreate it:

```bash
git branch -D capsule-demo-1.1.1
$CAPSULE_BIN run demo-1.1.1
```

**Expected:** Pipeline fails at provider (exit 1), but worktree and branch are created.

### 6.2 Clean up fully

```bash
$CAPSULE_BIN clean demo-1.1.1
```

**Expected:** `Cleaned capsule demo-1.1.1`, exit code 0.

### 6.3 Verify worktree removed

```bash
ls .capsule/worktrees/demo-1.1.1/
```

**Expected:** `No such file or directory`.

### 6.4 Verify branch removed

```bash
git branch | grep capsule-demo-1.1.1
```

**Expected:** No output (branch deleted).

### 6.5 Error case: clean again

```bash
$CAPSULE_BIN clean demo-1.1.1
```

**Expected:** Exit code 2, error: `clean: no worktree found for "demo-1.1.1"`.

## 7. Configuration

### 7.1 Defaults work without config file

No `.capsule/config.yaml` is needed for the pipeline to run. Default provider is
`claude`, default timeout is `5m`, default worktree base dir is `.capsule/worktrees`.

### 7.2 Project config file

Create a config file and verify it's loaded:

```bash
mkdir -p .capsule
cat > .capsule/config.yaml <<'EOF'
runtime:
  timeout: 10m
EOF
```

Run and verify no config errors:

```bash
$CAPSULE_BIN run demo-1.1.1 2>&1 | head -5
```

**Expected:** Pipeline runs (and fails at provider) without config errors.

Clean up:

```bash
rm .capsule/config.yaml
$CAPSULE_BIN clean demo-1.1.1 2>/dev/null; git branch -D capsule-demo-1.1.1 2>/dev/null
```

### 7.3 Environment variable override

```bash
CAPSULE_PROVIDER=fake $CAPSULE_BIN run demo-1.1.1
```

**Known limitation:** The env var is overridden by the Kong CLI flag default
(`--provider=claude`). The pipeline will run with provider `claude`, not `fake`.
This is because `RunCmd.Run()` unconditionally sets `cfg.Runtime.Provider = r.Provider`
after `ApplyEnv()`, and Kong always populates the default.

To verify env vars are loaded, check `CAPSULE_TIMEOUT`:

```bash
CAPSULE_TIMEOUT=invalid $CAPSULE_BIN run demo-1.1.1
```

**Expected:** Exit code 2, error about invalid `CAPSULE_TIMEOUT`.

### 7.4 CLI flag override

```bash
$CAPSULE_BIN run demo-1.1.1 --provider=fake
```

**Expected:** Exit code 2, error: `run: unknown provider "fake" (available: claude, kiro)`.

### 7.5 Bead resolve warning

```bash
$CAPSULE_BIN run nonexistent-bead 2>&1 | head -2
```

**Expected:** Warning line followed by pipeline progress:
```
warning: bead "nonexistent-bead" not found (try: bd ready)
[HH:MM:SS] [1/6] test-writer running
```

The pipeline continues despite the missing bead — the worklog is created with only
the task ID, and Epic/Feature sections are omitted.

## 8. Full Pipeline (requires `claude` CLI)

Skip this section if `claude` is not installed.

```bash
command -v claude >/dev/null || echo "SKIP: claude CLI not available"
```

### 8.1 Run full pipeline

```bash
$CAPSULE_BIN run demo-1.1.1
```

**Expected output:** Timestamped status lines for each phase:

```
[HH:MM:SS] [1/6] test-writer running
[HH:MM:SS] [1/6] test-writer passed
[HH:MM:SS] [2/6] test-review running
...
[HH:MM:SS] [6/6] merge passed
```

### 8.2 Verify worklog archived

```bash
cat .capsule/logs/demo-1.1.1/worklog.md
```

**Expected:** Worklog with phase entries showing status and timestamps.

### 8.3 Verify implementation on branch

```bash
git log capsule-demo-1.1.1 --oneline -5
```

**Expected:** Commits from the pipeline phases on the `capsule-demo-1.1.1` branch.

## 9. Cleanup

```bash
rm -rf /tmp/capsule-demo
```

## Source References

| Reference | Location |
|-----------|----------|
| Relative path hardcodes | [cmd/capsule/main.go:106-108][main-paths] |
| Plain text callback format | [cmd/capsule/main.go:297-319][callback] |
| Exit code mapping | [cmd/capsule/main.go:284-293][exitcode] |
| Branch naming (`capsule-<id>`) | [internal/worktree/worktree.go:67][branch] |
| Default phases (6-phase pipeline) | [internal/orchestrator/phases.go:57-65][phases] |
| Worklog template | [templates/worklog.md.template][template] |
| Demo template setup | [scripts/setup-template.sh][setup] |

[main-paths]: ../cmd/capsule/main.go#L106-L108
[callback]: ../cmd/capsule/main.go#L297-L319
[exitcode]: ../cmd/capsule/main.go#L284-L293
[branch]: ../internal/worktree/worktree.go#L67
[phases]: ../internal/orchestrator/phases.go#L57-L65
[template]: ../templates/worklog.md.template
[setup]: ../scripts/setup-template.sh
