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
2. **Git repository**: Run `git -C <target> rev-parse --git-dir`. Must be a git repo.
3. **Beads CLI**: Run `bd --version`. Report version or note it's missing.
4. **Prompt files**: Check `<target>/prompts/` for all 7 required files:
   - `test-writer.md`, `test-review.md`, `execute.md`, `execute-review.md`, `sign-off.md`, `merge.md`, `summary.md`
5. **Worklog template**: Check `<target>/templates/worklog.md.template` exists.
6. **Beads initialized**: Run `bd list` in the target directory. If it fails, beads isn't initialized.

## Output

Summarize results as a checklist. For any missing items, explain how to fix them:
- Missing prompts → copy from capsule repo's `prompts/` directory
- Missing template → copy from capsule repo's `templates/` directory
- Beads not initialized → run `bd init`

If everything passes, end with: **Ready. Run `/capsule-run <bead-id>` to start a pipeline.**
