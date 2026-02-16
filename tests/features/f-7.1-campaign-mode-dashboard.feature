# Feature: Campaign mode in dashboard TUI (cap-fj8.1)
Feature: Campaign mode in dashboard TUI
  As a developer
  I want the dashboard to automatically run child tasks when I select a feature or epic
  So that I don't have to run each task individually

  Background:
    Given a capsule dashboard is running
    And beads exist with types: task, feature, epic

  Scenario: Selecting a task runs a single pipeline
    Given a task bead "demo-task" is in the ready list
    When I select "demo-task" and press enter
    Then a single pipeline should start in pipeline mode
    And the phase list should show pipeline phases

  Scenario: Selecting a feature discovers and runs child tasks
    Given a feature bead "demo-feature" is in the ready list
    And "demo-feature" has 3 ready child tasks
    When I select "demo-feature" and press enter
    Then the dashboard should switch to campaign mode
    And the campaign view should show a task queue with 3 tasks
    And the first child task's pipeline phases should be running

  Scenario: Selecting an epic discovers and runs child tasks
    Given an epic bead "demo-epic" is in the ready list
    And "demo-epic" has 2 ready child tasks
    When I select "demo-epic" and press enter
    Then the dashboard should switch to campaign mode
    And the campaign view should show a task queue with 2 tasks

  Scenario: Campaign view shows inline phase nesting for running task
    Given a campaign is running for feature "demo-feature"
    And the first task pipeline is in progress
    Then the running task should show its pipeline phases indented below it
    And pending tasks should show as single lines without phases

  Scenario: Completed tasks collapse to single line
    Given a campaign is running for feature "demo-feature"
    And the first task has completed successfully
    Then the completed task should show as a single line with pass indicator and duration
    And the second task should now be running with phases visible

  Scenario: Right pane shows phase report for cursor-selected phase
    Given a campaign is running for feature "demo-feature"
    And the cursor is on a completed phase
    Then the right pane should show the phase report with summary and feedback

  Scenario: User can abort campaign with q
    Given a campaign is running for feature "demo-feature"
    When I press "q"
    Then the campaign should abort
    And the dashboard should show the campaign summary

  Scenario: Campaign summary shows after all tasks complete
    Given a campaign was running for feature "demo-feature"
    And all child tasks have completed
    Then the dashboard should switch to campaign summary mode
    And the summary should show pass/fail counts and total duration

  Scenario: Unknown bead type defaults to single pipeline
    Given a bead "unknown-bead" with empty type is in the ready list
    When I select "unknown-bead" and press enter
    Then a single pipeline should start in pipeline mode
