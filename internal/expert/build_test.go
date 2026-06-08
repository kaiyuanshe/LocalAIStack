package expert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildArtifactsWritesExecutableValidationAndBenchmarkFiles(t *testing.T) {
	plan := &ExpertPlan{
		Input: ExpertInput{
			Model:         "Qwen/Qwen3-4B-AWQ",
			Workload:      "openai-api",
			ContextLength: 8192,
			Concurrency:   1,
			Hardware: HardwareSpec{
				Vendor:   "nvidia",
				VRAMGB:   24,
				GPUModel: "RTX 4090",
			},
			Constraints: Constraints{
				LocalOnly: true,
				Docker:    true,
			},
		},
		ModelFacts: map[string]any{
			"family":          "qwen",
			"parameter_count": "4B",
			"format":          "safetensors",
			"quantization":    "awq",
		},
		HardwareFacts: map[string]any{
			"gpu_name": "RTX 4090",
			"vram_gb":  24,
		},
		Candidates: []CandidateRecipe{
			{
				Engine:     "vllm",
				Confidence: "high",
				Reason:     "test reason",
				Risks:      []string{"test risk"},
				Fallbacks:  []string{"reduce max_model_len"},
				Recipe: map[string]any{
					"max_model_len":          8192,
					"gpu_memory_utilization": 0.9,
					"tensor_parallel_size":   1,
					"enable_prefix_caching":  true,
				},
			},
		},
	}

	result, err := BuildArtifacts(plan, 0, t.TempDir())
	if err != nil {
		t.Fatalf("BuildArtifacts returned error: %v", err)
	}

	expected := []string{
		"recipe.yaml",
		"docker-compose.yaml",
		".env",
		"preflight.sh",
		"healthcheck.sh",
		"benchmark.sh",
		"benchmark-metadata.json",
	}
	for _, name := range expected {
		path := filepath.Join(result.PlanDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
		if strings.HasSuffix(name, ".sh") && info.Mode().Perm()&0111 == 0 {
			t.Fatalf("expected %s to be executable, mode=%s", name, info.Mode())
		}
	}

	benchmark, err := os.ReadFile(filepath.Join(result.PlanDir, "benchmark.sh"))
	if err != nil {
		t.Fatal(err)
	}
	compose, err := os.ReadFile(filepath.Join(result.PlanDir, "docker-compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(compose), "vllm/vllm-openai:v0.6.6") {
		t.Fatalf("vLLM compose should avoid latest CUDA image:\n%s", compose)
	}
	if !strings.Contains(string(compose), "VLLM_ENABLE_CUDA_COMPATIBILITY") {
		t.Fatalf("vLLM compose should enable CUDA compatibility:\n%s", compose)
	}
	if !strings.Contains(string(benchmark), "benchmark-results.jsonl") {
		t.Fatalf("benchmark script does not write local result store:\n%s", benchmark)
	}
	if plan.Candidates[0].Recipe["max_model_len"] == nil {
		t.Fatalf("expected build to preserve candidate engine config, got %#v", plan.Candidates[0].Recipe)
	}
	if plan.Candidates[0].Artifacts["compose_path"] == "" {
		t.Fatalf("expected build to record generated artifact paths")
	}

	metadata, err := os.ReadFile(filepath.Join(result.PlanDir, "benchmark-metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"recipe_id", "hardware_facts", "model_facts", "runtime_config"} {
		if !strings.Contains(string(metadata), want) {
			t.Fatalf("metadata missing %q:\n%s", want, metadata)
		}
	}
}
