---
name: changelog
description: Update CHANGELOG.md with user-focused entries. Use when completing features, closing milestones, or preparing releases.
argument-hint: [feature-description]
---

# Changelog Skill

Update CHANGELOG.md following Keep a Changelog format with user-focused entries.

## Core Principle

**Changelogs are for users, not developers.** Every entry describes observable behavior change from the user's perspective. Ask: "What is different for the person using this software?"

## Process

1. Read CHANGELOG.md
2. Determine what changed from the user's perspective
3. Add entries under `## [Unreleased]` in the appropriate category
4. If $ARGUMENTS provided, use that as context for the entry

## Categories (in order)

| Category | When to use |
|----------|-------------|
| **Added** | New capabilities users can now do |
| **Changed** | Existing behavior that works differently |
| **Deprecated** | Features marked for future removal |
| **Removed** | Features no longer available |
| **Fixed** | Bugs that affected users, now resolved |
| **Security** | Vulnerability patches |

Only include categories that have entries. Never leave empty sections.

## Writing Good Entries

**Format:** Imperative mood, one line, scannable.

**The test:** If you remove all code context, does the entry still make sense to someone who only *uses* the software? If not, rewrite it.

| Developer thinking | User-facing entry |
|--------------------|-------------------|
| Refactored AuthService to strategy pattern | *(omit — no user impact)* |
| Added retry logic to HTTP client | File uploads now recover automatically from network interruptions |
| Fixed null pointer in auth middleware | Fix crash when logging in with expired session |
| Optimized database query | Search results load 3x faster on large datasets |
| Bumped Go to 1.21 | **Breaking:** Requires Go 1.21 or later |

## Antipatterns — Do NOT

- **Dump the git log.** A changelog is curated, not a commit history
- **List implementation details.** "Refactored X to use Y pattern" is developer noise
- **Be vague.** "Bug fixes and improvements" communicates nothing
- **List internal refactors.** If behavior didn't change for users, omit it
- **List dependency bumps.** Unless they fix a user-facing bug or security issue
- **Include test/CI changes.** These are invisible to users
- **Reference bead IDs without context.** IDs are for linking, not the primary description

## What to Include vs Exclude

**Include:**
- New features and capabilities
- Changes to existing behavior (especially breaking)
- Bug fixes that affected users
- Security patches
- Performance improvements with user-visible impact
- Deprecation and removal notices
- Changes to runtime requirements

**Exclude:**
- Internal refactoring
- Developer tooling (CI, linters, formatters)
- Dependency bumps with no user impact
- Test additions or changes
- Code style changes
- Documentation formatting fixes

## Example

```markdown
## [Unreleased]

### Added
- Run a full AI-driven pipeline on any task with a single command
- Automatically retry failed phases with reviewer feedback

### Fixed
- Fix crash when task has no acceptance criteria defined

### Security
- Sanitize bead IDs to prevent path traversal in worktree creation
```
