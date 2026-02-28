package commands

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/llm"
	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
)

func TestBuildVLLMServeArgs(t *testing.T) {
	defaults := vllmRunDefaults{
		maxModelLen:            4096,
		gpuMemUtil:             0.88,
		dtype:                  "float16",
		tensorParallelSize:     2,
		enforceEager:           true,
		optimizationLevel:      0,
		maxNumSeqs:             4,
		disableCustomAllReduce: true,
	}
	args := buildVLLMServeArgs("org/repo", "0.0.0.0", 8080, defaults, true)
	joined := strings.Join(args, " ")

	wantTokens := []string{
		"serve", "org/repo", "--host", "0.0.0.0", "--port", "8080",
		"--dtype", "float16",
		"--max-model-len", "4096",
		"--gpu-memory-utilization", "0.88",
		"--tensor-parallel-size", "2",
		"--enforce-eager",
		"--optimization-level", "0",
		"--disable-custom-all-reduce",
		"--max-num-seqs", "4",
		"--trust-remote-code",
	}
	for _, token := range wantTokens {
		if !strings.Contains(joined, token) {
			t.Fatalf("expected token %q in args: %v", token, args)
		}
	}
}

func TestBuildLlamaServerArgs(t *testing.T) {
	args := buildLlamaServerArgs(
		"/models/foo.gguf",
		llamaRunDefaults{threads: 8, ctxSize: 4096, gpuLayers: 20, tensorSplit: "50,50"},
		"127.0.0.1",
		9000,
		llamaSamplingParams{
			Temperature:     0.7,
			TopP:            0.9,
			TopK:            40,
			MinP:            0.05,
			PresencePenalty: 1.2,
			RepeatPenalty:   1.1,
		},
		llamaBatchParams{BatchSize: 256, UBatchSize: 128},
		`{"enable_thinking":false}`,
	)
	joined := strings.Join(args, " ")

	wantTokens := []string{
		"--model /models/foo.gguf",
		"--threads 8",
		"--ctx-size 4096",
		"--n-gpu-layers 20",
		"--host 127.0.0.1",
		"--port 9000",
		"--tensor-split 50,50",
		"--batch-size 256",
		"--ubatch-size 128",
		"--chat-template-kwargs {\"enable_thinking\":false}",
	}
	for _, token := range wantTokens {
		if !strings.Contains(joined, token) {
			t.Fatalf("expected token %q in args: %v", token, args)
		}
	}
}

func TestBuildLlamaServerArgsSkipsOptional(t *testing.T) {
	args := buildLlamaServerArgs(
		"/models/foo.gguf",
		llamaRunDefaults{threads: 4, ctxSize: 2048, gpuLayers: 0},
		"0.0.0.0",
		8080,
		llamaSamplingParams{Temperature: 0.7, TopP: 0.8, TopK: 20, MinP: 0, PresencePenalty: 1.5, RepeatPenalty: 1.0},
		llamaBatchParams{},
		"",
	)
	joined := strings.Join(args, " ")

	skipTokens := []string{"--tensor-split", "--batch-size", "--ubatch-size", "--chat-template-kwargs"}
	for _, token := range skipTokens {
		if strings.Contains(joined, token) {
			t.Fatalf("did not expect token %q in args: %v", token, args)
		}
	}
}

func TestParseSmartRunAdvice(t *testing.T) {
	text := "```json\n{\"llama\":{\"threads\":12,\"ctx_size\":8192},\"reason\":\"ok\"}\n```"
	var out smartRunAdviceEnvelope
	if err := parseSmartRunAdvice(text, &out); err != nil {
		t.Fatalf("parseSmartRunAdvice returned error: %v", err)
	}
	if out.Llama.Threads == nil || *out.Llama.Threads != 12 {
		t.Fatalf("expected llama threads=12, got %+v", out.Llama.Threads)
	}
	if out.Llama.CtxSize == nil || *out.Llama.CtxSize != 8192 {
		t.Fatalf("expected llama ctx_size=8192, got %+v", out.Llama.CtxSize)
	}
}

func TestApplyLlamaAdviceRespectsChangedFlags(t *testing.T) {
	defaults := llamaRunDefaults{threads: 8, ctxSize: 4096, gpuLayers: 20}
	resolvedBatch := 256
	resolvedUBatch := 128
	sampling := llamaSamplingParams{Temperature: 0.7, TopP: 0.8, TopK: 20, MinP: 0, PresencePenalty: 1.5, RepeatPenalty: 1.0}
	chatKwargs := ""
	advice := llamaPlannerAdvice{
		Threads:   intPtr(24),
		CtxSize:   intPtr(16384),
		BatchSize: intPtr(1024),
	}
	changed := map[string]bool{
		"threads":    true,
		"ctx_size":   false,
		"batch_size": true,
	}
	applyLlamaAdvice(&defaults, &resolvedBatch, &resolvedUBatch, &sampling, &chatKwargs, advice, changed)
	if defaults.threads != 8 {
		t.Fatalf("threads should remain user-defined, got %d", defaults.threads)
	}
	if defaults.ctxSize != 16384 {
		t.Fatalf("ctx_size should be updated by advice, got %d", defaults.ctxSize)
	}
	if resolvedBatch != 256 {
		t.Fatalf("batch_size should remain user-defined, got %d", resolvedBatch)
	}
}

func TestApplyVLLMAdviceRespectsChangedFlags(t *testing.T) {
	defaults := vllmRunDefaults{maxModelLen: 4096, gpuMemUtil: 0.88}
	trustRemoteCode := false
	advice := vllmPlannerAdvice{
		MaxModelLen:          intPtr(8192),
		GPUMemoryUtilization: floatPtr(0.95),
		TrustRemoteCode:      boolPtr(true),
	}
	changed := map[string]bool{
		"max_model_len":          true,
		"gpu_memory_utilization": false,
		"trust_remote_code":      true,
	}
	applyVLLMAdvice(&defaults, &trustRemoteCode, advice, changed)
	if defaults.maxModelLen != 4096 {
		t.Fatalf("max_model_len should remain user-defined, got %d", defaults.maxModelLen)
	}
	if defaults.gpuMemUtil != 0.95 {
		t.Fatalf("gpu_memory_utilization should be updated by advice, got %.2f", defaults.gpuMemUtil)
	}
	if trustRemoteCode {
		t.Fatalf("trust_remote_code should remain user-defined false")
	}
}

func TestSuggestAdviceWithStubProvider(t *testing.T) {
	original := llmRegistryFactory
	defer func() { llmRegistryFactory = original }()
	llmRegistryFactory = func(cfg config.LLMConfig) (*llm.Registry, error) {
		registry := llm.NewRegistry()
		if err := registry.Register(stubLLMProvider{}); err != nil {
			return nil, err
		}
		return registry, nil
	}

	cfg := config.LLMConfig{Provider: "stub", Model: "deepseek-ai/DeepSeek-V3.2", TimeoutSeconds: 5}
	info := system.BaseInfoSummary{CPUCores: 8, MemoryKB: 32 * 1024 * 1024}
	llamaAdvice, err := suggestLlamaAdvice(context.Background(), cfg, "demo", "/tmp/demo.gguf", info, llamaRunDefaults{threads: 8, ctxSize: 4096, gpuLayers: 20}, llamaBatchParams{BatchSize: 256, UBatchSize: 128}, llamaSamplingParams{Temperature: 0.7, TopP: 0.8, TopK: 20}, "")
	if err != nil {
		t.Fatalf("suggestLlamaAdvice returned error: %v", err)
	}
	if llamaAdvice.CtxSize == nil || *llamaAdvice.CtxSize != 8192 {
		t.Fatalf("expected llama advice ctx_size=8192, got %+v", llamaAdvice.CtxSize)
	}

	vllmAdvice, err := suggestVLLMAdvice(context.Background(), cfg, "demo", "org/repo", info, vllmRunDefaults{maxModelLen: 4096, gpuMemUtil: 0.88}, false)
	if err != nil {
		t.Fatalf("suggestVLLMAdvice returned error: %v", err)
	}
	if vllmAdvice.MaxModelLen == nil || *vllmAdvice.MaxModelLen != 6144 {
		t.Fatalf("expected vllm advice max_model_len=6144, got %+v", vllmAdvice.MaxModelLen)
	}
}

func TestSuggestAdvicePromptIncludesRecommendationsAndBaseInfo(t *testing.T) {
	baseInfoJSON := `{"cpu":{"model":"Test CPU","cores":12},"gpu":"Test GPU","memory":"32768000 kB","disk":{"total":"1.0 TB","available":"900.0 GB"}}`

	var prompts []string
	original := llmRegistryFactory
	originalRecommendationsLoader := llamaRunRecommendationsLoader
	originalBaseInfoLoader := baseInfoPromptLoader
	defer func() {
		llmRegistryFactory = original
		llamaRunRecommendationsLoader = originalRecommendationsLoader
		baseInfoPromptLoader = originalBaseInfoLoader
	}()
	llmRegistryFactory = func(cfg config.LLMConfig) (*llm.Registry, error) {
		registry := llm.NewRegistry()
		if err := registry.Register(captureLLMProvider{capture: &prompts}); err != nil {
			return nil, err
		}
		return registry, nil
	}
	llamaRunRecommendationsLoader = func() (string, error) {
		return "# tuned defaults", nil
	}
	baseInfoPromptLoader = func() (string, error) {
		return baseInfoJSON, nil
	}

	cfg := config.LLMConfig{Provider: "capture", Model: "deepseek-ai/DeepSeek-V3.2", TimeoutSeconds: 5}
	info := system.BaseInfoSummary{CPUCores: 8, MemoryKB: 32 * 1024 * 1024}
	if _, err := suggestLlamaAdvice(context.Background(), cfg, "demo", "/tmp/demo.gguf", info, llamaRunDefaults{threads: 8, ctxSize: 4096, gpuLayers: 20}, llamaBatchParams{BatchSize: 256, UBatchSize: 128}, llamaSamplingParams{Temperature: 0.7, TopP: 0.8, TopK: 20}, ""); err != nil {
		t.Fatalf("suggestLlamaAdvice returned error: %v", err)
	}
	if _, err := suggestVLLMAdvice(context.Background(), cfg, "demo", "org/repo", info, vllmRunDefaults{maxModelLen: 4096, gpuMemUtil: 0.88}, false); err != nil {
		t.Fatalf("suggestVLLMAdvice returned error: %v", err)
	}

	if len(prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(prompts))
	}

	llamaPrompt := prompts[0]
	if !strings.Contains(llamaPrompt, "Reference tuning guide for llama.cpp") {
		t.Fatalf("expected llama prompt to include run recommendations")
	}
	if !strings.Contains(llamaPrompt, "Collected base hardware info (json):") {
		t.Fatalf("expected llama prompt to include base_info.json")
	}
	if !strings.Contains(llamaPrompt, `"cpu":{"model":"Test CPU","cores":12}`) {
		t.Fatalf("expected llama prompt to include base_info.json content")
	}

	vllmPrompt := prompts[1]
	if !strings.Contains(vllmPrompt, "Collected base hardware info (json):") {
		t.Fatalf("expected vllm prompt to include base_info.json")
	}
	if !strings.Contains(vllmPrompt, `"gpu":"Test GPU"`) {
		t.Fatalf("expected vllm prompt to include base_info.json content")
	}
}

func TestEvaluateSmartRunOutcome(t *testing.T) {
	source, reason, fatal := evaluateSmartRunOutcome(false, nil, false)
	if source != "static" || reason == "" || fatal != nil {
		t.Fatalf("unexpected disabled outcome: source=%q reason=%q fatal=%v", source, reason, fatal)
	}

	source, reason, fatal = evaluateSmartRunOutcome(true, nil, false)
	if source != "llm" || reason == "" || fatal != nil {
		t.Fatalf("unexpected success outcome: source=%q reason=%q fatal=%v", source, reason, fatal)
	}

	runErr := errors.New("planner failed")
	source, reason, fatal = evaluateSmartRunOutcome(true, runErr, false)
	if source != "static" || !strings.Contains(reason, "planner failed") || fatal != nil {
		t.Fatalf("unexpected fallback outcome: source=%q reason=%q fatal=%v", source, reason, fatal)
	}

	source, reason, fatal = evaluateSmartRunOutcome(true, runErr, true)
	if source != "static" || !strings.Contains(reason, "planner failed") || fatal == nil {
		t.Fatalf("unexpected strict outcome: source=%q reason=%q fatal=%v", source, reason, fatal)
	}
}

type stubLLMProvider struct{}

func (p stubLLMProvider) Name() string { return "stub" }

func (p stubLLMProvider) Generate(_ context.Context, req llm.Request) (llm.Response, error) {
	if strings.Contains(req.Prompt, `"runtime":"llama.cpp"`) {
		payload, _ := json.Marshal(map[string]any{
			"llama": map[string]any{
				"ctx_size": 8192,
			},
		})
		return llm.Response{Text: string(payload)}, nil
	}
	payload, _ := json.Marshal(map[string]any{
		"vllm": map[string]any{
			"max_model_len": 6144,
		},
	})
	return llm.Response{Text: string(payload)}, nil
}

func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool        { return &v }

type captureLLMProvider struct {
	capture *[]string
}

func (p captureLLMProvider) Name() string { return "capture" }

func (p captureLLMProvider) Generate(_ context.Context, req llm.Request) (llm.Response, error) {
	*p.capture = append(*p.capture, req.Prompt)
	if strings.Contains(req.Prompt, `"runtime":"llama.cpp"`) {
		return llm.Response{Text: `{"llama":{"ctx_size":8192}}`}, nil
	}
	return llm.Response{Text: `{"vllm":{"max_model_len":6144}}`}, nil
}
