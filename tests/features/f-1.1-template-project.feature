Feature: Reusable template project with bead fixtures
  As a capsule developer, I want a template project with pre-built
  beads so that I can repeatedly test the pipeline against a known
  starting state.

  Scenario: Setup script creates initialized git repo
    Given a clean directory
    When setup-template.sh runs
    Then a git repo exists with source files
    And AGENTS.md is present
    And .beads/ is initialized

  Scenario: Template repo has available task beads
    Given the template repo
    When bd ready runs
    Then at least one task bead is available to work

  Scenario: Task bead contains full metadata
    Given the template beads
    When bd show <task-id> runs
    Then the bead has title, description, acceptance criteria, and parent feature/epic

  Scenario: Setup script produces deterministic state
    Given multiple test runs
    When setup-template.sh runs each time
    Then the starting state is identical
