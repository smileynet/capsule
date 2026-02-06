Feature: Sign-off and merge-to-main headless claude prompt pair
  As a capsule developer, I want a sign-off prompt that performs final
  validation and a merge script that lands clean code on main so that
  completed work is properly integrated and archived.

  Scenario: Sign-off validates quality of passing tests
    Given a worktree with passing tests
    When claude -p runs with sign-off prompt
    Then it validates quality
    And returns PASS or NEEDS_WORK

  Scenario: Merge lands only implementation and test files on main
    Given PASS from sign-off
    When merge.sh runs
    Then only implementation and test files are on main
    And worklog.md is not on main

  Scenario: Worklog is archived after successful merge
    Given successful merge
    When ls .capsule/logs/<bead-id>/ runs
    Then worklog.md is archived there

  Scenario: Mission worktree is removed after merge
    Given successful merge
    When git worktree list runs
    Then the mission worktree is removed

  Scenario: Bead status is closed after merge
    Given successful merge
    When bd show <bead-id> runs
    Then the bead status is closed
