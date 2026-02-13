Feature: Pipeline pause and resume
  As a capsule user, I want to pause a running pipeline and resume it later
  so that I can interrupt long-running tasks without losing progress.

  Background:
    Given a pipeline with CheckpointStore enabled

  Scenario: Checkpoint saves phase results after each phase completion
    Given a 3-phase pipeline
    When all three phases complete with PASS
    Then the checkpoint is saved 3 times, once per phase
    And the final checkpoint contains 3 PhaseResult entries

  Scenario: Checkpoint records condition-skipped phases
    Given a phase with a condition that is not met
    When RunPipeline evaluates and skips the phase
    Then the checkpoint records a SKIP result for that phase

  Scenario: Checkpoint save errors are best-effort
    Given a CheckpointStore that returns an error on save
    When phases complete normally
    Then the pipeline still succeeds

  Scenario: Checkpoint load errors are best-effort
    Given a CheckpointStore that returns an error on load
    And a 3-phase pipeline
    When RunPipeline executes
    Then all three phases run with no phases skipped

  Scenario: Resume skips PASS phases from checkpoint
    Given a checkpoint with phase-a=PASS and phase-b=PASS
    And a 3-phase pipeline (phase-a, phase-b, phase-c)
    When RunPipeline resumes
    Then only phase-c is executed
    And phase-a and phase-b are skipped

  Scenario: Resume skips SKIP phases from checkpoint
    Given a checkpoint with phase-a=PASS and phase-b=SKIP
    And a 3-phase pipeline (phase-a, phase-b, phase-c)
    When RunPipeline resumes
    Then only phase-c is executed

  Scenario: Resume reruns ERROR phases from checkpoint
    Given a checkpoint with phase-a=PASS and phase-b=ERROR
    And a 3-phase pipeline (phase-a, phase-b, phase-c)
    When RunPipeline resumes
    Then both phase-b and phase-c are executed

  Scenario: Resume reruns NEEDS_WORK phases from checkpoint
    Given a checkpoint with phase-a=PASS and phase-b=NEEDS_WORK
    And a 3-phase pipeline (phase-a, phase-b, phase-c)
    When RunPipeline resumes
    Then both phase-b and phase-c are executed

  Scenario: Resume merges checkpoint skip set with input SkipPhases
    Given a checkpoint with phase-a=PASS
    And input SkipPhases includes phase-b
    When RunPipeline resumes
    Then phase-a is skipped from checkpoint
    And phase-b is skipped from input SkipPhases
    And only phase-c is executed

  Scenario: SIGUSR1 triggers pause and saves checkpoint
    Given a running pipeline with pause trigger registered
    When SIGUSR1 is sent between phases
    Then the pipeline returns ErrPipelinePaused
    And the checkpoint is saved with completed phase results

  Scenario: Pause before any phase runs saves empty checkpoint
    Given a pipeline where pause is requested before the first phase
    When RunPipeline starts
    Then it returns ErrPipelinePaused immediately
    And the checkpoint contains zero PhaseResult entries

  Scenario: Paused pipeline skips post-pipeline actions
    Given a RunCmd where the pipeline returns ErrPipelinePaused
    When RunCmd handles the result
    Then merge and bead-close are not called
    And the output contains "Pipeline paused" with a resume hint
