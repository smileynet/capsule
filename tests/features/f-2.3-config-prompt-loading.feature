Feature: Configuration loading and external prompt management
  As a capsule developer, I want configuration loading from YAML and
  external prompt file management so that runtime behavior and prompt
  content are configurable without code changes.

  Scenario: Config loads provider, timeout, and worktree settings
    Given a config.yaml
    When capsule loads config
    Then Runtime.Provider is set
    And Runtime.Timeout is set
    And Worktree.BaseDir is set

  Scenario: Missing config file falls back to sensible defaults
    Given no config file
    When capsule loads config
    Then sensible defaults are used

  Scenario: Prompt loader reads prompt file by name
    Given prompts/ directory
    When prompt loader loads test-writer
    Then prompts/test-writer.md content is returned

  Scenario: Prompt composer interpolates bead info and feedback
    Given prompt context
    When Compose is called
    Then bead info and feedback are interpolated into the prompt
