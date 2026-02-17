# Feature: Campaign completed task inspection (cap-fj8.3)
Feature: Campaign completed task inspection
  As a developer
  I want to navigate to completed tasks within a campaign and see their phase reports
  So that I can review what happened during a campaign run

  Background:
    Given a campaign is running in the dashboard
    And at least one task has completed

  Scenario: Navigate cursor to completed task
    Given the campaign has 3 tasks and the first has completed
    When I press "up" to move cursor to the completed task
    Then the cursor should highlight the completed task

  Scenario: Completed task expands its phases on selection
    Given the cursor is on a completed task
    Then the completed task should expand to show its pipeline phases below
    And each phase should show its pass/fail indicator and duration

  Scenario: Right pane shows phase reports for completed task
    Given the cursor is on a completed task with phase reports
    When I navigate to a specific phase within the expanded task
    Then the right pane should show the phase report for that phase

  Scenario: Only one task expanded at a time
    Given the campaign has 2 completed tasks
    And the cursor is on the first completed task showing its phases
    When I move the cursor to the second completed task
    Then the first task should collapse back to a single line
    And the second task should expand to show its phases

  Scenario: Phase results persist through campaign
    Given a task completed with phases: test-write (passed), execute (failed)
    When I navigate to the completed task
    Then the phase list should show test-write as passed and execute as failed
    And the phase reports should contain the original summary and feedback
