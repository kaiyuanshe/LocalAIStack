package express

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HealthOptions struct {
	BaseURL string
	Model   string
	Prompt  string
	Timeout time.Duration
}

func Healthcheck(ctx context.Context, opts HealthOptions) (HealthReport, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if strings.TrimSpace(opts.BaseURL) == "" {
		opts.BaseURL = "http://127.0.0.1:8000"
	}
	if strings.TrimSpace(opts.Prompt) == "" {
		opts.Prompt = "Say OK in one short sentence."
	}

	client := &http.Client{Timeout: opts.Timeout}
	report := HealthReport{BaseURL: strings.TrimRight(opts.BaseURL, "/")}

	models, err := getModels(ctx, client, report.BaseURL)
	if err != nil {
		return report, err
	}
	report.ModelsOK = true
	report.ModelCount = len(models)
	if opts.Model == "" && len(models) > 0 {
		opts.Model = models[0]
	}
	if strings.TrimSpace(opts.Model) == "" {
		opts.Model = "default"
	}

	start := time.Now()
	preview, err := chatCompletion(ctx, client, report.BaseURL, opts.Model, opts.Prompt)
	if err != nil {
		return report, err
	}
	report.Latency = time.Since(start)
	report.ChatOK = true
	report.ResponsePreview = preview
	report.OpenAICompatible = report.ModelsOK && report.ChatOK
	return report, nil
}

func getModels(ctx context.Context, client *http.Client, baseURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET /v1/models failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GET /v1/models returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var decoded struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode /v1/models: %w", err)
	}
	models := make([]string, 0, len(decoded.Data))
	for _, item := range decoded.Data {
		if strings.TrimSpace(item.ID) != "" {
			models = append(models, item.ID)
		}
	}
	return models, nil
}

func chatCompletion(ctx context.Context, client *http.Client, baseURL, model, prompt string) (string, error) {
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  32,
		"temperature": 0,
		"stream":      false,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST /v1/chat/completions failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("POST /v1/chat/completions returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode /v1/chat/completions: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("chat completion response has no choices")
	}
	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("chat completion response content is empty")
	}
	if len(content) > 160 {
		content = content[:160] + "..."
	}
	return content, nil
}
