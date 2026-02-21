# Feature: Agent-driven merge conflict resolution (cap-9f0.2)
Feature: Agent-driven merge conflict resolution
  As a developer running a campaign
  I want merge conflicts to be resolved automatically by the agent pair when possible
  And to receive a non-blocking notification when manual intervention is needed
  So that campaigns can self-heal without requiring me to watch them

  Background:
    Given a campaign is running with PostTaskFunc wired for merge/cleanup
    And a task has completed its pipeline successfully

  Scenario: Merge conflict triggers agent pair with conflict context
    Given the merge of "task-2" into main produces a merge conflict
    When PostTaskFunc handles the ErrMergeConflict
    Then the execute and sign-off agent pair should be invoked
    And the agent pair should receive the conflicting file paths and diff context

  Scenario: Agent pair resolves conflict and merge succeeds
    Given a merge conflict has triggered the agent pair
    And the agent pair produces a valid resolution
    When the merge is retried
    Then the merge should succeed
    And the worktree should be removed and pruned
    And the campaign should continue to the next task

  Scenario: Unresolvable conflict pauses campaign after max retries
    Given a merge conflict has triggered the agent pair
    And the agent pair cannot resolve the conflict after max retries
    Then the campaign should pause at the conflicted task
    And the campaign state should be saved with the paused status

  Scenario: Non-blocking notification for unresolved conflict
    Given a merge conflict could not be resolved after max retries
    When the campaign pauses
    Then a non-blocking notification should inform the user of the unresolved conflict
    And the notification should include the conflicting file paths

  Scenario: Paused campaign is resumable from the conflicted task
    Given a campaign was paused due to an unresolved merge conflict
    When the user manually resolves the conflict and resumes the campaign
    Then the campaign should resume from the conflicted task
    And the merge should be retried before advancing to the next task
