package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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
		attentionBackend:       "TRITON_ATTN",
		enforceEager:           true,
		optimizationLevel:      0,
		maxNumSeqs:             4,
		maxNumBatchedTokens:    256,
		compilationConfig:      `{"cudagraph_mode":"full_and_piecewise","cudagraph_capture_sizes":[1]}`,
		skipMMProfiling:        true,
		limitMMPerPrompt:       `{"image":0,"video":0}`,
		disableCustomAllReduce: true,
	}
	args := buildVLLMServeArgs("org/repo", "qwen35-27b", "0.0.0.0", 8080, defaults, true)
	joined := strings.Join(args, " ")

	wantTokens := []string{
		"serve", "--model", "org/repo", "--host", "0.0.0.0", "--port", "8080",
		"--served-model-name", "qwen35-27b",
		"--dtype", "float16",
		"--max-model-len", "4096",
		"--gpu-memory-utilization", "0.88",
		"--tensor-parallel-size", "2",
		"--attention-backend", "TRITON_ATTN",
		"--enforce-eager",
		"--optimization-level", "0",
		"--disable-custom-all-reduce",
		"--max-num-seqs", "4",
		"--max-num-batched-tokens", "256",
		"--skip-mm-profiling",
		"--limit-mm-per-prompt", "{\"image\":0,\"video\":0}",
		"--compilation-config", "{\"cudagraph_mode\":\"full_and_piecewise\",\"cudagraph_capture_sizes\":[1]}",
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

func TestSaveAndLoadSmartRunAdvice(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	advice := smartRunAdviceEnvelope{Llama: llamaPlannerAdvice{Threads: intPtr(16), CtxSize: intPtr(8192)}}
	if err := saveSmartRunAdvice("llama.cpp", "demo/model", "Q4_K_M.gguf", advice); err != nil {
		t.Fatalf("saveSmartRunAdvice returned error: %v", err)
	}

	loaded, err := loadSmartRunAdvice("llama.cpp", "demo/model", "Q4_K_M.gguf")
	if err != nil {
		t.Fatalf("loadSmartRunAdvice returned error: %v", err)
	}
	if loaded.Llama.Threads == nil || *loaded.Llama.Threads != 16 {
		t.Fatalf("expected saved threads=16, got %+v", loaded.Llama.Threads)
	}
	if loaded.Llama.CtxSize == nil || *loaded.Llama.CtxSize != 8192 {
		t.Fatalf("expected saved ctx_size=8192, got %+v", loaded.Llama.CtxSize)
	}

	path, err := smartRunAdvicePath("llama.cpp", "demo/model", "Q4_K_M.gguf")
	if err != nil {
		t.Fatalf("smartRunAdvicePath returned error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected persisted advice file at %s: %v", path, err)
	}
	if !strings.Contains(path, filepath.Join(".localaistack", "smart-run")) {
		t.Fatalf("expected advice path under .localaistack/smart-run, got %s", path)
	}
}

func TestLoadSmartRunAdviceMissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, err := loadSmartRunAdvice("vllm", "demo/model", "org/repo")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestEvaluateSmartRunOutcomeWithSourceUsesLocalReason(t *testing.T) {
	source, reason, fatal := evaluateSmartRunOutcomeWithSource(true, "local", "Reused last saved smart-run parameters", nil, false)
	if fatal != nil {
		t.Fatalf("expected nil fatal error, got %v", fatal)
	}
	if source != "local" {
		t.Fatalf("expected source local, got %s", source)
	}
	if reason != "Reused last saved smart-run parameters" {
		t.Fatalf("unexpected reason: %s", reason)
	}
}

func TestModelRunRefreshRequiresSmartRun(t *testing.T) {
	root := &cobra.Command{Use: "las"}
	RegisterModelCommands(root)

	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"model", "run", "demo/model", "--smart-run-refresh"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("expected error when --smart-run-refresh is used without --smart-run")
	}
	if !strings.Contains(err.Error(), "smart-run-refresh requires --smart-run") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSmartRunCacheListCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := saveSmartRunAdvice("llama.cpp", "demo/model", "Q4_K_M.gguf", smartRunAdviceEnvelope{
		Llama: llamaPlannerAdvice{Threads: intPtr(12)},
	}); err != nil {
		t.Fatalf("saveSmartRunAdvice returned error: %v", err)
	}
	if err := saveSmartRunAdvice("vllm", "another/model", "org/repo", smartRunAdviceEnvelope{
		VLLM: vllmPlannerAdvice{MaxModelLen: intPtr(4096)},
	}); err != nil {
		t.Fatalf("saveSmartRunAdvice returned error: %v", err)
	}

	root := &cobra.Command{Use: "las"}
	RegisterModelCommands(root)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"model", "smart-run-cache", "list", "demo/model"})

	if err := root.Execute(); err != nil {
		t.Fatalf("list command failed: %v", err)
	}
	text := buf.String()
	if !strings.Contains(text, "demo/model") || !strings.Contains(text, "Q4_K_M.gguf") {
		t.Fatalf("expected demo/model cache entry in output, got: %s", text)
	}
	if strings.Contains(text, "another/model") {
		t.Fatalf("did not expect another/model entry in filtered output: %s", text)
	}
}

func TestSmartRunCacheRemoveCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := saveSmartRunAdvice("llama.cpp", "demo/model", "Q4_K_M.gguf", smartRunAdviceEnvelope{
		Llama: llamaPlannerAdvice{Threads: intPtr(12)},
	}); err != nil {
		t.Fatalf("saveSmartRunAdvice returned error: %v", err)
	}
	if err := saveSmartRunAdvice("vllm", "demo/model", "org/repo", smartRunAdviceEnvelope{
		VLLM: vllmPlannerAdvice{MaxModelLen: intPtr(4096)},
	}); err != nil {
		t.Fatalf("saveSmartRunAdvice returned error: %v", err)
	}

	root := &cobra.Command{Use: "las"}
	RegisterModelCommands(root)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"model", "smart-run-cache", "rm", "demo/model"})

	if err := root.Execute(); err != nil {
		t.Fatalf("rm command failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Removed 2 smart-run cache entries.") {
		t.Fatalf("unexpected rm output: %s", buf.String())
	}

	entries, err := listSmartRunAdviceEntries("", "demo/model")
	if err != nil {
		t.Fatalf("listSmartRunAdviceEntries returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries after rm, got %d", len(entries))
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
