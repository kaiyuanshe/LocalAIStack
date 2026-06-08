package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/expert"
)

func TestExpertBuildCommandBuildsPlanJSON(t *testing.T) {
	planJSON := `{
  "input": {
    "hardware": {"vendor": "nvidia", "vram_gb": 24, "gpu_model": "RTX 4090"},
    "model": "Qwen/Qwen3-4B-AWQ",
    "workload": "openai-api",
    "context_length": 8192,
    "concurrency": 1,
    "constraints": {"local_only": true, "docker": true}
  },
  "candidates": [
    {
      "engine": "vllm",
      "confidence": "high",
      "reason": "NVIDIA CUDA serving path",
      "risks": ["verify generated benchmark before production use"],
      "fallbacks": ["reduce max_model_len to 4096"],
      "recipe": {
        "max_model_len": 8192,
        "gpu_memory_utilization": 0.9,
        "tensor_parallel_size": 1,
        "enable_prefix_caching": true
      }
    },
    {
      "engine": "sglang",
      "confidence": "medium",
      "reason": "agent workload candidate",
      "risks": ["VRAM may be tight under high concurrency"],
      "fallbacks": ["reduce context_length to 4096"],
      "recipe": {
        "context_length": 8192,
        "enable_radix_cache": true,
        "host": "0.0.0.0",
        "mem_fraction_static": 0.9,
        "port": 30000
      }
    }
  ],
  "hardware_facts": {"gpu_name": "RTX 4090", "vram_gb": 24},
  "model_facts": {"family": "qwen", "format": "safetensors", "parameter_count": "4B"}
}`

	tmp := t.TempDir()
	planPath := filepath.Join(tmp, "plan.json")
	if err := os.WriteFile(planPath, []byte(planJSON), 0644); err != nil {
		t.Fatal(err)
	}

	rootCmd := &cobra.Command{Use: "test"}
	RegisterExpertCommands(rootCmd)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"expert", "build", planPath, "--candidate", "2", "--output-dir", tmp})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\n%s", err, out.String())
	}

	output := out.String()
	if !strings.Contains(output, "Expert recipe built:") {
		t.Fatalf("unexpected output: %s", output)
	}
	if !strings.Contains(output, "benchmark-metadata.json") {
		t.Fatalf("expected benchmark metadata in output: %s", output)
	}

	matches, err := filepath.Glob(filepath.Join(tmp, "expert-qwen-qwen3-4b-awq-*", "benchmark.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one generated benchmark script, got %d", len(matches))
	}
}

func TestBuildRisksAlwaysReturnsAtLeastOneRisk(t *testing.T) {
	risks := buildRisks(
		&expert.LLMFitModel{FitLevel: "perfect", ParamsB: 4, ParameterCount: "4B"},
		"vllm",
		&expert.LLMFitSystem{HasGPU: true, HasCUDA: true, AvailableVRAMGB: 24, GPUName: "RTX 4090", GPUVRAMGB: 24},
		expert.ExpertInput{ContextLength: 8192},
	)
	if len(risks) == 0 {
		t.Fatal("expected at least one risk")
	}
}

func TestExpertUpCommandDryRunUsesGeneratedCompose(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	planDir := filepath.Join(home, ".localaistack", "expert", "expert-test")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"docker-compose.yaml": "services:\n  test:\n    image: busybox\n",
		".env":                "MODEL_ID=test\n",
		"preflight.sh":        "#!/usr/bin/env sh\nexit 0\n",
	} {
		if err := os.WriteFile(filepath.Join(planDir, name), []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}

	rootCmd := &cobra.Command{Use: "test"}
	RegisterExpertCommands(rootCmd)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"expert", "up", "expert-test", "--dry-run"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\n%s", err, out.String())
	}

	output := out.String()
	if !strings.Contains(output, "docker compose -f") {
		t.Fatalf("expected docker compose dry-run output: %s", output)
	}
	if !strings.Contains(output, filepath.Join(planDir, "docker-compose.yaml")) {
		t.Fatalf("expected generated compose path in output: %s", output)
	}
}
