package main

import (
	"fmt"

	"github.com/alecthomas/kong"
)

var version = "dev"

// CLI is the top-level command structure for capsule.
type CLI struct {
	Version kong.VersionFlag `help:"Show version." short:"V"`
	Run     RunCmd           `cmd:"" help:"Run a capsule pipeline."`
	Abort   AbortCmd         `cmd:"" help:"Abort a running capsule."`
	Clean   CleanCmd         `cmd:"" help:"Clean up capsule worktree and artifacts."`
}

// RunCmd executes a capsule pipeline for a given bead.
type RunCmd struct {
	BeadID   string `arg:"" help:"Bead ID to run."`
	Provider string `help:"Provider to use for completions." default:"claude"`
	Timeout  int    `help:"Timeout in seconds." default:"300"`
	Debug    bool   `help:"Enable debug output."`
}

// Run executes the run command.
func (r *RunCmd) Run() error {
	return fmt.Errorf("run: not implemented")
}

// AbortCmd aborts a running capsule.
type AbortCmd struct {
	BeadID string `arg:"" help:"Bead ID to abort."`
	Force  bool   `help:"Force abort without confirmation."`
}

// Run executes the abort command.
func (a *AbortCmd) Run() error {
	return fmt.Errorf("abort: not implemented")
}

// CleanCmd cleans up capsule worktree and artifacts.
type CleanCmd struct {
	BeadID string `arg:"" help:"Bead ID to clean."`
}

// Run executes the clean command.
func (c *CleanCmd) Run() error {
	return fmt.Errorf("clean: not implemented")
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli, kong.Vars{"version": version})
	ctx.FatalIfErrorf(ctx.Run())
}
