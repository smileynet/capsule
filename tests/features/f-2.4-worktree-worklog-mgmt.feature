Feature: Worktree creation/cleanup and worklog lifecycle
  As a capsule developer, I want worktree creation and cleanup
  functions and worklog lifecycle management so that each mission
  gets an isolated workspace that is properly archived on completion.

  Scenario: Worktree Create produces isolated branch workspace
    Given a bead ID
    When worktree.Create runs
    Then .capsule/worktrees/<id>/ exists on branch capsule-<id>

  Scenario: Worklog Create populates mission briefing
    Given a bead context
    When worklog.Create runs
    Then worklog.md in worktree has mission briefing

  Scenario: Worklog Archive moves worklog to logs directory
    Given a completed mission
    When worklog.Archive runs
    Then worklog is in .capsule/logs/<id>/

  Scenario: Worktree Remove cleans up worktree and branch
    Given a worktree
    When worktree.Remove runs
    Then worktree and branch are cleaned up
