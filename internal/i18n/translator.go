package i18n

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
)

type Translator interface {
	Translate(text, source, target string) (string, error)
}

type LLMTranslator struct {
	provider string
	model    string
	apiKey   string
	baseURL  string
	timeout  time.Duration
	client   *http.Client
}

func NewLLMTranslator(cfg config.TranslationConfig) *LLMTranslator {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &LLMTranslator{
		provider: cfg.Provider,
		model:    cfg.Model,
		apiKey:   cfg.APIKey,
		baseURL:  cfg.BaseURL,
		timeout:  timeout,
		client:   &http.Client{Timeout: timeout},
	}
}

func (t *LLMTranslator) Translate(text, source, target string) (string, error) {
	if t == nil {
		return "", fmt.Errorf("translation provider is nil")
	}
	if strings.TrimSpace(t.baseURL) == "" {
		return "", fmt.Errorf("translation base URL is empty")
	}
	if strings.TrimSpace(text) == "" {
		return "", nil
	}
	prompt := buildPrompt(text, source, target)
	reqBody := chatCompletionRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "system", Content: "You are a translation engine that only returns the translated text."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal translation request: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create translation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(t.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(t.apiKey))
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("translation request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("translation request failed with status %d", resp.StatusCode)
	}
	var decoded chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode translation response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("translation response has no choices")
	}
	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	return content, nil
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float32       `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func buildPrompt(text, source, target string) string {
	return fmt.Sprintf(
		"Translate the following text from %s to %s. Keep the placeholders like %%s, %%d, %%v, %%q, %%w, and preserve line breaks. Only return the translated text.\n\n%s",
		source,
		target,
		text,
	)
}
