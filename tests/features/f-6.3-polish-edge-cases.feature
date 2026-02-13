Feature: Polish, shared lifecycle, and edge case handling
  As a user
  I want the dashboard to handle edge cases gracefully
  So that it is robust and reliable in real use

  Background:
    Given a terminal environment with TTY

  Scenario: Post-pipeline lifecycle runs in background on return
    Given a pipeline has completed successfully in the dashboard
    When I press any key to return to browse
    Then post-pipeline lifecycle runs in the background
    And the bead list refreshes
    And the completed bead no longer appears in the ready list

  Scenario: Terminal resize re-layouts panes
    Given the dashboard is displaying with two panes
    When the terminal window is resized
    Then both panes re-layout proportionally
    And the left pane maintains minimum 28 character width

  Scenario: Abort during pipeline returns to browse
    Given a pipeline is running in the dashboard
    When I press Ctrl+C
    Then cleanup runs gracefully
    And the dashboard returns to browse mode

  Scenario: Double-press force quits
    Given a pipeline abort is in progress
    When I press Ctrl+C again
    Then the dashboard exits immediately

  Scenario: bd not installed shows clear error
    Given the bd CLI is not available
    When I run capsule dashboard
    Then a clear error message is shown

  Scenario: Empty bead list shows refresh hint
    Given no ready beads exist
    When the dashboard loads
    Then a "No ready beads" message is shown with a hint to press r to refresh
