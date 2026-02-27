package failure

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	t.Run("status 403", func(t *testing.T) {
		out := Classify(errors.New("siliconflow request failed with status 403"))
		if out.Category != CategoryAuth || out.StatusCode != 403 || out.Retryable {
			t.Fatalf("unexpected classification: %+v", out)
		}
	})

	t.Run("status 429", func(t *testing.T) {
		out := Classify(errors.New("siliconflow request failed with status 429"))
		if out.Category != CategoryRateLimit || out.StatusCode != 429 || !out.Retryable {
			t.Fatalf("unexpected classification: %+v", out)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		out := Classify(errors.New("context deadline exceeded"))
		if out.Category != CategoryTimeout || !out.Retryable {
			t.Fatalf("unexpected classification: %+v", out)
		}
	})

	t.Run("exit status", func(t *testing.T) {
		out := Classify(errors.New("install step service failed: exit status 3"))
		if out.Category != CategoryCommandExit || out.ExitCode != 3 || out.Retryable {
			t.Fatalf("unexpected classification: %+v", out)
		}
	})

	t.Run("invalid output", func(t *testing.T) {
		out := Classify(errors.New("llm plan response did not include json"))
		if out.Category != CategoryInvalidOutput || out.Retryable {
			t.Fatalf("unexpected classification: %+v", out)
		}
	})
}

func TestRecorderRecord(t *testing.T) {
	baseDir := t.TempDir()
	recorder, err := NewRecorder(baseDir)
	if err != nil {
		t.Fatalf("NewRecorder returned error: %v", err)
	}

	fixed := time.Date(2026, 2, 27, 18, 0, 0, 123, time.UTC)
	recorder.now = func() time.Time { return fixed }

	event := Event{
		Phase:   PhaseInstallPlanner,
		Module:  "p2-smoke",
		Model:   "deepseek-ai/DeepSeek-V3.2",
		Message: "planner failed and fallback used",
		Error:   "siliconflow request failed with status 403",
		Context: map[string]any{
			"strict": true,
		},
	}
	path, err := recorder.Record(event)
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	expectedPath := filepath.Join(baseDir, "20260227.jsonl")
	if path != expectedPath {
		t.Fatalf("unexpected record path: %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("record file not found: %v", err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open record file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("expected at least one record line")
	}
	line := scanner.Text()
	if strings.TrimSpace(line) == "" {
		t.Fatalf("record line is empty")
	}

	var saved Event
	if err := json.Unmarshal([]byte(line), &saved); err != nil {
		t.Fatalf("unmarshal line: %v", err)
	}
	if saved.Phase != PhaseInstallPlanner {
		t.Fatalf("unexpected phase: %s", saved.Phase)
	}
	if saved.Classification.Category != CategoryAuth {
		t.Fatalf("unexpected classification: %+v", saved.Classification)
	}
	if saved.ID == "" || saved.Timestamp == "" {
		t.Fatalf("id/timestamp should be generated: %+v", saved)
	}
}

func TestRecorderRecordValidation(t *testing.T) {
	recorder, err := NewRecorder(t.TempDir())
	if err != nil {
		t.Fatalf("NewRecorder returned error: %v", err)
	}

	_, err = recorder.Record(Event{Phase: PhaseSmartRun})
	if err == nil || !strings.Contains(err.Error(), "event error is required") {
		t.Fatalf("expected validation error, got: %v", err)
	}

	_, err = recorder.Record(Event{Error: "sample error"})
	if err == nil || !strings.Contains(err.Error(), "event phase is required") {
		t.Fatalf("expected validation error, got: %v", err)
	}
}

func TestRecordBestEffortDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	RecordBestEffort(Event{
		Phase: PhaseSmartRun,
		Model: "demo",
		Error: "context deadline exceeded",
	})

	today := time.Now().Format("20060102")
	target := filepath.Join(home, ".localaistack", "failures", today+".jsonl")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected default record file at %s: %v", target, err)
	}
	if !strings.Contains(string(data), "\"phase\":\"smart_run\"") {
		t.Fatalf("expected phase smart_run in record, got: %s", string(data))
	}
}

func TestBuildAdvice(t *testing.T) {
	rateLimit := BuildAdvice(Classification{Category: CategoryRateLimit})
	if !rateLimit.Retryable || len(rateLimit.RetryDelays) == 0 {
		t.Fatalf("expected retryable rate_limit advice, got: %+v", rateLimit)
	}

	auth := BuildAdvice(Classification{Category: CategoryAuth})
	if auth.Retryable {
		t.Fatalf("expected non-retryable auth advice by default, got: %+v", auth)
	}
	if !strings.Contains(strings.ToLower(auth.Suggestion), "api key") {
		t.Fatalf("expected api key hint for auth advice, got: %+v", auth)
	}
}

func TestRecordWithResultBestEffort(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	classification, advice, path := RecordWithResultBestEffort(Event{
		Phase: PhaseConfigPlanner,
		Error: "siliconflow request failed with status 429",
	})

	if classification.Category != CategoryRateLimit {
		t.Fatalf("expected rate_limit, got: %+v", classification)
	}
	if !advice.Retryable || len(advice.RetryDelays) == 0 {
		t.Fatalf("expected retryable advice, got: %+v", advice)
	}
	if strings.TrimSpace(path) == "" {
		t.Fatalf("expected non-empty path")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("record path should exist: %v", err)
	}
}

func TestFailureDebugEnabled(t *testing.T) {
	t.Setenv(EnvFailureDebug, "1")
	if !FailureDebugEnabled() {
		t.Fatalf("expected debug enabled for value 1")
	}
	t.Setenv(EnvFailureDebug, "true")
	if !FailureDebugEnabled() {
		t.Fatalf("expected debug enabled for value true")
	}
	t.Setenv(EnvFailureDebug, "off")
	if FailureDebugEnabled() {
		t.Fatalf("expected debug disabled for value off")
	}
}
