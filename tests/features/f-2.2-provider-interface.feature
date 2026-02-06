Feature: Provider interface wrapping headless claude subprocess
  As a capsule developer, I want a Provider interface that wraps the
  headless claude subprocess so that prompt execution is abstracted
  behind a clean contract with structured results.

  Scenario: Provider Execute returns structured Result
    Given a Provider
    When Execute(ctx, prompt, workDir) is called
    Then it returns Result with Output, ExitCode, Duration, and Signal

  Scenario: ClaudeProvider invokes claude with correct flags
    Given ClaudeProvider
    When Execute runs
    Then it invokes claude -p with correct flags in workDir

  Scenario: Provider respects timeout context
    Given a timeout context
    When the subprocess exceeds it
    Then the process is killed
    And a TimeoutError is returned

  Scenario: Result contains parsed phase signal from JSON output
    Given claude output with JSON signal
    When Result is returned
    Then Signal field contains parsed PhaseResult
