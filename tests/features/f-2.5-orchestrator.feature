Feature: Orchestrator sequencing phase pairs with retry logic
  As a capsule developer, I want an orchestrator that sequences phase
  pairs with retry logic so that the pipeline progresses through
  worker-reviewer cycles with proper error handling.

  Scenario: Pipeline runs phases in correct order
    Given a bead ID
    When RunPipeline executes
    Then phases run in order: test-writer, test-review, execute, execute-review, sign-off, merge

  Scenario: Reviewer NEEDS_WORK triggers worker retry with feedback
    Given NEEDS_WORK from a reviewer
    When retry logic runs
    Then the worker phase re-runs with feedback
    And retries up to configured max

  Scenario: Pipeline aborts on ERROR from any phase
    Given ERROR from any phase
    When pipeline detects it
    Then pipeline aborts with diagnostic error

  Scenario: Status callback fires on each phase transition
    Given each phase transition
    When StatusCallback fires
    Then it includes phase, status, attempt, and maxRetry

  Scenario: Successful pipeline merges worktree and archives worklog
    Given successful pipeline
    When complete
    Then worktree is merged
    And worklog is archived
