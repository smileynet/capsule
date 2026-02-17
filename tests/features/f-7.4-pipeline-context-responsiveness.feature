# Feature: Pipeline context and responsiveness improvements (cap-fj8.4)
Feature: Pipeline context and responsiveness improvements
  As a developer
  I want the dashboard to feel responsive and always show me what's running
  So that I don't lose context or get distracted by visual noise

  Background:
    Given a capsule dashboard is running

  Scenario: Pipeline mode shows bead header
    Given I dispatch a pipeline for bead "cap-xyz" titled "Fix auth bug"
    Then the left pane should show "cap-xyz Fix auth bug" as a header line
    And the header should be dimmed/muted style

  Scenario: Campaign mode shows bead header
    Given I dispatch a campaign for feature "cap-abc" titled "User login"
    Then the left pane should show "cap-abc User login" as a header line

  Scenario: Running phase shows elapsed time
    Given a pipeline is running and the "execute" phase started 42 seconds ago
    Then the execute phase should display elapsed time like "(42s)"
    And the elapsed time should update every second

  Scenario: Debounced bead resolution prevents thrash
    Given the browse mode is showing ready beads
    When I rapidly scroll through 5 beads in quick succession
    Then the right pane should only resolve the final bead
    And intermediate beads should not trigger resolution requests

  Scenario: Cache hits bypass debounce
    Given bead "cap-xyz" detail is already cached
    When I scroll to "cap-xyz"
    Then the right pane should show the cached detail immediately
    And no debounce delay should be applied

  Scenario: Sticky cursor after pipeline return
    Given I dispatched a pipeline for bead "cap-xyz"
    And the pipeline has completed
    When I return to browse mode
    Then the cursor should be on "cap-xyz" in the bead list

  Scenario: Post-pipeline status line
    Given a pipeline completed and post-pipeline merge succeeded
    When I return to browse mode
    Then a status line should show "Merged and closed successfully" for 5 seconds
    And after 5 seconds the status line should disappear

  Scenario: Post-pipeline failure status
    Given a pipeline completed but post-pipeline merge failed
    When I return to browse mode
    Then a status line should show the error message for 5 seconds
