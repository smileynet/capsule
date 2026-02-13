Feature: Advanced retry strategies for pipeline phase pairs
  As a capsule developer, I want configurable retry strategies with backoff
  and provider escalation so that the pipeline can adapt its approach when
  initial attempts fail.

  Background:
    Given a pipeline with worker-reviewer phase pairs
    And retry strategy configuration via WithRetryDefaults

  Scenario: ResolveRetryStrategy unifies retry configuration
    Given a reviewer phase with MaxRetries=5
    And pipeline-level retry defaults with MaxAttempts=3
    When ResolveRetryStrategy resolves the reviewer's strategy
    Then the effective MaxAttempts is 5 (phase overrides pipeline)
    And BackoffFactor and EscalateProvider come from pipeline defaults

  Scenario: BackoffFactor multiplies timeout per retry attempt
    Given a phase pair with Timeout=30s on both phases
    And BackoffFactor=2.0 in the retry strategy
    When the reviewer returns NEEDS_WORK on attempt 1
    Then attempt 1 uses 30s timeout (30s * 2^0)
    And attempt 2 uses 60s timeout (30s * 2^1)

  Scenario: BackoffFactor has no effect when phases have no timeout
    Given a phase pair without Timeout set
    And BackoffFactor=2.0 in the retry strategy
    When retries execute
    Then no deadline is applied to any attempt

  Scenario: EscalateProvider switches provider after N attempts
    Given EscalateProvider="alternate" and EscalateAfter=1
    And an alternate provider is registered
    When the reviewer returns NEEDS_WORK on attempt 1
    Then attempt 1 uses the default provider for worker and reviewer
    And attempt 2 uses the alternate provider for worker and reviewer

  Scenario: EscalateProvider has no effect when empty
    Given EscalateProvider is empty and EscalateAfter=1
    When retries execute through multiple attempts
    Then all attempts use the default provider

  Scenario: Unknown EscalateProvider returns error on escalation
    Given EscalateProvider="nonexistent" and EscalateAfter=1
    And no provider named "nonexistent" is registered
    When escalation is triggered on attempt 2
    Then a PipelineError is returned mentioning the unknown provider name
