---
allowed-tools: Bash, Read, Glob, Grep
argument-hint: "[target-dir]"
description: Verify a project is ready for capsule pipelines
---

Check whether the target project has everything capsule needs to run a pipeline. Report what's present, what's missing, and how to fix it.

**Target directory:** Use `$ARGUMENTS` if provided, otherwise use the current working directory.

## Pre-flight checks

Run these checks and collect results (pass/fail + details for each):

1. **capsule binary**: Run `capsule --version`. Report the version or note it's missing.
2. **Git repository**: Run `git -C <target> rev-parse --git-dir`. Must be a git repo. Report branch and commit count.
3. **Beads CLI**: Run `bd --version`. Report version or note it's missing.
4. **Prompt files**: Check `<target>/prompts/` for all 7 required files:
   - `test-writer.md`, `test-review.md`, `execute.md`, `execute-review.md`, `sign-off.md`, `merge.md`, `summary.md`
5. **Worklog template**: Check `<target>/templates/worklog.md.template` exists.
6. **Beads initialized**: Run `bd list` in the target directory. If it works, count issues and check `bd ready --json` for available tasks.

## Output format

Show a narrative readiness report:

~~~
CAPSULE SETUP: <target-dir>
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Checking prerequisites for capsule pipeline...

  capsule binary    <version>                          ✓
  git repository    <branch> (<commit count> commits)  ✓
  beads CLI         <version>                          ✓
  prompt templates  7/7 files in prompts/              ✓
  worklog template  templates/worklog.md.template      ✓
  beads initialized <N> issues (<N> ready)             ✓

───────────────────────────────────────────
RESULT
───────────────────────────────────────────

Ready. All 6 checks passed.

Available tasks:
  <id> [P<n>] <title>
  <id> [P<n>] <title>

NEXT STEP: /capsule-run <bead-id>
~~~

If anything is missing, show remediation with exact commands:

~~~
  prompt templates  0/7 files in prompts/              ✗

    Missing: test-writer.md, test-review.md, execute.md,
             execute-review.md, sign-off.md, merge.md, summary.md

    Fix: cp -r /path/to/capsule/prompts/ <target>/prompts/

  capsule binary    not found on PATH                  ✗

    Fix: cd /path/to/capsule && make build
         Then: export PATH="$PATH:/path/to/capsule"
         Or use absolute path: /path/to/capsule/capsule run <id>

───────────────────────────────────────────
RESULT
───────────────────────────────────────────

Not ready. <N> issues to fix (see above).
~~~
