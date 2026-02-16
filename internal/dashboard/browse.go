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
	beads      []BeadSummary
	cursor     int
	loading    bool
	err        error
	showClosed bool
	readyBeads []BeadSummary // remembered ready beads for toggle-back
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

// closedBeadLimit is the maximum number of closed beads to fetch.
const closedBeadLimit = 50

// initClosedBrowse returns a tea.Cmd that calls lister.Closed() asynchronously
// and wraps the result in a ClosedBeadListMsg.
func initClosedBrowse(lister BeadLister) tea.Cmd {
	return func() tea.Msg {
		beads, err := lister.Closed(closedBeadLimit)
		return ClosedBeadListMsg{Beads: beads, Err: err}
	}
}

// Update processes messages for the browse state.
func (bs browseState) Update(msg tea.Msg) (browseState, tea.Cmd) {
	switch msg := msg.(type) {
	case BeadListMsg:
		return bs.applyBeadList(msg.Beads, msg.Err), nil

	case ClosedBeadListMsg:
		return bs.applyBeadList(msg.Beads, msg.Err), nil

	case tea.KeyMsg:
		if bs.loading {
			return bs, nil
		}
		return bs.handleKey(msg)
	}

	return bs, nil
}

// applyBeadList applies a fetched bead list (or error) to the browse state,
// clearing the loading indicator and resetting the cursor.
func (bs browseState) applyBeadList(beads []BeadSummary, err error) browseState {
	bs.loading = false
	if err != nil {
		bs.err = err
		bs.beads = nil
		return bs
	}
	bs.err = nil
	bs.beads = append([]BeadSummary(nil), beads...)
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
		if len(bs.beads) > 0 && bs.cursor < len(bs.beads) && !bs.showClosed {
			selected := bs.beads[bs.cursor]
			return bs, func() tea.Msg {
				return DispatchMsg{BeadID: selected.ID, BeadType: selected.Type, BeadTitle: selected.Title}
			}
		}
		return bs, nil

	case "r":
		bs.loading = true
		bs.err = nil
		return bs, func() tea.Msg { return RefreshBeadsMsg{} }

	case "h":
		if bs.showClosed {
			// Toggle back to ready: restore saved beads.
			bs.showClosed = false
			bs.beads = append([]BeadSummary(nil), bs.readyBeads...)
			bs.cursor = 0
			bs.err = nil
			return bs, nil
		}
		// Toggle to closed: save ready beads, request fetch.
		bs.showClosed = true
		bs.readyBeads = append([]BeadSummary(nil), bs.beads...)
		bs.loading = true
		bs.err = nil
		return bs, func() tea.Msg { return ToggleHistoryMsg{} }
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
// spinnerView is the current spinner frame (may be empty when spinner is inactive).
func (bs browseState) View(width, height int, spinnerView string) string {
	if bs.loading {
		return fmt.Sprintf("%s Loading beads...", spinnerView)
	}

	if bs.err != nil {
		return fmt.Sprintf("Error: %s\n\nPress r to retry", bs.err)
	}

	if len(bs.beads) == 0 {
		if bs.showClosed {
			return "No closed beads"
		}
		return "No ready beads — press r to refresh"
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

		if bs.showClosed {
			line := fmt.Sprintf("%s P%d %s", bead.ID, bead.Priority, bead.Title)
			if bead.Type != "" {
				line += " [" + bead.Type + "]"
			}
			b.WriteString(mutedText.Render(line))
		} else {
			line := bead.ID + " " + PriorityBadge(bead.Priority) + " " + bead.Title
			if bead.Type != "" {
				line += " [" + bead.Type + "]"
			}
			b.WriteString(line)
		}
	}
	return b.String()
}
