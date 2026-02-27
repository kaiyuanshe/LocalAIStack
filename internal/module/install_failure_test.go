package module

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInstallRecordsFailureEvent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := Install("definitely-not-existing-module")
	if err == nil {
		t.Fatalf("expected install error for missing module")
	}

	today := time.Now().Format("20060102")
	target := filepath.Join(home, ".localaistack", "failures", today+".jsonl")
	data, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("expected failure log file at %s: %v", target, readErr)
	}

	line := strings.TrimSpace(string(data))
	if !strings.Contains(line, "\"phase\":\"module_install\"") {
		t.Fatalf("expected module_install phase, got: %s", line)
	}
	if !strings.Contains(line, "\"module\":\"definitely-not-existing-module\"") {
		t.Fatalf("expected module name in log line, got: %s", line)
	}
}
