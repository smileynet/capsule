# Feature: Campaign error observability (cap-9f0.3)
Feature: Campaign error observability
  As a developer running a campaign
  I want to see why tasks failed and be alerted to infrastructure issues
  So that I can diagnose problems without checking log files manually

  Background:
    Given a campaign is running

  Scenario: State save failures are logged to stderr
    Given a campaign state save fails with an I/O error
    When the campaign continues execution
    Then the error should be logged to stderr with the save path and error detail
    And the campaign should not abort due to the save failure

  Scenario: Failed tasks include error detail text in the dashboard
    Given a task "task-2" has failed during the execute phase
    When the campaign view renders the task list
    Then "task-2" should show a fail indicator
    And the task row should include the error detail text

  Scenario: Selecting a failed task shows error and reviewer feedback in right pane
    Given a task "task-2" has failed with reviewer feedback
    When I navigate the cursor to "task-2"
    Then the right pane should show the error message
    And the right pane should show the reviewer feedback from the failed phase

  Scenario: Bead close failures are logged as warnings
    Given a task has completed and its bead close fails
    When the campaign advances to the next task
    Then a warning should be logged to stderr with the bead ID and error
    And the campaign should continue without aborting

  Scenario: CLI campaign output includes error detail for failed tasks
    Given the campaign is running in CLI plain text mode
    And a task "task-3" has failed during the sign-off phase
    When the campaign prints the task result
    Then the output should include the task name, failed phase, and error detail
