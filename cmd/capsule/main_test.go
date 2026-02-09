package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/alecthomas/kong"
)

// errExitCalled is a sentinel used to catch kong's os.Exit calls in tests.
var errExitCalled = errors.New("exit called")

func TestFeature_GoProjectSkeleton(t *testing.T) {
	t.Run("version flag prints version commit and date", func(t *testing.T) {
		// Given: a CLI parser with version, commit, and date fields
		var cli CLI
		var buf bytes.Buffer
		versionStr := "v1.0.0 abc1234 2026-01-01T00:00:00Z"
		k, err := kong.New(&cli,
			kong.Vars{"version": versionStr},
			kong.Writers(&buf, &buf),
			kong.Exit(func(int) { panic(errExitCalled) }),
		)
		if err != nil {
			t.Fatal(err)
		}

		// When: --version flag is passed
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic from --version flag")
			}
			err, ok := r.(error)
			if !ok || !errors.Is(err, errExitCalled) {
				panic(r)
			}

			// Then: version, commit, and date are all present in output
			output := buf.String()
			for _, want := range []string{"v1.0.0", "abc1234", "2026-01-01T00:00:00Z"} {
				if !bytes.Contains(buf.Bytes(), []byte(want)) {
					t.Errorf("version output = %q, want to contain %q", output, want)
				}
			}
		}()

		k.Parse([]string{"--version"}) //nolint:errcheck // --version triggers panic via Exit hook
	})

	t.Run("no args shows usage and errors", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: no arguments are provided
		_, err = k.Parse([]string{})

		// Then: an error is returned (usage printed)
		if err == nil {
			t.Fatal("expected error when no command provided")
		}
	})

	t.Run("run command parses bead ID", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: run command is invoked with a bead ID
		kctx, err := k.Parse([]string{"run", "some-bead-id"})
		if err != nil {
			t.Fatal(err)
		}

		// Then: the command and bead ID are parsed correctly
		if kctx.Command() != "run <bead-id>" {
			t.Errorf("got command %q, want %q", kctx.Command(), "run <bead-id>")
		}
		if cli.Run.BeadID != "some-bead-id" {
			t.Errorf("got bead-id %q, want %q", cli.Run.BeadID, "some-bead-id")
		}
	})

	t.Run("run command accepts flags", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: run command is invoked with all flags
		_, err = k.Parse([]string{
			"run", "bead-123",
			"--provider", "claude",
			"--timeout", "120",
			"--debug",
		})
		if err != nil {
			t.Fatal(err)
		}

		// Then: all flags are parsed correctly
		if cli.Run.Provider != "claude" {
			t.Errorf("provider = %q, want %q", cli.Run.Provider, "claude")
		}
		if cli.Run.Timeout != 120 {
			t.Errorf("timeout = %d, want %d", cli.Run.Timeout, 120)
		}
		if !cli.Run.Debug {
			t.Error("debug = false, want true")
		}
	})

	t.Run("run command has sensible defaults", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: run command is invoked with no optional flags
		_, err = k.Parse([]string{"run", "bead-456"})
		if err != nil {
			t.Fatal(err)
		}

		// Then: defaults are applied
		if cli.Run.Provider != "claude" {
			t.Errorf("default provider = %q, want %q", cli.Run.Provider, "claude")
		}
		if cli.Run.Timeout != 300 {
			t.Errorf("default timeout = %d, want %d", cli.Run.Timeout, 300)
		}
		if cli.Run.Debug {
			t.Error("default debug = true, want false")
		}
	})

	t.Run("run command requires bead ID", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: run command is invoked without a bead ID
		_, err = k.Parse([]string{"run"})

		// Then: an error is returned
		if err == nil {
			t.Fatal("expected error when bead-id missing")
		}
	})

	t.Run("abort command parses bead ID", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: abort command is invoked with a bead ID
		kctx, err := k.Parse([]string{"abort", "bead-789"})
		if err != nil {
			t.Fatal(err)
		}

		// Then: the command and bead ID are parsed correctly
		if kctx.Command() != "abort <bead-id>" {
			t.Errorf("got command %q, want %q", kctx.Command(), "abort <bead-id>")
		}
		if cli.Abort.BeadID != "bead-789" {
			t.Errorf("got bead-id %q, want %q", cli.Abort.BeadID, "bead-789")
		}
	})

	t.Run("abort command accepts force flag", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: abort command is invoked with --force
		_, err = k.Parse([]string{"abort", "bead-789", "--force"})
		if err != nil {
			t.Fatal(err)
		}

		// Then: force flag is set
		if !cli.Abort.Force {
			t.Error("force = false, want true")
		}
	})

	t.Run("clean command parses bead ID", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: clean command is invoked with a bead ID
		kctx, err := k.Parse([]string{"clean", "bead-abc"})
		if err != nil {
			t.Fatal(err)
		}

		// Then: the command and bead ID are parsed correctly
		if kctx.Command() != "clean <bead-id>" {
			t.Errorf("got command %q, want %q", kctx.Command(), "clean <bead-id>")
		}
		if cli.Clean.BeadID != "bead-abc" {
			t.Errorf("got bead-id %q, want %q", cli.Clean.BeadID, "bead-abc")
		}
	})
}
