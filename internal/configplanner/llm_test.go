package configplanner

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/llm"
	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
)

func TestBuildLLMPlanWithStubProvider(t *testing.T) {
	orig := llmRegistryFactory
	defer func() { llmRegistryFactory = orig }()
	llmRegistryFactory = func(cfg config.LLMConfig) (*llm.Registry, error) {
		r := llm.NewRegistry()
		if err := r.Register(stubProvider{}); err != nil {
			return nil, err
		}
		return r, nil
	}

	base, err := BuildStaticPlan("llama.cpp", "demo", system.BaseInfoSummary{CPUCores: 8})
	if err != nil {
		t.Fatalf("BuildStaticPlan returned error: %v", err)
	}
	cfg := config.LLMConfig{
		Provider:       "stub",
		Model:          "deepseek-ai/DeepSeek-V3.2",
		TimeoutSeconds: 5,
	}

	plan, err := BuildLLMPlan(context.Background(), base, system.BaseInfoSummary{}, cfg)
	if err != nil {
		t.Fatalf("BuildLLMPlan returned error: %v", err)
	}
	if plan.Source != "llm" {
		t.Fatalf("expected source llm, got %q", plan.Source)
	}
	got := map[string]any{}
	for _, c := range plan.Changes {
		got[c.Key] = c.Value
	}
	if got["ctx_size"] == nil || got["ctx_size"].(int) != 4096 {
		t.Fatalf("expected ctx_size=4096, got %#v", got["ctx_size"])
	}
}

func TestMergeLLMChangesRejectsUnknownKey(t *testing.T) {
	base, err := BuildStaticPlan("llama.cpp", "demo", system.BaseInfoSummary{CPUCores: 8})
	if err != nil {
		t.Fatalf("BuildStaticPlan returned error: %v", err)
	}
	_, err = mergeLLMChanges(base, []Change{
		{Key: "unknown_key", Value: 1},
	})
	if err == nil {
		t.Fatalf("expected error for unknown key")
	}
}

type stubProvider struct{}

func (p stubProvider) Name() string { return "stub" }

func (p stubProvider) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	_ = ctx
	_ = req
	payload, _ := json.Marshal(map[string]any{
		"reason": "stub advice",
		"changes": []map[string]any{
			{"scope": "model.run.llama.cpp", "key": "ctx_size", "value": 4096, "reason": "increase ctx"},
		},
	})
	return llm.Response{Text: string(payload)}, nil
}
