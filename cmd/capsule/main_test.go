package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/alecthomas/kong"
)

// errExitCalled is a sentinel used to catch kong's os.Exit calls in tests.
var errExitCalled = errors.New("exit called")

func TestVersionFlag(t *testing.T) {
	var cli CLI
	var buf bytes.Buffer
	k, err := kong.New(&cli,
		kong.Vars{"version": "test-version"},
		kong.Writers(&buf, &buf),
		kong.Exit(func(int) { panic(errExitCalled) }),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic from --version flag")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, errExitCalled) {
			panic(r)
		}
		if !bytes.Contains(buf.Bytes(), []byte("test-version")) {
			t.Errorf("version output = %q, want to contain %q", buf.String(), "test-version")
		}
	}()

	k.Parse([]string{"--version"}) //nolint:errcheck // --version triggers panic via Exit hook
}

func TestRunCommand(t *testing.T) {
	var cli CLI
	k, err := kong.New(&cli, kong.Vars{"version": "test"})
	if err != nil {
		t.Fatal(err)
	}
	kctx, err := k.Parse([]string{"run", "some-bead-id"})
	if err != nil {
		t.Fatal(err)
	}
	if kctx.Command() != "run <bead-id>" {
		t.Errorf("got command %q, want %q", kctx.Command(), "run <bead-id>")
	}
	if cli.Run.BeadID != "some-bead-id" {
		t.Errorf("got bead-id %q, want %q", cli.Run.BeadID, "some-bead-id")
	}
}

func TestRunCommandFlags(t *testing.T) {
	var cli CLI
	k, err := kong.New(&cli, kong.Vars{"version": "test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = k.Parse([]string{
		"run", "bead-123",
		"--provider", "claude",
		"--timeout", "120",
		"--debug",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cli.Run.Provider != "claude" {
		t.Errorf("provider = %q, want %q", cli.Run.Provider, "claude")
	}
	if cli.Run.Timeout != 120 {
		t.Errorf("timeout = %d, want %d", cli.Run.Timeout, 120)
	}
	if !cli.Run.Debug {
		t.Error("debug = false, want true")
	}
}

func TestRunCommandDefaults(t *testing.T) {
	var cli CLI
	k, err := kong.New(&cli, kong.Vars{"version": "test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = k.Parse([]string{"run", "bead-456"})
	if err != nil {
		t.Fatal(err)
	}
	if cli.Run.Provider != "claude" {
		t.Errorf("default provider = %q, want %q", cli.Run.Provider, "claude")
	}
	if cli.Run.Timeout != 300 {
		t.Errorf("default timeout = %d, want %d", cli.Run.Timeout, 300)
	}
	if cli.Run.Debug {
		t.Error("default debug = true, want false")
	}
}

func TestAbortCommand(t *testing.T) {
	var cli CLI
	k, err := kong.New(&cli, kong.Vars{"version": "test"})
	if err != nil {
		t.Fatal(err)
	}
	kctx, err := k.Parse([]string{"abort", "bead-789"})
	if err != nil {
		t.Fatal(err)
	}
	if kctx.Command() != "abort <bead-id>" {
		t.Errorf("got command %q, want %q", kctx.Command(), "abort <bead-id>")
	}
	if cli.Abort.BeadID != "bead-789" {
		t.Errorf("got bead-id %q, want %q", cli.Abort.BeadID, "bead-789")
	}
}

func TestAbortForceFlag(t *testing.T) {
	var cli CLI
	k, err := kong.New(&cli, kong.Vars{"version": "test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = k.Parse([]string{"abort", "bead-789", "--force"})
	if err != nil {
		t.Fatal(err)
	}
	if !cli.Abort.Force {
		t.Error("force = false, want true")
	}
}

func TestCleanCommand(t *testing.T) {
	var cli CLI
	k, err := kong.New(&cli, kong.Vars{"version": "test"})
	if err != nil {
		t.Fatal(err)
	}
	kctx, err := k.Parse([]string{"clean", "bead-abc"})
	if err != nil {
		t.Fatal(err)
	}
	if kctx.Command() != "clean <bead-id>" {
		t.Errorf("got command %q, want %q", kctx.Command(), "clean <bead-id>")
	}
	if cli.Clean.BeadID != "bead-abc" {
		t.Errorf("got bead-id %q, want %q", cli.Clean.BeadID, "bead-abc")
	}
}

func TestRunMissingBeadIDErrors(t *testing.T) {
	var cli CLI
	k, err := kong.New(&cli, kong.Vars{"version": "test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = k.Parse([]string{"run"})
	if err == nil {
		t.Fatal("expected error when bead-id missing")
	}
}

func TestNoArgsShowsHelp(t *testing.T) {
	var cli CLI
	k, err := kong.New(&cli, kong.Vars{"version": "test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = k.Parse([]string{})
	if err == nil {
		t.Fatal("expected error when no command provided")
	}
}
