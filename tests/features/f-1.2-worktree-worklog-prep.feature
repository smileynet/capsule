Feature: Worktree creation and worklog instantiation from bead template
  As a capsule developer, I want the prep phase to create a git worktree
  and instantiate a worklog from the bead template so that each mission
  starts with proper context and isolation.

  Scenario: Prep script creates git worktree for bead
    Given a template project with beads
    When prep.sh <bead-id> runs
    Then a git worktree exists at .capsule/worktrees/<bead-id>/

  Scenario: Worklog contains mission briefing with bead context
    Given the worktree
    When cat worklog.md runs
    Then the worklog contains Mission Briefing with epic/feature/task context
    And the worklog contains acceptance criteria

  Scenario: Worktree branch follows naming convention
    Given the worktree
    When git branch runs
    Then the branch is named capsule-<bead-id>

  Scenario: Worklog is bead-specific not generic
    Given the worklog template
    When instantiated for different beads
    Then each worklog has bead-specific content
