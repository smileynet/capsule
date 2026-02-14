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

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	var out []byte
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 'A' || s[j] > 'Z') && (s[j] < 'a' || s[j] > 'z') {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
		} else {
			out = append(out, s[i])
			i++
		}
	}
	return string(out)
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
