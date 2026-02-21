# Feature: Depth-aware campaign callback adapter (cap-9f0.4)
Feature: Depth-aware campaign callback adapter
  As a developer dispatching an epic from the dashboard
  I want to see the epic's features as the top-level campaign view with tasks nested under the running feature
  So that I can track progress across the full epic hierarchy

  Background:
    Given a capsule dashboard is running
    And an epic "demo-epic" has 2 child features with tasks

  Scenario: Epic dispatch shows features as top-level rows
    Given "demo-epic" is dispatched as a campaign
    When the campaign view renders
    Then the features should appear as top-level rows in the task list
    And the features should not be replaced by their child tasks

  Scenario: Running feature shows nested child tasks below feature row
    Given the campaign is running for "demo-epic"
    When "feature-1" starts executing
    Then "feature-1" child tasks should appear nested below the feature row
    And pending features should remain as single rows without expansion

  Scenario: Pipeline phases animate under active task within feature
    Given "feature-1" is running and its first task is active
    When the task's pipeline phases are in progress
    Then the pipeline phases should appear indented below the active task
    And the phase spinner and elapsed time should animate

  Scenario: Completed feature collapses nested tasks and shows checkmark with duration
    Given "feature-1" has completed all its child tasks
    When the campaign view updates
    Then "feature-1" nested tasks should collapse to a single line
    And "feature-1" should show a pass indicator and total duration

  Scenario: Background mode works correctly with nested campaign messages
    Given the campaign is running for "demo-epic" with nested tasks visible
    When I press "Esc" to enter background mode
    And "feature-1" completes while in background mode
    Then the campaign state should update correctly in the background
    And returning to the campaign view should show the updated state

  Scenario: CLI adapter logs subcampaign events with indented formatting
    Given the campaign is running in CLI plain text mode for "demo-epic"
    When "feature-1" starts and discovers 3 child tasks
    Then the CLI output should log subcampaign start with feature name
    And child task progress should be logged with indented formatting
    And feature completion should be logged with duration
