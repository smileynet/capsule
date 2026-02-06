Feature: Bubble Tea model for phase pipeline status display
  As a capsule user, I want a Bubble Tea TUI that displays pipeline
  phase status so that I can monitor progress with clear visual
  indicators for each phase.

  Scenario: TUI displays all phases with current status
    Given a running pipeline
    When the TUI renders
    Then I see all phases listed with current status
    And statuses include pending, running, pass, and fail

  Scenario: Active phase shows animated spinner
    Given an active phase
    When the TUI renders
    Then a spinner animates next to the active phase name

  Scenario: Retry attempt counter is displayed
    Given a retry
    When the TUI updates
    Then the attempt counter shows the current and max attempts

  Scenario: Pipeline completion shows summary
    Given pipeline completion
    When the TUI renders
    Then a summary line shows total duration and result
