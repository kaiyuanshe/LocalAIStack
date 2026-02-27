package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/failure"
)

func TestFailureListAndShowCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	failure.RecordBestEffort(failure.Event{
		ID:    "evt-cli-1",
		Phase: failure.PhaseSmartRun,
		Model: "demo",
		Error: "context deadline exceeded",
	})

	root := &cobra.Command{Use: "las"}
	RegisterFailureCommands(root)

	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"failure", "list", "--limit", "5"})
	if err := root.Execute(); err != nil {
		t.Fatalf("list command failed: %v", err)
	}
	if !strings.Contains(buf.String(), "evt-cli-1") {
		t.Fatalf("expected event id in list output, got: %s", buf.String())
	}

	buf.Reset()
	root.SetArgs([]string{"failure", "show", "evt-cli-1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("show command failed: %v", err)
	}
	text := buf.String()
	if !strings.Contains(text, "\"event\"") || !strings.Contains(text, "\"advice\"") {
		t.Fatalf("expected event/advice in show output, got: %s", text)
	}

	today := filepath.Join(home, ".localaistack", "failures")
	if _, err := os.Stat(today); err != nil {
		t.Fatalf("expected failure dir: %v", err)
	}
}
