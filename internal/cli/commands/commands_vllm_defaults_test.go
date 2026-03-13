package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
)

func TestDefaultVLLMRunParams_V100DualGPU(t *testing.T) {
	info := system.BaseInfoSummary{
		MemoryKB: 32691216,
		GPUName:  "Tesla V100-SXM2-16GB",
		GPUCount: 2,
	}

	got := defaultVLLMRunParams(info)

	if got.maxModelLen != 2048 {
		t.Fatalf("expected maxModelLen=2048, got %d", got.maxModelLen)
	}
	if got.gpuMemUtil != 0.88 {
		t.Fatalf("expected gpuMemUtil=0.88, got %.2f", got.gpuMemUtil)
	}
	if got.dtype != "float16" {
		t.Fatalf("expected dtype=float16, got %q", got.dtype)
	}
	if got.tensorParallelSize != 2 {
		t.Fatalf("expected tensorParallelSize=2, got %d", got.tensorParallelSize)
	}
	if !got.enforceEager {
		t.Fatalf("expected enforceEager=true")
	}
	if got.optimizationLevel != 0 {
		t.Fatalf("expected optimizationLevel=0, got %d", got.optimizationLevel)
	}
	if got.maxNumSeqs != 2 {
		t.Fatalf("expected maxNumSeqs=2, got %d", got.maxNumSeqs)
	}
	if !got.disableCustomAllReduce {
		t.Fatalf("expected disableCustomAllReduce=true")
	}
	if len(got.env) != 0 {
		t.Fatalf("expected env empty, got %v", got.env)
	}
}

func TestDefaultVLLMRunParams_A100HighVRAM(t *testing.T) {
	info := system.BaseInfoSummary{
		MemoryKB: 262144000,
		GPUName:  "NVIDIA A100-SXM4-80GB",
		GPUCount: 4,
	}

	got := defaultVLLMRunParams(info)

	if got.maxModelLen != 32768 {
		t.Fatalf("expected maxModelLen=32768, got %d", got.maxModelLen)
	}
	if got.gpuMemUtil != 0.92 {
		t.Fatalf("expected gpuMemUtil=0.92, got %.2f", got.gpuMemUtil)
	}
	if got.dtype != "" {
		t.Fatalf("expected dtype unset for newer gpu, got %q", got.dtype)
	}
	if got.tensorParallelSize != 4 {
		t.Fatalf("expected tensorParallelSize=4, got %d", got.tensorParallelSize)
	}
	if got.enforceEager {
		t.Fatalf("expected enforceEager=false")
	}
	if got.optimizationLevel != 2 {
		t.Fatalf("expected optimizationLevel=2, got %d", got.optimizationLevel)
	}
	if got.maxNumSeqs != 16 {
		t.Fatalf("expected maxNumSeqs=16, got %d", got.maxNumSeqs)
	}
	if got.disableCustomAllReduce {
		t.Fatalf("expected disableCustomAllReduce=false")
	}
	if len(got.env) != 0 {
		t.Fatalf("expected env empty, got %v", got.env)
	}
}

func TestFinalizeVLLMRunParams_V100RejectsUnsafeSmartRunAdvice(t *testing.T) {
	info := system.BaseInfoSummary{
		MemoryKB: 32691216,
		GPUName:  "Tesla V100-SXM2-16GB",
		GPUCount: 2,
	}

	got := finalizeVLLMRunParams(info, vllmRunDefaults{
		maxModelLen:            8192,
		gpuMemUtil:             0.90,
		dtype:                  "float16",
		tensorParallelSize:     2,
		enforceEager:           false,
		optimizationLevel:      1,
		maxNumSeqs:             4,
		maxNumBatchedTokens:    512,
		disableCustomAllReduce: false,
		env:                    []string{"NCCL_IB_DISABLE=1", "NCCL_P2P_DISABLE=1"},
	}, true)

	if got.maxModelLen != 512 {
		t.Fatalf("expected maxModelLen=512, got %d", got.maxModelLen)
	}
	if got.gpuMemUtil != 0.80 {
		t.Fatalf("expected gpuMemUtil=0.80, got %.2f", got.gpuMemUtil)
	}
	if !got.enforceEager {
		t.Fatalf("expected enforceEager=true")
	}
	if got.optimizationLevel != 0 {
		t.Fatalf("expected optimizationLevel=0, got %d", got.optimizationLevel)
	}
	if got.maxNumSeqs != 1 {
		t.Fatalf("expected maxNumSeqs=1, got %d", got.maxNumSeqs)
	}
	if got.maxNumBatchedTokens != 128 {
		t.Fatalf("expected maxNumBatchedTokens=128, got %d", got.maxNumBatchedTokens)
	}
	if got.attentionBackend != "TRITON_ATTN" {
		t.Fatalf("expected attentionBackend=TRITON_ATTN, got %q", got.attentionBackend)
	}
	if got.compilationConfig != `{"cudagraph_mode":"full_and_piecewise","cudagraph_capture_sizes":[1]}` {
		t.Fatalf("unexpected compilationConfig: %q", got.compilationConfig)
	}
	if !got.skipMMProfiling {
		t.Fatalf("expected skipMMProfiling=true")
	}
	if got.limitMMPerPrompt != `{"image":0,"video":0}` {
		t.Fatalf("unexpected limitMMPerPrompt: %q", got.limitMMPerPrompt)
	}
	if !got.disableCustomAllReduce {
		t.Fatalf("expected disableCustomAllReduce=true")
	}
	envJoined := strings.Join(got.env, " ")
	if !strings.Contains(envJoined, "CUDA_VISIBLE_DEVICES=0,1") {
		t.Fatalf("expected CUDA_VISIBLE_DEVICES=0,1 in env, got %v", got.env)
	}
	if !strings.Contains(envJoined, "VLLM_DISABLE_PYNCCL=1") {
		t.Fatalf("expected VLLM_DISABLE_PYNCCL=1 in env, got %v", got.env)
	}
	if strings.Contains(envJoined, "NCCL_IB_DISABLE=1") || strings.Contains(envJoined, "NCCL_P2P_DISABLE=1") {
		t.Fatalf("did not expect NCCL disable env vars in env, got %v", got.env)
	}
}

func TestFinalizeVLLMRunParams_RecomputesEnvFromFinalFlags(t *testing.T) {
	info := system.BaseInfoSummary{
		MemoryKB: 262144000,
		GPUName:  "NVIDIA A100-SXM4-80GB",
		GPUCount: 4,
	}

	got := finalizeVLLMRunParams(info, vllmRunDefaults{
		maxModelLen:            32768,
		gpuMemUtil:             0.92,
		tensorParallelSize:     8,
		optimizationLevel:      2,
		maxNumSeqs:             16,
		disableCustomAllReduce: false,
		env:                    []string{"NCCL_IB_DISABLE=1", "NCCL_P2P_DISABLE=1"},
	}, true)

	if got.tensorParallelSize != 4 {
		t.Fatalf("expected tensorParallelSize=4, got %d", got.tensorParallelSize)
	}
	envJoined := strings.Join(got.env, " ")
	if !strings.Contains(envJoined, "CUDA_VISIBLE_DEVICES=0,1,2,3") {
		t.Fatalf("expected CUDA_VISIBLE_DEVICES=0,1,2,3 in env, got %v", got.env)
	}
}

func TestBuildCUDAVisibleDevices(t *testing.T) {
	if got := buildCUDAVisibleDevices(2); got != "0,1" {
		t.Fatalf("expected 0,1 got %q", got)
	}
	if got := buildCUDAVisibleDevices(4); got != "0,1,2,3" {
		t.Fatalf("expected 0,1,2,3 got %q", got)
	}
	if got := buildCUDAVisibleDevices(0); got != "" {
		t.Fatalf("expected empty string got %q", got)
	}
}

func TestSuggestVLLMServedModelName(t *testing.T) {
	if got := suggestVLLMServedModelName("tclf90/Qwen3.5-27B-AWQ"); got != "qwen35-27b" {
		t.Fatalf("expected qwen35-27b got %q", got)
	}
	if got := suggestVLLMServedModelName("unsloth/Qwen3.5-35B-A3B-GGUF"); got != "qwen35-35b-a3b" {
		t.Fatalf("expected qwen35-35b-a3b got %q", got)
	}
}

func TestBuildVLLMServeArgs_SkipsUnsetOptimizationLevel(t *testing.T) {
	args := buildVLLMServeArgs("org/repo", "qwen35-27b", "0.0.0.0", 8080, vllmRunDefaults{
		optimizationLevel: -1,
	}, true)
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--optimization-level") {
		t.Fatalf("did not expect --optimization-level in args: %v", args)
	}
}

func TestFinalizeVLLMRunParams_UserOverridesCanBeReapplied(t *testing.T) {
	info := system.BaseInfoSummary{
		MemoryKB: 32691216,
		GPUName:  "Tesla V100-SXM2-16GB",
		GPUCount: 2,
	}

	got := finalizeVLLMRunParams(info, vllmRunDefaults{
		maxModelLen:        32768,
		gpuMemUtil:         0.95,
		tensorParallelSize: 2,
	}, false)

	// finalize keeps conservative defaults; caller may reapply explicit user flags afterward.
	got.maxModelLen = clampInt(32768, 256, 131072)
	got.gpuMemUtil = clampFloat(0.95, 0, 0.98)

	if got.maxModelLen != 32768 {
		t.Fatalf("expected maxModelLen=32768, got %d", got.maxModelLen)
	}
	if got.gpuMemUtil != 0.95 {
		t.Fatalf("expected gpuMemUtil=0.95, got %.2f", got.gpuMemUtil)
	}
}

func TestFinalizeVLLMRunParams_LegacyMultimodalSkipsTextOnlyFlags(t *testing.T) {
	info := system.BaseInfoSummary{
		MemoryKB: 32691216,
		GPUName:  "Tesla V100-SXM2-16GB",
		GPUCount: 2,
	}

	got := finalizeVLLMRunParams(info, vllmRunDefaults{}, false)

	if !got.skipMMProfiling {
		t.Fatalf("expected skipMMProfiling=true")
	}
	if got.limitMMPerPrompt != `{"image":0,"video":0}` {
		t.Fatalf("unexpected limitMMPerPrompt: %q", got.limitMMPerPrompt)
	}
}

func TestIsLikelyTextOnlyVLLMModel(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"architectures":["Qwen3ForCausalLM"]}`), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if !isLikelyTextOnlyVLLMModel(dir) {
		t.Fatalf("expected text-only model detection")
	}
}

func TestIsLikelyTextOnlyVLLMModel_Multimodal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"architectures":["Qwen2_5_VLForConditionalGeneration"]}`), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if isLikelyTextOnlyVLLMModel(dir) {
		t.Fatalf("expected multimodal model detection")
	}
}

func TestShouldAutoEnableVLLMTrustRemoteCode_AutoMap(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"auto_map":{"AutoConfig":"configuration_ouro.OuroConfig"}}`), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if !shouldAutoEnableVLLMTrustRemoteCode(dir) {
		t.Fatalf("expected trust_remote_code auto-enable when auto_map exists")
	}
}

func TestShouldAutoEnableVLLMTrustRemoteCode_PythonFiles(t *testing.T) {
	dir := t.TempDir()
	pyPath := filepath.Join(dir, "modeling_ouro.py")
	if err := os.WriteFile(pyPath, []byte("# custom model code"), 0o644); err != nil {
		t.Fatalf("failed to write python file: %v", err)
	}

	if !shouldAutoEnableVLLMTrustRemoteCode(dir) {
		t.Fatalf("expected trust_remote_code auto-enable when python files exist")
	}
}

func TestShouldAutoEnableVLLMTrustRemoteCode_DefaultFalse(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"architectures":["LlamaForCausalLM"]}`), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if shouldAutoEnableVLLMTrustRemoteCode(dir) {
		t.Fatalf("expected trust_remote_code auto-enable to be false without markers")
	}
}

func TestShouldAutoEnableVLLMTrustRemoteCode_Qwen35ConditionalGeneration(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"architectures":["Qwen3_5ForConditionalGeneration"],"model_type":"qwen3_5"}`), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if !shouldAutoEnableVLLMTrustRemoteCode(dir) {
		t.Fatalf("expected trust_remote_code auto-enable for qwen3_5 conditional generation")
	}
}
