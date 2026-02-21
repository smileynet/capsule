# Feature: Worktree merge/cleanup per campaign task (cap-9f0.1)
Feature: Worktree merge/cleanup per campaign task
  As a developer running a multi-task campaign
  I want each task's changes merged to main before the next task starts
  So that tasks build on each other's work and worktrees don't accumulate

  Background:
    Given a campaign is running with multiple tasks
    And each task creates a git worktree for its pipeline

  Scenario: Successful task worktree is merged, removed, and pruned before next task
    Given a task "task-1" has completed its pipeline successfully
    When the campaign advances to the next task
    Then "task-1" changes should be merged to main
    And "task-1" worktree directory should be removed
    And git worktree prune should have been called
    And the next task should not start until cleanup completes

  Scenario: Campaign tasks branch from updated main
    Given "task-1" has been merged to main
    When "task-2" starts its pipeline
    Then "task-2" worktree should branch from the updated main
    And the worktree should contain "task-1" changes

  Scenario: PostTaskFunc is injectable without importing worktree package
    Given a campaign runner configured with a PostTaskFunc
    Then the campaign package should not import the worktree package
    And PostTaskFunc should receive the completed task's bead ID and result

  Scenario: CLI campaign command wires PostTaskFunc with merge/cleanup logic
    Given the CLI campaign command is invoked
    When the campaign runner is constructed
    Then PostTaskFunc should be wired to the existing merge and cleanup logic
    And each successful task should trigger merge, worktree removal, and prune

  Scenario: Dashboard campaign dispatch wires PostTaskFunc identically
    Given a campaign is dispatched from the dashboard
    When the campaign runner is constructed
    Then PostTaskFunc should be wired to the same merge and cleanup logic as CLI
    And each successful task should trigger merge, worktree removal, and prune
