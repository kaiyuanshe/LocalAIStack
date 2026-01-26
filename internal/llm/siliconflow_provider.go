package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

type SiliconFlowConfig struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

type SiliconFlowProvider struct {
	cfg    SiliconFlowConfig
	client *http.Client
}

func NewSiliconFlowProvider(cfg SiliconFlowConfig) *SiliconFlowProvider {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &SiliconFlowProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

func (p *SiliconFlowProvider) Name() string {
	return "siliconflow"
}

func (p *SiliconFlowProvider) Generate(ctx context.Context, req Request) (Response, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = strings.TrimSpace(p.cfg.Model)
	}
	if model == "" {
		return Response{}, i18n.Errorf("siliconflow model is required")
	}
	if strings.TrimSpace(p.cfg.BaseURL) == "" {
		return Response{}, i18n.Errorf("siliconflow base URL is required")
	}
	if strings.TrimSpace(p.cfg.APIKey) == "" {
		return Response{}, i18n.Errorf("siliconflow API key is required")
	}

	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": req.Prompt,
			},
		},
		"temperature": 0,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL, bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Response{}, i18n.Errorf("siliconflow request failed with status %d", resp.StatusCode)
	}

	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return Response{}, err
	}
	if len(decoded.Choices) == 0 {
		return Response{}, i18n.Errorf("siliconflow response missing choices")
	}
	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return Response{}, i18n.Errorf("siliconflow response was empty")
	}
	return Response{Text: content}, nil
}
