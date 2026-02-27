package failure

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const EnvFailureDebug = "LOCALAISTACK_FAILURE_DEBUG"

const (
	PhaseInstallPlanner = "install_planner"
	PhaseConfigPlanner  = "config_planner"
	PhaseSmartRun       = "smart_run"
	PhaseModuleInstall  = "module_install"
	PhaseModelRun       = "model_run"
)

const (
	CategoryAuth          = "auth"
	CategoryRateLimit     = "rate_limit"
	CategoryProvider      = "provider_unavailable"
	CategoryTimeout       = "timeout"
	CategoryNetwork       = "network"
	CategoryCommandExit   = "command_exit"
	CategoryInvalidOutput = "invalid_output"
	CategoryNotFound      = "not_found"
	CategoryUnknown       = "unknown"
)

type Classification struct {
	Category   string `json:"category"`
	Retryable  bool   `json:"retryable"`
	StatusCode int    `json:"status_code,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type Event struct {
	ID             string         `json:"id"`
	Timestamp      string         `json:"timestamp"`
	Phase          string         `json:"phase"`
	Module         string         `json:"module,omitempty"`
	Model          string         `json:"model,omitempty"`
	Provider       string         `json:"provider,omitempty"`
	Message        string         `json:"message,omitempty"`
	Error          string         `json:"error"`
	Classification Classification `json:"classification"`
	Context        map[string]any `json:"context,omitempty"`
}

type Recorder struct {
	baseDir string
	now     func() time.Time
}

type Advice struct {
	Retryable   bool  `json:"retryable"`
	RetryDelays []int `json:"retry_delays,omitempty"`
	Suggestion  string `json:"suggestion"`
}

func NewRecorder(baseDir string) (*Recorder, error) {
	dir := strings.TrimSpace(baseDir)
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, ".localaistack", "failures")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Recorder{
		baseDir: dir,
		now:     time.Now,
	}, nil
}

func RecordBestEffort(event Event) {
	recorder, err := NewRecorder("")
	if err != nil {
		return
	}
	_, _ = recorder.Record(event)
}

func RecordWithResultBestEffort(event Event) (Classification, Advice, string) {
	classification := event.Classification
	if strings.TrimSpace(classification.Category) == "" {
		classification = Classify(errors.New(event.Error))
		event.Classification = classification
	}
	advice := BuildAdvice(classification)
	recorder, err := NewRecorder("")
	if err != nil {
		return classification, advice, ""
	}
	path, err := recorder.Record(event)
	if err != nil {
		return classification, advice, ""
	}
	return classification, advice, path
}

func (r *Recorder) Record(event Event) (string, error) {
	if r == nil {
		return "", errors.New("recorder is nil")
	}
	if strings.TrimSpace(event.Error) == "" {
		return "", errors.New("event error is required")
	}
	if strings.TrimSpace(event.Phase) == "" {
		return "", errors.New("event phase is required")
	}
	if strings.TrimSpace(event.ID) == "" {
		event.ID = buildEventID(r.now())
	}
	if strings.TrimSpace(event.Timestamp) == "" {
		event.Timestamp = r.now().Format(time.RFC3339Nano)
	}
	if strings.TrimSpace(event.Classification.Category) == "" {
		event.Classification = Classify(errors.New(event.Error))
	}

	target := filepath.Join(r.baseDir, fmt.Sprintf("%s.jsonl", r.now().Format("20060102")))
	file, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	defer file.Close()

	payload, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(append(payload, '\n')); err != nil {
		return "", err
	}
	return target, nil
}

func FailureDebugEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(EnvFailureDebug)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func BuildAdvice(classification Classification) Advice {
	advice := Advice{
		Retryable: classification.Retryable,
	}
	switch classification.Category {
	case CategoryAuth:
		advice.Suggestion = "Check provider API key, base URL, and model permission."
	case CategoryRateLimit:
		advice.RetryDelays = []int{2, 5, 10}
		advice.Suggestion = "Rate limited by provider. Retry later or reduce request frequency."
	case CategoryProvider:
		advice.RetryDelays = []int{2, 5, 10}
		advice.Suggestion = "Provider unavailable. Retry with backoff or switch provider."
	case CategoryTimeout:
		advice.RetryDelays = []int{1, 3, 5}
		advice.Suggestion = "Request timed out. Retry and consider increasing timeout."
	case CategoryNetwork:
		advice.RetryDelays = []int{1, 3, 5}
		advice.Suggestion = "Network error. Check connectivity and DNS, then retry."
	case CategoryCommandExit:
		advice.Suggestion = "Underlying command failed. Check module/runtime logs and dependencies."
	case CategoryInvalidOutput:
		advice.Suggestion = "Planner output invalid. Enable planner debug and verify prompt/schema."
	case CategoryNotFound:
		advice.Suggestion = "Target not found. Verify module/model id and local install state."
	default:
		advice.Suggestion = "Unknown failure. Enable debug flags and inspect logs."
	}
	if len(advice.RetryDelays) > 0 {
		advice.Retryable = true
	}
	return advice
}

func Classify(err error) Classification {
	if err == nil {
		return Classification{Category: CategoryUnknown, Reason: "nil error"}
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if message == "" {
		return Classification{Category: CategoryUnknown, Reason: "empty error"}
	}

	if status := parseStatusCode(message); status > 0 {
		switch {
		case status == 401 || status == 403:
			return Classification{Category: CategoryAuth, Retryable: false, StatusCode: status, Reason: "provider authentication/authorization failed"}
		case status == 429:
			return Classification{Category: CategoryRateLimit, Retryable: true, StatusCode: status, Reason: "provider rate limit"}
		case status >= 500:
			return Classification{Category: CategoryProvider, Retryable: true, StatusCode: status, Reason: "provider service unavailable"}
		}
	}

	if exitCode := parseExitCode(message); exitCode > 0 {
		return Classification{Category: CategoryCommandExit, Retryable: false, ExitCode: exitCode, Reason: "command returned non-zero exit code"}
	}
	if containsAny(message, "deadline exceeded", "timeout", "timed out") {
		return Classification{Category: CategoryTimeout, Retryable: true, Reason: "request timeout"}
	}
	if containsAny(message, "connection refused", "no such host", "temporary failure in name resolution", "tls handshake timeout", "network is unreachable") {
		return Classification{Category: CategoryNetwork, Retryable: true, Reason: "network failure"}
	}
	if containsAny(message, "did not include json", "invalid character", "cannot unmarshal", "unsupported key", "invalid plan") {
		return Classification{Category: CategoryInvalidOutput, Retryable: false, Reason: "invalid planner output"}
	}
	if containsAny(message, "not found", "no such file") {
		return Classification{Category: CategoryNotFound, Retryable: false, Reason: "resource not found"}
	}

	return Classification{Category: CategoryUnknown, Retryable: false, Reason: "unclassified"}
}

func containsAny(text string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

var (
	statusCodeRegexp = regexp.MustCompile(`status\s+(\d{3})`)
	exitCodeRegexp   = regexp.MustCompile(`exit status\s+(\d+)`)
)

func parseStatusCode(message string) int {
	match := statusCodeRegexp.FindStringSubmatch(message)
	if len(match) != 2 {
		return 0
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return value
}

func parseExitCode(message string) int {
	match := exitCodeRegexp.FindStringSubmatch(message)
	if len(match) != 2 {
		return 0
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return value
}

func buildEventID(now time.Time) string {
	return fmt.Sprintf("fail-%d", now.UnixNano())
}
