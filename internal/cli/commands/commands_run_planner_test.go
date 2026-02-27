package commands

import (
	"strings"
	"testing"
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
