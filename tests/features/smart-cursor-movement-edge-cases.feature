Feature: Smart cursor movement and edge cases
  As a dashboard user
  I want cursor movement to feel natural when expanding/collapsing nodes
  So that navigation is intuitive

  Scenario: Expanding a node moves cursor to first child
    Given a dashboard with a collapsed epic containing tasks
    When I expand the epic
    Then the cursor should move to the first child task

  Scenario: Expanding a node with no children keeps cursor on node
    Given a dashboard with an empty epic (no open tasks)
    When I expand the epic
    Then the cursor should remain on the epic
    And a "(no open tasks)" message should be displayed

  Scenario: Collapsing a node keeps cursor on the collapsed node
    Given a dashboard with an expanded epic
    And the cursor is on the epic
    When I collapse the epic
    Then the cursor should remain on the epic

  Scenario: Left arrow on root node is no-op
    Given a dashboard with a root-level epic
    And the cursor is on the epic
    When I press the left arrow key
    Then nothing should happen
    And the cursor should remain on the epic

  Scenario: Right arrow on empty epic shows message
    Given a dashboard with an epic containing no open tasks
    When I expand the epic
    Then "(no open tasks)" should be displayed
    And the child count badge should show [0]

  Scenario: Cursor never lands on invalid index after expand
    Given a dashboard with various nodes
    When I perform expand operations
    Then the cursor should always be within valid bounds

  Scenario: Cursor never lands on invalid index after collapse
    Given a dashboard with expanded nodes
    When I collapse nodes that contain the cursor
    Then the cursor should move to a valid position
    And should not go out of bounds
