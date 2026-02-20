Feature: Collapsible tree with expand/collapse state
  As a dashboard user
  I want to expand and collapse epic/feature nodes
  So that I can focus on relevant parts of the hierarchy without visual clutter

  Scenario: Expand a collapsed node to show children
    Given a dashboard with a collapsed epic node
    When I press the right arrow key on the collapsed node
    Then the node should expand
    And its children should be visible in the tree

  Scenario: Collapse an expanded node to hide children
    Given a dashboard with an expanded epic node
    When I press the left arrow key on the expanded node
    Then the node should collapse
    And its children should be hidden from the tree

  Scenario: Visual indicators show expansion state
    Given a dashboard with mixed expanded and collapsed nodes
    Then collapsed nodes should show ▶ indicator
    And expanded nodes should show ▼ indicator
    And leaf nodes should show • indicator

  Scenario: Child count badge shows number of open children
    Given a dashboard with an epic containing 3 open tasks
    Then the epic should display [3] badge
    When I close one of the tasks
    Then the epic should display [2] badge

  Scenario: Default expansion state
    Given a fresh dashboard load
    Then epic nodes should be expanded by default
    And feature nodes should be collapsed by default

  Scenario: Expansion state persists during navigation
    Given I have expanded a feature node
    When I navigate to other nodes
    And return to the feature node
    Then the feature should still be expanded
