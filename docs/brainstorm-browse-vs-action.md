# Brainstorm: Browse vs Action UX Pattern

## Problem Statement

Users need to:
1. **Browse** into epics/features to see their child tasks
2. **Initiate work** on epics/features (launch sequential task campaigns)
3. Continue browsing freely while work runs in background

Currently, `Enter` launches work immediately, with no way to drill into hierarchy without triggering execution.

## Research: TUI Best Practices

### Common Patterns from File Managers (Ranger, nnn, lf)

**Ranger/Vim-style:**
- `l` or `→` (right arrow): Enter directory / open file
- `h` or `←` (left arrow): Go up to parent
- `Enter`: Execute/open with default application
- `Space`: Select/mark items

**Key insight:** Directional keys for navigation, Enter for action.

### Keyboard Navigation Standards

From accessibility and UX research:
- **Tab/Shift+Tab**: Move between major UI sections
- **Arrow keys**: Navigate within lists/trees
- **Enter**: Primary action (commit, execute, confirm)
- **Space**: Secondary action (select, toggle, preview)
- **Escape**: Cancel, go back

### TUI Application Patterns

**lazygit:**
- Arrow keys: Navigate lists
- Enter: View details / drill in
- Space: Stage/unstage (toggle action)
- Letter keys: Quick actions (c=commit, p=push)

**k9s (Kubernetes TUI):**
- Arrow keys: Navigate
- Enter: Describe resource (view details)
- Letter keys: Actions (d=delete, l=logs, s=shell)

## Current Capsule Implementation

### Existing Behavior
```
Browse Mode:
  ↑/k, ↓/j     - Navigate list
  Enter        - Launch pipeline/campaign (with confirmation)
  Tab          - Switch between left (tree) and right (detail) panes
  r            - Refresh
  q            - Quit
```

### Current Tree Structure
- Hierarchical display (epic → feature → task)
- All nodes visible at once (no collapse/expand)
- Cursor moves through flat list of all nodes
- Right pane shows details of selected node

## Design Options

### Option 1: Directional Navigation (Ranger-style)

**Concept:** Use arrow keys for hierarchy, Enter for action.

```
Browse Mode:
  ↑/k, ↓/j     - Navigate list (same level or across levels)
  →/l          - Expand node / drill into children
  ←/h          - Collapse node / go to parent
  Enter        - Launch work (with confirmation)
  Space        - Toggle selection (for future batch operations)
  Tab          - Switch panes
```

**Pros:**
- Familiar to vim/ranger users
- Clear semantic separation (arrows=navigate, enter=action)
- Enables collapsible tree (reduce visual clutter)
- Natural for hierarchical data

**Cons:**
- Requires implementing collapse/expand state
- More complex tree rendering
- Learning curve for non-vim users

### Option 2: Enter for Browse, Letter Key for Action

**Concept:** Enter drills in, dedicated key launches work.

```
Browse Mode:
  ↑/k, ↓/j     - Navigate list
  Enter        - Drill into epic/feature (filter to children)
  Backspace/u  - Go back up hierarchy
  r            - Run/launch work on selected item (with confirmation)
  Tab          - Switch panes
```

**Pros:**
- Enter feels natural for "go deeper"
- Single letter for action is fast
- No collapse/expand complexity
- Consistent with "r for refresh" pattern

**Cons:**
- Overloading 'r' (refresh vs run)
- Less discoverable (need help text)
- Doesn't leverage spatial metaphor

### Option 3: Modal Approach (View vs Action Mode)

**Concept:** Toggle between browse and action modes.

```
Browse Mode (default):
  ↑/k, ↓/j     - Navigate
  Enter        - View details / drill in
  Tab          - Switch panes
  a            - Enter Action Mode

Action Mode:
  ↑/k, ↓/j     - Navigate
  Enter        - Launch work (with confirmation)
  Esc          - Back to Browse Mode
```

**Pros:**
- Clear mode separation
- Prevents accidental launches
- Familiar to vim users (normal vs insert mode)

**Cons:**
- Mode confusion for non-vim users
- Extra keypress to launch work
- Need mode indicator in UI

### Option 4: Context Menu / Action Panel

**Concept:** Dedicated action key opens menu of available actions.

```
Browse Mode:
  ↑/k, ↓/j     - Navigate
  Enter        - View details (right pane)
  Space        - Open action menu for selected item
  Tab          - Switch panes

Action Menu (when Space pressed):
  r - Run campaign
  v - View details
  c - Close bead
  Esc - Cancel
```

**Pros:**
- Discoverable (menu shows options)
- Extensible (easy to add actions)
- No ambiguity

**Cons:**
- Extra step for common actions
- More UI complexity
- Slower workflow

### Option 5: Hybrid - Right Arrow + Enter for Action

**Concept:** Combine directional navigation with Enter for primary action.

```
Browse Mode:
  ↑/k, ↓/j     - Navigate list
  →/l/Enter    - Drill into epic/feature (show children)
  ←/h          - Go back to parent level
  Shift+Enter  - Launch work (with confirmation)
  Space        - Quick launch (skip confirmation for tasks)
  Tab          - Switch panes
```

**Pros:**
- Intuitive (right=deeper, left=up)
- Enter still works for drilling in
- Shift+Enter is common for "alternate action"
- Flexible (multiple ways to do things)

**Cons:**
- Shift+Enter less discoverable
- Multiple keys for same action (→ vs Enter)

## Recommended Approach: Option 1 (Directional Navigation)

### Rationale

1. **Semantic clarity:** Arrows for navigation, Enter for action
2. **Industry standard:** Matches ranger, lf, nnn (popular TUI file managers)
3. **Enables collapsible trees:** Reduces visual clutter for large projects
4. **Spatial metaphor:** Right=deeper, Left=up is intuitive
5. **Vim-friendly:** Capsule already uses vim keys (j/k), this extends naturally

### Implementation Plan

#### Phase 1: Add Collapse/Expand State
- Add `expanded` bool to tree nodes
- Default: epics/features expanded, show immediate children
- Store expansion state in browse model

#### Phase 2: Update Key Bindings
```go
case "right", "l":
    // If on epic/feature: expand and move cursor to first child
    // If on task: no-op (or show detail in right pane)

case "left", "h":
    // If node is expanded: collapse it
    // If node is collapsed: move cursor to parent

case "enter":
    // Launch work with confirmation
    // For closed beads: no-op
```

#### Phase 3: Update Tree Rendering
- Only render expanded nodes' children
- Add expand/collapse indicators (▶/▼)
- Update help text

#### Phase 4: Preserve State
- Remember expansion state during refresh
- Restore cursor position after operations

### Visual Design

```
Browse Mode                          Detail Pane
┌────────────────────────────────┐  ┌──────────────────────────┐
│ ▼ epic-1  Epic Title       [2] │  │ epic-1  P1  epic         │
│   ▶ feat-1  Feature A      [3] │  │ Epic Title               │
│   ▼ feat-2  Feature B      [1] │  │                          │
│     • task-1  Implement X      │  │ Epic: epic-1 — Epic Desc │
│ ▶ epic-2  Another Epic     [5] │  │                          │
│                                 │  │ Description:             │
│                                 │  │ This epic covers...      │
└────────────────────────────────┘  └──────────────────────────┘

Keys: ↑↓/jk=nav  →l=expand  ←h=collapse  Enter=run  Tab=pane  r=refresh  q=quit
```

**Indicators:**
- `▼` = Expanded (has children, showing them)
- `▶` = Collapsed (has children, hiding them)
- `•` = Leaf node (task, no children)
- `[N]` = Open child count

### Edge Cases

1. **Empty epic/feature:** Show "(no open tasks)" when expanded
2. **All children closed:** Show "(all tasks closed)" when expanded
3. **Cursor on collapsed node:** Enter launches, → expands
4. **Cursor on leaf task:** → is no-op, Enter launches
5. **Background work running:** Allow browsing, show status indicator

## Alternative Considerations

### Why Not Option 2 (Enter for Browse)?

While simpler to implement, it breaks the mental model that Enter = action. Users expect Enter to "do something" not just "show more". This would be confusing when:
- Pressing Enter on a task (does it run or show details?)
- Consistency with confirmation screens (Enter = confirm)

### Why Not Option 3 (Modal)?

Modes add cognitive load. The directional approach achieves the same goal without requiring users to track which mode they're in.

### Why Not Option 4 (Context Menu)?

Too slow for power users. The goal is efficient keyboard-driven workflow, not discoverability through menus.

## Consistency Across Views

### Browse Mode
- `→/l`: Expand node
- `←/h`: Collapse node / go to parent
- `Enter`: Launch work

### Campaign Mode (running)
- `↑/↓/j/k`: Navigate task list
- `Enter`: View task details (if completed/failed)
- `Esc`: Background the campaign, return to browse

### Pipeline Mode (single task running)
- `↑/↓/j/k`: Scroll phase list
- `Tab`: Switch to output pane
- `Esc`: Background the pipeline, return to browse

### Summary Mode (after completion)
- `Enter/Esc/b`: Return to browse
- Cursor auto-snaps to completed item

## Anti-Patterns to Avoid

1. **Overloading Enter:** Don't make Enter do different things based on context without clear visual cues
2. **Hidden actions:** Every action should be in help text
3. **Inconsistent navigation:** Arrow keys should always move cursor, not trigger actions
4. **Mode confusion:** If using modes, show clear indicator
5. **Accidental launches:** Always confirm before starting work (except for explicit "quick launch" key)

## Future Enhancements

1. **Batch operations:** Space to select multiple, Enter to launch all
2. **Filtering:** `/` to search/filter tree
3. **Bookmarks:** `m` to mark, `'` to jump to marked nodes
4. **Quick actions:** Number keys for common actions (1=run, 2=close, etc.)
5. **Tree persistence:** Remember expansion state across sessions

## Open Questions

1. Should we auto-expand when cursor moves to a collapsed node?
2. Should `Enter` on an epic/feature expand it OR launch it?
   - **Recommendation:** Launch (with confirmation). Use `→` to expand.
3. Should we show a preview of children count before expanding?
   - **Recommendation:** Yes, show `[N]` badge.
4. How to handle very deep hierarchies (epic → feature → sub-feature → task)?
   - **Recommendation:** Support arbitrary depth, use indentation.

## Success Metrics

- Users can browse hierarchy without accidentally launching work
- Users can launch work with ≤2 keystrokes (navigate + Enter)
- Help text clearly shows available actions
- No user confusion about what Enter does
- Background work doesn't block browsing

## Next Steps

1. Implement collapse/expand state in browse model
2. Add `→/←/h/l` key handlers
3. Update tree rendering to show expand indicators
4. Update help text
5. Add tests for new navigation patterns
6. Update documentation
