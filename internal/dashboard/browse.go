package dashboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CursorMarker is the prefix shown on the selected bead row.
const CursorMarker = "▸ "

// browseState manages the bead list, cursor, and loading/error states
// for browse mode's left pane.
type browseState struct {
	beads   []BeadSummary
	cursor  int
	loading bool
	err     error
}

// newBrowseState returns a browseState in the loading state.
func newBrowseState() browseState {
	return browseState{loading: true}
}

// initBrowse returns a tea.Cmd that calls lister.Ready() asynchronously
// and wraps the result in a BeadListMsg.
func initBrowse(lister BeadLister) tea.Cmd {
	return func() tea.Msg {
		beads, err := lister.Ready()
		return BeadListMsg{Beads: beads, Err: err}
	}
}

// Update processes messages for the browse state.
func (bs browseState) Update(msg tea.Msg) (browseState, tea.Cmd) {
	switch msg := msg.(type) {
	case BeadListMsg:
		return bs.handleBeadList(msg), nil

	case tea.KeyMsg:
		if bs.loading {
			return bs, nil
		}
		return bs.handleKey(msg)
	}

	return bs, nil
}

func (bs browseState) handleBeadList(msg BeadListMsg) browseState {
	bs.loading = false
	if msg.Err != nil {
		bs.err = msg.Err
		bs.beads = nil
		return bs
	}
	bs.err = nil
	bs.beads = msg.Beads
	bs.cursor = 0
	return bs
}

func (bs browseState) handleKey(msg tea.KeyMsg) (browseState, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if len(bs.beads) > 0 {
			bs.cursor--
			if bs.cursor < 0 {
				bs.cursor = len(bs.beads) - 1
			}
		}
		return bs, nil

	case "down", "j":
		if len(bs.beads) > 0 {
			bs.cursor++
			if bs.cursor >= len(bs.beads) {
				bs.cursor = 0
			}
		}
		return bs, nil

	case "enter":
		if len(bs.beads) > 0 && bs.cursor < len(bs.beads) {
			selected := bs.beads[bs.cursor]
			return bs, func() tea.Msg {
				return DispatchMsg{BeadID: selected.ID}
			}
		}
		return bs, nil

	case "r":
		bs.loading = true
		bs.err = nil
		return bs, func() tea.Msg { return RefreshBeadsMsg{} }
	}

	return bs, nil
}

// SelectedID returns the bead ID at the current cursor position,
// or "" if the list is empty or still loading.
func (bs browseState) SelectedID() string {
	if len(bs.beads) == 0 || bs.cursor < 0 || bs.cursor >= len(bs.beads) {
		return ""
	}
	return bs.beads[bs.cursor].ID
}

// View renders the browse pane content for the given dimensions.
func (bs browseState) View(width, height int) string {
	if bs.loading {
		return "Loading beads..."
	}

	if bs.err != nil {
		return fmt.Sprintf("Error: %s\n\nPress r to retry", bs.err)
	}

	if len(bs.beads) == 0 {
		return "No ready beads"
	}

	var b strings.Builder
	for i, bead := range bs.beads {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i == bs.cursor {
			b.WriteString(CursorMarker)
		} else {
			b.WriteString("  ")
		}
		b.WriteString(bead.ID)
		b.WriteByte(' ')
		b.WriteString(PriorityBadge(bead.Priority))
		b.WriteByte(' ')
		b.WriteString(bead.Title)
	}
	return b.String()
}
