Feature: Full pipeline orchestration script and command documentation
  As a capsule developer, I want a single pipeline script that
  orchestrates all phases end-to-end and comprehensive command
  documentation so that the entire workflow is automated and
  reproducible.

  Scenario: Pipeline script executes all phases in order
    Given a template project
    When run-pipeline.sh <bead-id> runs
    Then the full pipeline executes in order: prep, test-write, test-review, execute, execute-review, sign-off, merge

  Scenario: Pipeline retries worker phase on reviewer NEEDS_WORK
    Given the pipeline
    When a reviewer phase returns NEEDS_WORK
    Then the worker phase retries with feedback
    And retries up to configured max

  Scenario: Pipeline archives full worklog on completion
    Given the pipeline completes
    When reviewing .capsule/logs/<bead-id>/
    Then the full worklog with all phase entries is archived

  Scenario: Commands documentation covers all invocations
    Given docs/commands.md
    When reading it
    Then every claude command, script invocation, and expected output is documented
