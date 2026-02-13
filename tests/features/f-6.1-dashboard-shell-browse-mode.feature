Feature: Dashboard shell with two-pane layout and bead browsing
  As a user
  I want to run capsule dashboard and see a two-pane layout with bead list and detail
  So that I can browse available work with full context

  Background:
    Given a terminal environment with TTY

  Scenario: Dashboard launches with two-pane layout
    Given ready beads exist in the project
    When I run capsule dashboard
    Then a loading spinner appears
    And a two-pane layout with rounded borders is displayed
    And the left pane shows ready beads with ID, title, priority badge, and type

  Scenario: Cursor navigation updates detail pane
    Given the dashboard is showing a bead list
    When I press the down arrow key
    Then the cursor moves to the next bead
    And the right pane updates with the resolved bead detail
    And the detail shows epic/feature/task hierarchy, description, and acceptance criteria

  Scenario: Tab switches focus between panes
    Given the dashboard is showing a bead list with left pane focused
    When I press Tab
    Then the right pane border color changes to indicate focus
    And arrow keys now scroll the right pane viewport
    When I press Tab again
    Then the left pane border color changes to indicate focus
    And arrow keys now navigate the bead list

  Scenario: Manual refresh reloads bead list
    Given the dashboard is showing a bead list
    When I press 'r'
    Then the bead list refreshes from bd ready
    And the detail cache is invalidated

  Scenario: Quit exits cleanly
    Given the dashboard is showing a bead list
    When I press 'q'
    Then the dashboard exits cleanly

  Scenario: Non-TTY environment shows error
    Given stdout is not a terminal
    When I run capsule dashboard
    Then an error prints saying dashboard requires a terminal
