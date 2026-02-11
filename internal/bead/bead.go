// Package bead resolves bead context from the bd CLI for pipeline input.
package bead

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/smileynet/capsule/internal/worklog"
)

// Sentinel errors for caller-checkable conditions.
var (
	ErrCLINotFound = errors.New("bead: bd CLI not found on PATH")
	ErrNotFound    = errors.New("bead: issue not found")
)

// issue is the JSON structure returned by bd show --json.
type issue struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Acceptance   string       `json:"acceptance_criteria"`
	Status       string       `json:"status"`
	Priority     int          `json:"priority"`
	IssueType    string       `json:"issue_type"`
	Parent       string       `json:"parent"`
	Dependencies []dependency `json:"dependencies"`
}

// dependency is a single dependency entry in the bd JSON output.
type dependency struct {
	IssueID     string `json:"issue_id"`
	DependsOnID string `json:"depends_on_id"`
	Type        string `json:"type"`
}

// Summary is a minimal view of a bead for listing.
type Summary struct {
	ID       string
	Title    string
	Priority int
	Type     string
}

// Client calls the bd CLI to resolve bead context.
type Client struct {
	// Dir is the working directory for bd commands.
	Dir string
}

// NewClient creates a Client that runs bd in the given directory.
func NewClient(dir string) *Client {
	return &Client{Dir: dir}
}

// Resolve fetches bead metadata and walks the parent chain to build
// a full BeadContext for worklog instantiation.
// Returns a context with just TaskID set if bd is unavailable.
func (c *Client) Resolve(id string) (worklog.BeadContext, error) {
	if err := c.checkBD(); err != nil {
		return worklog.BeadContext{TaskID: id}, nil
	}

	task, err := c.show(id)
	if err != nil {
		return worklog.BeadContext{TaskID: id}, nil
	}

	ctx := worklog.BeadContext{
		TaskID:             task.ID,
		TaskTitle:          task.Title,
		TaskDescription:    task.Description,
		AcceptanceCriteria: task.Acceptance,
	}

	// Walk parent chain: task → feature → epic.
	parentID := c.extractParentID(task)
	if parentID == "" {
		return ctx, nil
	}

	parent, err := c.show(parentID)
	if err != nil {
		return ctx, nil
	}

	switch parent.IssueType {
	case "feature":
		ctx.FeatureID = parent.ID
		ctx.FeatureTitle = parent.Title
		ctx.FeatureGoal = parent.Description

		// Look for epic above feature.
		grandparentID := c.extractParentID(parent)
		if grandparentID != "" {
			grandparent, err := c.show(grandparentID)
			if err == nil && grandparent.IssueType == "epic" {
				ctx.EpicID = grandparent.ID
				ctx.EpicTitle = grandparent.Title
				ctx.EpicGoal = grandparent.Description
			}
		}
	case "epic":
		ctx.EpicID = parent.ID
		ctx.EpicTitle = parent.Title
		ctx.EpicGoal = parent.Description
	}

	return ctx, nil
}

// Close marks a bead as closed via bd close.
func (c *Client) Close(id string) error {
	if err := c.checkBD(); err != nil {
		return err
	}

	cmd := exec.Command("bd", "close", id)
	cmd.Dir = c.Dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("bead: closing %s: %w\n%s", id, err, bytes.TrimSpace(out))
	}
	return nil
}

// Ready returns the list of beads with no blockers.
func (c *Client) Ready() ([]Summary, error) {
	if err := c.checkBD(); err != nil {
		return nil, err
	}

	cmd := exec.Command("bd", "ready", "--json")
	cmd.Dir = c.Dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bead: bd ready: %w", err)
	}

	var issues []issue
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&issues); err != nil {
		return nil, fmt.Errorf("bead: parsing ready output: %w", err)
	}

	summaries := make([]Summary, len(issues))
	for i, iss := range issues {
		summaries[i] = Summary{
			ID:       iss.ID,
			Title:    iss.Title,
			Priority: iss.Priority,
			Type:     iss.IssueType,
		}
	}
	return summaries, nil
}

// show fetches a single issue by ID.
func (c *Client) show(id string) (issue, error) {
	cmd := exec.Command("bd", "show", id, "--json")
	cmd.Dir = c.Dir
	out, err := cmd.Output()
	if err != nil {
		return issue{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	// bd show returns an array with one element.
	var issues []issue
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&issues); err != nil {
		return issue{}, fmt.Errorf("bead: parsing show output for %s: %w", id, err)
	}
	if len(issues) == 0 {
		return issue{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return issues[0], nil
}

// extractParentID returns the parent ID from an issue.
// Checks the Parent field first, falls back to scanning dependencies.
func (c *Client) extractParentID(iss issue) string {
	if iss.Parent != "" {
		return iss.Parent
	}
	for _, dep := range iss.Dependencies {
		if dep.Type == "parent-child" && dep.DependsOnID != iss.ID {
			return dep.DependsOnID
		}
	}
	return ""
}

// checkBD verifies that bd is on PATH.
func (c *Client) checkBD() error {
	if _, err := exec.LookPath("bd"); err != nil {
		return ErrCLINotFound
	}
	return nil
}
