---
name: acceptance
description: Fill out acceptance reports for features and epics. Use when a feature or epic is ready for sign-off.
argument-hint: [feature-or-epic-id]
---

# Acceptance Skill

Fill out acceptance reports using the templates in `docs/templates/`.

## Core Principle

**"Is this done, and how do we know?"** Every acceptance report answers this with evidence. A claim without proof is fiction.

## Process

1. Read the relevant template (`docs/templates/feature-acceptance.md` or `docs/templates/epic-acceptance.md`)
2. Read the bead (`bd show <id>`) to get title, description, acceptance criteria, and parent epic
3. For features: find tests that cover each criterion, run them, record evidence
4. For epics: confirm all child features have acceptance reports, then verify E2E journeys
5. Fill out the template — every feature criterion needs an evidence column entry; every epic needs E2E verification commands
6. Write the report to `docs/features/<id>-acceptance.md` or `docs/epics/<id>-acceptance.md`
7. If $ARGUMENTS provided, use that as the feature/epic ID

## Writing Good Criteria

**Declarative, not imperative.** Describe what the system does, not what the developer did.

**Binary.** Every criterion is pass or fail. If you can't answer yes/no, the criterion is too vague.

**Evidence-linked.** Every "Verified: Yes" needs a test name, command, or output reference.

**3-5 per feature.** More suggests the feature should be split. Fewer suggests missing coverage.

| Developer thinking | Acceptance criterion |
|--------------------|---------------------|
| Implemented retry logic in HTTP client | File uploads recover from network interruptions |
| Added selective staging in merge script | Only implementation files appear on main after merge |
| Wrote test for edge case | Merge fails gracefully when worktree is missing |
| Refactored to strategy pattern | *(omit — no observable behavior change)* |

## Antipatterns — Do NOT

- **Vague criteria.** "Works correctly" or "Handles errors" is not verifiable
- **No evidence.** "Verified: Yes" with an empty evidence column is rubber-stamping
- **Implementation details.** "Uses selective git staging" belongs in a commit message, not acceptance
- **Over-documentation.** Deliverables tables (file paths are in git), task checklists (beads tracks this), review summaries (captured in bead comments) are internal bookkeeping noise
- **Retroactive rubber-stamping.** Writing acceptance after the fact to check a box. If criteria weren't defined before implementation, say so honestly in Known Limitations
- **Kitchen-sink criteria.** Listing every test case as a separate criterion. Group by observable behavior

## What to Include vs Exclude

**Include:**
- Observable behavior from the user's perspective
- Exact commands to reproduce verification
- Known limitations with linked issues for deferred work
- Explicit out-of-scope boundaries

**Exclude:**
- File paths and deliverables (git history is the source of truth)
- Task/bead checklists (beads tracks completion)
- Review verdicts (captured in bead comments)
- Test pass/fail counts (evidence column links to specific tests)
- Internal architecture decisions (use ADRs for these)
