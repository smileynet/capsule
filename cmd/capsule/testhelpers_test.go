package main

import (
	"strings"
	"testing"
)

func TestExtractBeadID(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantID string
	}{
		{
			name:   "Created issue format",
			input:  "✓ Created issue: proj-abc\n  Title: my task\n  Priority: P2\n  Status: open",
			wantID: "proj-abc",
		},
		{
			name:   "Created issue with multi-hyphen prefix",
			input:  "✓ Created issue: my-long-prefix-1\n  Title: test\n",
			wantID: "my-long-prefix-1",
		},
		{
			name:   "Created issue with alphanumeric hash suffix",
			input:  "✓ Created issue: cap-z3f\n",
			wantID: "cap-z3f",
		},
		{
			name:   "with ANSI escape codes",
			input:  "\033[32m✓\033[0m Created issue: proj-42\n",
			wantID: "proj-42",
		},
		{
			name:   "does not match hyphenated words in other context",
			input:  "error: something went wrong with non-bead output",
			wantID: "",
		},
		{
			name:   "does not match flag-like arguments",
			input:  "usage: bd create --title=foo --type=task",
			wantID: "",
		},
		{
			name:   "warning prefix before Created line",
			input:  "warning: beads.role not configured.\n✓ Created issue: test-bead-regex-1\n  Title: test\n",
			wantID: "test-bead-regex-1",
		},
		{
			name:   "trailing punctuation stripped",
			input:  "✓ Created issue: proj-abc.\n",
			wantID: "proj-abc",
		},
		{
			name:   "empty input",
			input:  "",
			wantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBeadID(tt.input)
			if got != tt.wantID {
				t.Errorf("extractBeadID(%q) = %q, want %q", tt.input, got, tt.wantID)
			}
		})
	}
}

// extractBeadID extracts the bead ID from "bd create" output.
// It looks for the "Created issue: <ID>" pattern, which is the canonical
// output format from the bd CLI.
func extractBeadID(output string) string {
	clean := stripANSI(output)
	for _, line := range strings.Split(clean, "\n") {
		// Match the canonical "Created issue: <ID>" format from bd CLI.
		const prefix = "Created issue: "
		if idx := strings.Index(line, prefix); idx != -1 {
			id := strings.TrimSpace(line[idx+len(prefix):])
			id = strings.TrimRight(id, ".:,;")
			if id != "" {
				return id
			}
		}
	}
	return ""
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}
