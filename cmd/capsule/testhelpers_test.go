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

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "plain text", input: "hello world", want: "hello world"},
		{name: "CSI color", input: "\033[31mred\033[0m", want: "red"},
		{name: "CSI cursor move", input: "\033[2;5Htext", want: "text"},
		{name: "OSC window title", input: "\033]0;my title\007rest", want: "rest"},
		{name: "OSC with ST terminator", input: "\033]0;my title\033\\rest", want: "rest"},
		{name: "OSC hyperlink", input: "\033]8;;http://example.com\033\\click\033]8;;\033\\", want: "click"},
		{name: "SS3 sequence", input: "\033OPtext", want: "text"},
		{name: "two-char ESC M", input: "\033Mtext", want: "text"},
		{name: "mixed sequences", input: "\033[1m\033]0;title\007bold\033[0m", want: "bold"},
		{name: "bare ESC at end", input: "text\033", want: "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// stripANSI removes ANSI escape sequences from a string.
// Handles CSI (ESC[), OSC (ESC]), and two-char (ESC+letter) sequences.
func stripANSI(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\033' {
			i++
			if i >= len(s) {
				break // bare ESC at end
			}
			switch s[i] {
			case '[': // CSI: ESC [ params letter
				i++
				for i < len(s) && !isLetter(s[i]) {
					i++
				}
				if i < len(s) {
					i++ // skip final letter
				}
			case ']': // OSC: ESC ] ... (BEL | ESC \)
				i++
				for i < len(s) {
					if s[i] == '\007' {
						i++
						break
					}
					if s[i] == '\033' && i+1 < len(s) && s[i+1] == '\\' {
						i += 2
						break
					}
					i++
				}
			default: // two-char sequence (ESC + letter), e.g. ESC M, ESC O P
				if isLetter(s[i]) {
					i++ // skip the letter
					// SS3 (ESC O) is followed by one more byte
					if i > 1 && s[i-1] == 'O' && i < len(s) {
						i++
					}
				}
			}
			continue
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

// isLetter reports whether b is an ASCII letter.
// Used as the CSI final-byte check. ECMA-48 defines the full final-byte
// range as 0x40-0x7E, but letters cover all sequences seen in practice.
func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
