Feature: TUI keyboard controls for abort and status
  As a capsule user, I want keyboard controls in the TUI so that I
  can abort a running pipeline gracefully or toggle detailed status
  views during execution.

  Scenario: Pressing q or Ctrl+C aborts pipeline gracefully
    Given a running pipeline in TUI
    When I press 'q' or Ctrl+C
    Then the pipeline aborts gracefully with cleanup

  Scenario: Pressing d toggles detailed status view
    Given a running pipeline in TUI
    When I press 'd'
    Then I see detailed status including current phase output
    And I see worklog tail

  Scenario: Abort shows summary before exiting
    Given an abort
    When cleanup completes
    Then TUI shows abort summary before exiting
