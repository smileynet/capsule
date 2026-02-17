# Feature: History view with archived pipeline results (cap-fj8.2)
Feature: History view with archived pipeline results
  As a developer
  I want to browse completed beads and see their pipeline results in the dashboard
  So that I can review past work without leaving the TUI

  Background:
    Given a capsule dashboard is running in browse mode
    And closed beads exist with archived pipeline results

  Scenario: Toggle to history view with h key
    Given the dashboard shows ready beads
    When I press "h"
    Then the bead list should show closed beads instead of ready beads
    And closed beads should be displayed with muted styling and "[closed]" tag

  Scenario: Return to ready view with h key
    Given the dashboard shows closed beads (history view)
    When I press "h"
    Then the bead list should show ready beads again

  Scenario: View archived detail for closed bead
    Given the dashboard shows closed beads
    When I select a closed bead "completed-task"
    Then the right pane should show the bead detail
    And below the detail should be the archived summary
    And below the summary should be a worklog excerpt

  Scenario: Most recent closed beads shown first
    Given there are 60 closed beads
    When I press "h" to toggle history view
    Then at most 50 closed beads should be shown
    And they should be ordered most recent first

  Scenario: Help bar updates for history mode
    Given the dashboard shows ready beads
    Then the help bar should show "h: history"
    When I press "h"
    Then the help bar should show "h: ready"
