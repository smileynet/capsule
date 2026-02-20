Feature: Preserve expansion state across operations
  As a dashboard user
  I want expansion state to persist during refresh and navigation
  So that I don't lose my place in the hierarchy

  Scenario: Expansion state preserved when refreshing bead list
    Given a dashboard with mixed expanded and collapsed nodes
    When I press 'r' to refresh the bead list
    Then all previously expanded nodes should remain expanded
    And all previously collapsed nodes should remain collapsed

  Scenario: Cursor position restored after refresh
    Given a dashboard with cursor on a specific task
    When I press 'r' to refresh
    Then the cursor should return to the same task

  Scenario: Expansion state preserved when returning from pipeline
    Given a dashboard with expanded nodes
    When I launch a pipeline
    And the pipeline completes
    And I return to browse mode
    Then the expansion state should be preserved

  Scenario: Expansion state preserved when returning from campaign
    Given a dashboard with expanded nodes
    When I launch a campaign
    And the campaign completes
    And I return to browse mode
    Then the expansion state should be preserved

  Scenario: Collapse-all action clears expansion state
    Given a dashboard with multiple expanded nodes
    When I press 'c' to collapse all
    Then all nodes should return to default expansion state
    And epics should be expanded
    And features should be collapsed
