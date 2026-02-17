# TUI Design Reference

Living reference for capsule dashboard TUI design decisions, best practices, and antipatterns.

## 1. Design Decisions

### Unified browse view over history toggle

**Decision:** Show all beads (open + closed) in one tree view. Remove the `h` toggle.

**Rationale:** The `h` toggle hid closed items behind invisible state. Users had no indication that closed beads existed unless they knew to press `h`. This violates the principle that all active filters must be visible. A unified view surfaces everything at once, using visual styling (dim text) to de-emphasize completed work.

### Tree hierarchy with ID prefix convention

**Decision:** Build parent-child trees using ID prefix matching (e.g., `demo-1.1` is a child of `demo-1`). No changes to `bd` CLI or `internal/bead/`.

**Rationale:** The codebase already uses `isChildOf(childID, parentID)` for campaign child discovery. Reusing this convention avoids adding metadata fields to the bead CLI. For projects with < 20 beads, prefix matching produces correct trees without additional plumbing.

### Semantic color palette with ANSI 0-15

**Decision:** Use only ANSI named colors (0-15) plus adaptive light/dark pairs. Avoid extended 256-color values.

**Rationale:** ANSI named colors respect user terminal themes. Extended palette colors (like `208` for orange) render as fixed RGB values that may clash with dark/light theme switches. Named colors let the terminal remap them to theme-appropriate values.

### Symbol-first accessibility

**Decision:** Every state is readable without color. Symbols are paired with color for redundancy.

| State | Symbol | Color |
|-------|--------|-------|
| Pending | `○` | Dim |
| Active/Running | `⧗` | Cyan |
| Passed/Complete | `✓` | Green |
| Failed/Error | `✗` | Red |
| Blocked/Warning | `!` | Yellow |
| Skipped | `–` | Dim |

**Rationale:** ~8% of males have red-green color blindness (Section 508). Color alone is never sufficient to convey meaning.

### Recursive campaign execution with depth limit

**Decision:** Campaign `Run()` recurses into feature/epic children up to depth 3. Each recursion level creates its own `State` keyed by parentID.

**Rationale:** Epic campaigns contain features which contain tasks. Executing only direct children misses the task level. Recursion with depth limit prevents infinite loops while handling the full hierarchy. Checkpoint/resume works at every level because each state is independent.

### Fully-expanded tree for < 20 items

**Decision:** Always show the full tree without collapse/expand controls.

**Rationale:** Collapse/expand adds interaction cost (keypresses, mental model of hidden state) without benefit when the item count is small. For typical capsule projects (< 20 beads), full expansion is optimal. Collapse/expand can be added later if item counts grow.

## 2. TUI Best Practices

### Visual hierarchy

- **Completed items:** Dim/faint text (ANSI dim attribute or gray foreground). This is the universal convention (k9s, taskwarrior, gh CLI).
- **Active items:** Cyan foreground. Draws the eye to what needs attention.
- **Open items:** Default terminal foreground. No special styling needed.
- **Parent nodes:** Show progress counts (`done/total`) for quick scanning.

### Color usage

- Use **4-6 semantic hues maximum** before cognitive overload (Miller's Law applied to color categories).
- ANSI named colors (0-15) respect user themes. Adaptive pairs (`{Light: "1", Dark: "9"}`) handle both light and dark terminals.
- Bold increases contrast for emphasis. Dim reduces contrast for de-emphasis.
- Reserve underline for links or interactive elements.

### Accessibility

- **Always pair color with symbols or text** — a checkmark + green is better than green alone.
- Support `NO_COLOR` environment variable (no-color.org). When set, disable all color output.
- Test with a monochrome terminal to verify information is conveyed without color.
- Box-drawing characters (`├── └── │`) are widely supported in modern terminals.

### Progressive disclosure

- Summary counts on parent nodes let users skip subtrees at a glance.
- Detail view shows full information for the selected item.
- Keybinding hints: show 5-8 most relevant in the help bar. Full list on `?`.

### Tree rendering

Use Unicode box-drawing characters for tree connectors:

```
├── child (non-last)
└── child (last)
│   continuation line
    (no continuation for last-child subtree)
```

## 3. TUI Antipatterns

### Hidden state

Toggle modes that hide data without visible indication. Users don't know what they can't see. **Our fix:** unified view with visual styling instead of separate views.

### Color as sole signal

Red/green without text or symbol backup. Fails for colorblind users and monochrome terminals. **Our fix:** every state has a unique symbol.

### Information overload

Showing all fields for all items simultaneously. Priority badges, type tags, and IDs are useful; full descriptions inline are noise. **Our fix:** summary in left pane, detail in right pane.

### Mode confusion

Unclear which mode is active or what keys are available. **Our fix:** help bar always visible with context-aware bindings.

### Extended ANSI colors

256-color or truecolor values that break on light/dark theme switches. `Color("208")` renders as a fixed orange regardless of theme. **Our fix:** ANSI named colors only.

### Blink attribute

Widely unsupported, universally annoying. Never use it.

## 4. Color Reference

### Semantic roles

| Role | ANSI Light | ANSI Dark | Usage |
|------|-----------|-----------|-------|
| Active | `6` (cyan) | `14` (bright cyan) | In-progress items, running phases |
| Success | `2` (green) | `10` (bright green) | Completed items (symbol only — text is dim) |
| Warning | `3` (yellow) | `11` (bright yellow) | Blocked items, overdue |
| Error | `1` (red) | `9` (bright red) | Failed tasks, errors |
| Dim | `245` | `242` | Completed item text, secondary info |
| Normal | Default | Default | Open items |

### Priority badges

| Priority | Color | Notes |
|----------|-------|-------|
| P0 | Red (1/9) | Critical |
| P1 | Yellow (3/11) | High — changed from extended `208` for theme compliance |
| P2 | Yellow (3/11) | Medium |
| P3 | Blue (4/12) | Low |
| P4 | Gray (240/245) | Backlog |

### Style attributes

- **Bold:** Phase names while running, parent node titles with children.
- **Dim:** Completed item text, secondary metadata, elapsed time.
- **Underline:** Not currently used; reserved for future interactive elements.
- **Strikethrough:** Not used.

### NO_COLOR support

When the `NO_COLOR` environment variable is set, lipgloss automatically disables color output. No additional code needed — lipgloss checks this by default.

## 5. Component Patterns

### Tree node rendering

Each flattened tree node carries a prefix string built during tree traversal:

```go
type flatNode struct {
    Node   *treeNode
    Prefix string  // e.g., "│   ├── " or "    └── "
    Depth  int
}
```

Prefix is composed of ancestor continuation marks (`│   ` or `    `) plus the node's own connector (`├── ` or `└── `).

### Status indicators

Centralized symbol + style pairing avoids duplication across browse, pipeline, and campaign views:

```go
const (
    SymbolPending  = "○"
    SymbolProgress = "⧗"
    SymbolCheck    = "✓"
    SymbolCross    = "✗"
    SymbolSkipped  = "–"
)
```

### Progress counts on parent nodes

Parent nodes display `done/total` where `done` = closed children (recursively) and `total` = all children. When all children are closed, the parent shows `✓` alongside its count.

### Two-pane layout

- Left pane: 1/3 of terminal width (minimum 28 characters).
- Right pane: remaining width.
- Independent focus with tab toggle.
- Focused pane has accent-colored border, unfocused has dim border.

## Sources

- Nielsen Norman Group: Minimize Cognitive Load
- Section 508: Making Color Usage Accessible
- NO_COLOR convention (no-color.org)
- Taskwarrior color themes and precedence rules
- K9s color codes and semantic mapping
- Ratatui community discussion on ANSI named colors
- Charmbracelet Lipgloss and Bubble Tea patterns
- Miller's Law (7±2 for distinct categories)
