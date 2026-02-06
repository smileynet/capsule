Feature: Automatic TTY detection with TUI/plain text switching
  As a capsule user, I want automatic detection of whether stdout is
  a TTY so that I get the rich TUI in terminals and clean plain text
  output in CI or piped contexts.

  Scenario: TTY stdout launches Bubble Tea TUI
    Given stdout is a TTY
    When capsule run executes
    Then Bubble Tea TUI is shown

  Scenario: Piped stdout prints plain text
    Given stdout is piped
    When capsule run executes
    Then plain text lines are printed

  Scenario: No-TUI flag forces plain text in TTY
    Given --no-tui flag
    When capsule run in TTY executes
    Then plain text is forced

  Scenario: Display interface routes to correct renderer
    Given StatusCallback
    When Display interface receives updates
    Then correct renderer handles them
