package express

import "time"

type CheckStatus string

const (
	CheckPass CheckStatus = "pass"
	CheckWarn CheckStatus = "warn"
	CheckFail CheckStatus = "fail"
)

type CheckResult struct {
	Name       string
	Status     CheckStatus
	Message    string
	Suggestion string
}

type PreflightReport struct {
	RecipeID string
	Checks   []CheckResult
}

func (r PreflightReport) Failed() bool {
	for _, check := range r.Checks {
		if check.Status == CheckFail {
			return true
		}
	}
	return false
}

type HealthReport struct {
	BaseURL          string
	ModelsOK         bool
	ChatOK           bool
	Latency          time.Duration
	ModelCount       int
	ResponsePreview  string
	OpenAICompatible bool
}

type BenchmarkReport struct {
	BaseURL         string
	Requests        int
	Successful      int
	Failed          int
	TotalDuration   time.Duration
	AverageLatency  time.Duration
	ApproxTokens    int
	ApproxTokensSec float64
}

type RunPlan struct {
	RecipeID string
	Mode     string
	WorkDir  string
	Command  []string
	Notes    []string
}
