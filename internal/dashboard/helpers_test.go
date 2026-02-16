package dashboard

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// containsText is a test alias for strings.Contains.
func containsText(s, sub string) bool {
	return strings.Contains(s, sub)
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
		if s[i] == '\x1b' {
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
					if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' {
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

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// containsPlainText checks if s contains sub after stripping ANSI escapes.
func containsPlainText(s, sub string) bool {
	return strings.Contains(stripANSI(s), sub)
}

// execBatch executes a tea.Cmd, handling both single commands and batch
// commands. It returns all resulting messages. Spinner ticks are skipped
// to avoid infinite recursion.
func execBatch(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			if c != nil {
				result := c()
				// Skip spinner ticks to avoid recursion.
				if _, isTick := result.(spinner.TickMsg); !isTick {
					msgs = append(msgs, result)
				}
			}
		}
		return msgs
	}
	return []tea.Msg{msg}
}
