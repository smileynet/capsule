package dashboard

import "strings"

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
