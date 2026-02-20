Feature: Directional keyboard navigation (→←hl)
  As a dashboard user
  I want to use arrow keys to expand/collapse nodes
  So that I can navigate the hierarchy without accidentally launching work

  Scenario: Right arrow expands collapsed node and moves to first child
    Given a dashboard with a collapsed epic containing tasks
    And the cursor is on the collapsed epic
    When I press the right arrow key
    Then the epic should expand
    And the cursor should move to the first child task

  Scenario: Right arrow on expanded node moves to first child
    Given a dashboard with an expanded epic
    And the cursor is on the epic
    When I press the right arrow key
    Then the cursor should move to the first child task
    And the epic should remain expanded

  Scenario: Right arrow on leaf node is no-op
    Given a dashboard with a task node (leaf)
    And the cursor is on the task
    When I press the right arrow key
    Then nothing should happen
    And the cursor should remain on the task

  Scenario: Left arrow collapses expanded node
    Given a dashboard with an expanded epic
    And the cursor is on the epic
    When I press the left arrow key
    Then the epic should collapse
    And the cursor should remain on the epic

  Scenario: Left arrow on collapsed node moves to parent
    Given a dashboard with a collapsed feature under an epic
    And the cursor is on the feature
    When I press the left arrow key
    Then the cursor should move to the parent epic

  Scenario: 'l' key works same as right arrow
    Given a dashboard with a collapsed epic
    And the cursor is on the epic
    When I press the 'l' key
    Then the epic should expand
    And the cursor should move to the first child

  Scenario: 'h' key works same as left arrow
    Given a dashboard with an expanded epic
    And the cursor is on the epic
    When I press the 'h' key
    Then the epic should collapse
    And the cursor should remain on the epic

  Scenario: Navigation works consistently across all node types
    Given a dashboard with epics, features, and tasks
    When I use arrow keys to navigate
    Then expand/collapse behavior should be consistent
    And cursor movement should be predictable
