Feature: Configurable phase definitions
  As a capsule developer, I want to configure per-phase conditions, providers,
  and timeouts so that the pipeline adapts to project needs without code changes.

  Background:
    Given a pipeline with configurable PhaseDefinition entries
    And each phase supports Condition, Provider, and Timeout fields

  Scenario: Condition skips phase when glob matches no files
    Given a phase with Condition: "files_match:src/*.rb"
    And the worktree contains no .rb files
    When RunPipeline executes the phase
    Then the phase is skipped with status SKIP
    And feedback indicates "condition not met"

  Scenario: Condition runs phase when glob matches files
    Given a phase with Condition: "files_match:src/*.go"
    And the worktree contains src/main.go
    When RunPipeline executes the phase
    Then the phase runs normally and receives a provider signal

  Scenario: Provider override uses named provider
    Given a phase with Provider: "alternate"
    And a provider named "alternate" is registered in the registry
    When executePhase runs the phase
    Then the alternate provider is called instead of the default

  Scenario: Empty provider uses orchestrator default
    Given a phase with Provider: "" (empty string)
    When executePhase runs the phase
    Then the orchestrator's default provider is used

  Scenario: Timeout applies context.WithTimeout to phase execution
    Given a phase with Timeout: 30s
    When executePhase runs the phase
    Then the provider receives a context with a 30s deadline

  Scenario: No timeout means no deadline
    Given a phase with Timeout: 0 (unset)
    When executePhase runs the phase
    Then the provider receives the parent context without additional deadline
