Feature: Execute and execute-review headless claude prompt pair
  As a capsule developer, I want an execute prompt that writes
  implementation code and an execute-review prompt that validates it
  so that the green phase produces quality code that passes all tests.

  Scenario: Execute prompt creates passing implementation
    Given a worktree with failing tests
    When claude -p runs with execute prompt
    Then implementation code is created that passes the tests

  Scenario: Execute output contains structured JSON signal
    Given execute output
    When parsed
    Then JSON signal contains status with PASS, NEEDS_WORK, or ERROR
    And JSON signal contains files_changed

  Scenario: Execute-review validates implementation quality
    Given implementation
    When claude -p runs with execute-review prompt
    Then it verifies tests pass
    And code quality is acceptable

  Scenario: Execute improves implementation based on review feedback
    Given NEEDS_WORK from review
    When execute re-runs with feedback
    Then implementation improves
