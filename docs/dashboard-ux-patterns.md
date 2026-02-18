# Dashboard UX Patterns

Best practices and antipatterns for the dashboard's confirmation and action transparency.

## When to Confirm

Reserve confirmation for actions with serious, hard-to-reverse consequences.

| Confirm | Don't Confirm |
|---------|---------------|
| Pipeline dispatch (creates worktree, runs AI, modifies files) | Post-pipeline lifecycle (merge, close, cleanup) — expected continuation |
| Campaign dispatch (N sequential pipelines) | Navigation (cursor, pane focus, scroll) |
| | Browse refresh, detail resolve |

Source: [NN/g — Confirmation Dialogs](https://www.nngroup.com/articles/confirmation-dialog/)

## Confirmation Design

| Practice | Rationale |
|----------|-----------|
| Be specific: show bead ID, title, task count, consequences | Vague prompts cause auto-confirm behavior |
| Use action-specific labels ("Run 4 tasks", not "Yes") | Forces conscious decision-making |
| Default focus on safe option (Esc/Cancel) | Prevents accidental confirmation |
| Reserve for significant/irreversible actions only | Overuse causes habituation |
| Progressive disclosure: summary in help bar, detail in confirm | Reduces overwhelm while preserving discoverability |
| Preview/dry-run: show what will execute without running it | User retains control, can inspect before committing |
| Explain consequences of continuation, don't block on them | "Next: merge to main" — inform, don't require approval |

Sources:
- [UX Planet — Confirmation Dialogs](https://uxplanet.org/confirmation-dialogs-how-to-design-dialogues-without-irritation-7b4cf2599956)
- [IxDF — Progressive Disclosure](https://www.interaction-design.org/literature/topics/progressive-disclosure)
- [IT'S FOSS — Dry-Run Flag](https://itsfoss.gitlab.io/post/how-to-use-dry-run-flag-in-linux-commands-avoid-mistakes-before-they-happen/)

## Antipatterns

| Antipattern | Problem | Alternative |
|-------------|---------|-------------|
| "Are you sure?" with no detail | Users auto-confirm without reading | Show specific bead ID, title, task count |
| Generic Yes/No buttons | Enables mindless clicking | Use "Run 4 tasks" / "Cancel" |
| Confirming every action | Dialog fatigue | Only confirm pipeline/campaign dispatch |
| Hidden consequences (silent post-pipeline) | Users surprised by merge/close/cleanup | Explain "Next:" actions in summary view |
| Default on dangerous action (Enter=confirm) | Accidental confirmation from muscle memory | Esc is the default escape; Enter requires deliberate press |
| Modal overuse for reversible actions | Blocks workflow | Use undo pattern or explain-and-continue |

Sources:
- [NN/g — Confirmation Dialogs](https://www.nngroup.com/articles/confirmation-dialog/)
- [Smashing Magazine — Dangerous Actions](https://www.smashingmagazine.com/2024/09/how-manage-dangerous-actions-user-interfaces/)

## TUI-Specific Design

- **Mode transitions, not modal overlays**: `ModeConfirm` is idiomatic Bubble Tea.
- **Help bar as progressive disclosure**: Show "run pipeline" or "run campaign (N tasks)" while browsing; full detail only in confirm mode.
- **Keyboard consistency**: Enter=action, Esc=back, q=quit across all modes.
- **Key swallowing**: In confirmation mode, swallow unrelated keys (navigation, refresh) to prevent accidental actions.
- **Explain-don't-block**: After pipeline summary, show "Next: merge to main, close bead, cleanup worktree" rather than a second confirmation dialog.
