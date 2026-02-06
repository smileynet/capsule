Feature: Go project skeleton with Kong CLI and test harness
  As a capsule developer, I want a Go project with Kong CLI parsing
  and a test harness so that the binary can be built, versioned, and
  tested from the start.

  Scenario: Project builds a capsule binary
    Given the project
    When go build ./... runs
    Then a capsule binary is produced

  Scenario: Binary prints version information
    Given the binary
    When capsule version runs
    Then version, commit, and date are printed

  Scenario: Binary prints usage when run without args
    Given the binary
    When capsule run without args runs
    Then usage is printed
    And exit code is non-zero

  Scenario: Makefile test target passes
    Given the Makefile
    When make test runs
    Then all tests pass
