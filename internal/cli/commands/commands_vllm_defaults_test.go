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
	envJoined := strings.Join(got.env, " ")
	if !strings.Contains(envJoined, "NCCL_IB_DISABLE=1") {
		t.Fatalf("expected NCCL_IB_DISABLE=1 in env, got %v", got.env)
	}
	if !strings.Contains(envJoined, "NCCL_P2P_DISABLE=1") {
		t.Fatalf("expected NCCL_P2P_DISABLE=1 in env, got %v", got.env)
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
