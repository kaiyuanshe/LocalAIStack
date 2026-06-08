package express

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type BenchmarkOptions struct {
	BaseURL  string
	Model    string
	Prompt   string
	Requests int
	Timeout  time.Duration
}

func Benchmark(ctx context.Context, opts BenchmarkOptions) (BenchmarkReport, error) {
	if opts.Requests <= 0 {
		opts.Requests = 3
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 60 * time.Second
	}
	if strings.TrimSpace(opts.BaseURL) == "" {
		opts.BaseURL = "http://127.0.0.1:8000"
	}
	if strings.TrimSpace(opts.Prompt) == "" {
		opts.Prompt = "Write one short sentence about local AI."
	}

	client := &http.Client{Timeout: opts.Timeout}
	baseURL := strings.TrimRight(opts.BaseURL, "/")
	if strings.TrimSpace(opts.Model) == "" {
		models, err := getModels(ctx, client, baseURL)
		if err != nil {
			return BenchmarkReport{BaseURL: baseURL, Requests: opts.Requests}, err
		}
		if len(models) > 0 {
			opts.Model = models[0]
		} else {
			opts.Model = "default"
		}
	}

	report := BenchmarkReport{BaseURL: baseURL, Requests: opts.Requests}
	startAll := time.Now()
	for i := 0; i < opts.Requests; i++ {
		start := time.Now()
		preview, err := chatCompletion(ctx, client, baseURL, opts.Model, opts.Prompt)
		if err != nil {
			report.Failed++
			continue
		}
		report.Successful++
		report.TotalDuration += time.Since(start)
		report.ApproxTokens += approxTokenCount(preview)
	}
	if report.TotalDuration == 0 {
		report.TotalDuration = time.Since(startAll)
	}
	if report.Successful > 0 {
		report.AverageLatency = report.TotalDuration / time.Duration(report.Successful)
		seconds := report.TotalDuration.Seconds()
		if seconds > 0 {
			report.ApproxTokensSec = float64(report.ApproxTokens) / seconds
		}
	}
	return report, nil
}

func approxTokenCount(text string) int {
	words := strings.Fields(text)
	if len(words) == 0 {
		return 0
	}
	return len(words)
}
