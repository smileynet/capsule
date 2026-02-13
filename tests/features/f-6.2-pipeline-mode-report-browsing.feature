Feature: Pipeline mode with phase list and report browsing
  As a user
  I want to dispatch a pipeline and see phase reports in the right pane
  So that I have clear visibility into what each phase produced

  Background:
    Given a terminal environment with TTY
    And the dashboard is showing a bead list

  Scenario: Dispatching a pipeline shows phase list
    Given I have selected a bead in the list
    When I press Enter
    Then the left pane switches to a phase list with status indicators
    And pending phases show a dot indicator
    And the running phase shows a spinner indicator

  Scenario: Completed phase reports appear in right pane
    Given a pipeline is running in the dashboard
    When a phase completes with PASS status
    Then a checkmark appears next to the phase in the left pane
    And selecting that phase shows its report in the right pane
    And the report includes summary, files changed, and duration

  Scenario: Auto-follow tracks the running phase
    Given a pipeline is running with multiple phases
    When a new phase starts running
    Then the cursor automatically moves to the running phase
    And the right pane shows the running phase status

  Scenario: Manual cursor disables auto-follow
    Given a pipeline is running with auto-follow active
    When I press the up arrow key
    Then auto-follow is disabled
    And I can browse completed phase reports freely
    And the right pane shows the report for the cursor-selected phase

  Scenario: Pipeline completion shows overall summary
    Given a pipeline is running in the dashboard
    When all phases complete successfully
    Then the right pane shows overall result summary with pass count and duration

  Scenario: Abort during pipeline
    Given a pipeline is running in the dashboard
    When I press Ctrl+C
    Then the pipeline aborts gracefully
    And cleanup runs before returning to browse mode
