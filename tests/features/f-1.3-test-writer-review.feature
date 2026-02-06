Feature: Test-writer and test-review headless claude prompt pair
  As a capsule developer, I want a test-writer prompt that generates
  tests and a test-review prompt that validates them so that the
  red-green cycle begins with meaningful failing tests.

  Scenario: Test-writer creates test files in worktree
    Given a worktree with worklog
    When claude -p runs with test-writer prompt
    Then test files are created in the worktree

  Scenario: Test-writer output contains structured JSON signal
    Given test-writer output
    When the last JSON block is parsed
    Then it contains status, feedback, and files_changed

  Scenario: Test-review validates created tests
    Given created tests
    When claude -p runs with test-review prompt
    Then it returns PASS or NEEDS_WORK with specific feedback

  Scenario: Test-writer improves tests based on review feedback
    Given NEEDS_WORK from test-review
    When test-writer re-runs with feedback
    Then tests improve based on feedback
