package expert

import (
	"fmt"
	"math"
	"strings"
)

func AllAdapters() []EngineAdapter {
	return []EngineAdapter{
		&vLLMAdapter{},
		&SGLangAdapter{},
		&LlamaCppAdapter{},
	}
}

type vLLMAdapter struct{}

func (a *vLLMAdapter) Name() string { return "vllm" }

func (a *vLLMAdapter) CanHandle(system *LLMFitSystem, model *LLMFitModel) bool {
	if !system.HasGPU || !system.HasCUDA {
		return false
	}
	if model.Runtime != "" && !strings.EqualFold(model.Runtime, "vllm") {
		return model.ParamsB > 0
	}
	return system.AvailableVRAMGB > 0
}

func (a *vLLMAdapter) Generate(system *LLMFitSystem, model *LLMFitModel, contextLength int, concurrency int) map[string]any {
	maxModelLen := contextLength
	if maxModelLen == 0 {
		maxModelLen = 16384
	}
	if requiresVLLMHalfDtype(system) && system.AvailableVRAMGB <= 16 && maxModelLen > 8192 {
		maxModelLen = 8192
	}

	gpuMemUtil := 0.90
	if requiresVLLMHalfDtype(system) && system.AvailableVRAMGB <= 16 {
		gpuMemUtil = 0.85
	}
	if concurrency > 4 {
		gpuMemUtil = 0.88
	}
	if model.UtilizationPct > 80 {
		gpuMemUtil = math.Min(gpuMemUtil, 0.85)
	}

	config := map[string]any{
		"max_model_len":          maxModelLen,
		"gpu_memory_utilization": gpuMemUtil,
		"tensor_parallel_size":   1,
		"enable_prefix_caching":  true,
	}
	if requiresVLLMHalfDtype(system) {
		config["dtype"] = "half"
	}
	if requiresVLLMHalfDtype(system) && system.AvailableVRAMGB <= 16 {
		config["max_num_seqs"] = 1
	}

	if q := vLLMQuantization(model.BestQuant); q != "" {
		config["quantization"] = q
	}

	return config
}

func requiresVLLMHalfDtype(system *LLMFitSystem) bool {
	name := strings.ToLower(system.GPUName)
	switch {
	case strings.Contains(name, "v100"):
		return true
	case strings.Contains(name, "t4"):
		return true
	case strings.Contains(name, "tesla p"):
		return true
	case strings.Contains(name, "gtx"):
		return true
	default:
		return false
	}
}

func vLLMQuantization(quant string) string {
	switch strings.ToLower(strings.TrimSpace(quant)) {
	case "awq", "gptq", "squeezellm", "bitsandbytes", "fp8", "fbgemm_fp8", "modelopt", "gguf":
		return strings.ToLower(strings.TrimSpace(quant))
	default:
		return ""
	}
}

func (a *vLLMAdapter) GenerateFallbacks(system *LLMFitSystem, model *LLMFitModel) []string {
	fallbacks := []string{
		fmt.Sprintf("reduce max_model_len to %d", int(float64(model.ContextLength)*0.5)),
		"reduce gpu_memory_utilization to 0.80",
	}
	if model.ParamsB > 14 {
		fallbacks = append(fallbacks, fmt.Sprintf("switch to a %s7B model", modelFamily(model)))
	} else {
		fallbacks = append(fallbacks, "switch to a smaller model")
	}
	return fallbacks
}

type SGLangAdapter struct{}

func (a *SGLangAdapter) Name() string { return "sglang" }

func (a *SGLangAdapter) CanHandle(system *LLMFitSystem, model *LLMFitModel) bool {
	if !system.HasGPU || !system.HasCUDA {
		return false
	}
	if model.FitLevel == "too_tight" {
		return false
	}
	return system.AvailableVRAMGB >= 8
}

func (a *SGLangAdapter) Generate(system *LLMFitSystem, model *LLMFitModel, contextLength int, concurrency int) map[string]any {
	ctxLen := contextLength
	if ctxLen == 0 {
		ctxLen = 16384
	}

	gpuMemUtil := 0.90
	if model.UtilizationPct > 80 {
		gpuMemUtil = 0.85
	}

	return map[string]any{
		"host":                "0.0.0.0",
		"port":                30000,
		"context_length":      ctxLen,
		"enable_radix_cache":  true,
		"mem_fraction_static": gpuMemUtil,
	}
}

func (a *SGLangAdapter) GenerateFallbacks(system *LLMFitSystem, model *LLMFitModel) []string {
	return []string{
		"reduce context_length to 8192",
		"switch to vLLM (better memory efficiency on NVIDIA)",
	}
}

type LlamaCppAdapter struct{}

func (a *LlamaCppAdapter) Name() string { return "llamacpp" }

func (a *LlamaCppAdapter) CanHandle(system *LLMFitSystem, model *LLMFitModel) bool {
	return true
}

func (a *LlamaCppAdapter) Generate(system *LLMFitSystem, model *LLMFitModel, contextLength int, concurrency int) map[string]any {
	ctxSize := contextLength
	if ctxSize == 0 {
		ctxSize = 8192
	}

	config := map[string]any{
		"ctx_size":   ctxSize,
		"threads":    "auto",
		"batch_size": 512,
	}

	if system.HasGPU && system.HasCUDA {
		config["backend"] = "cuda"
		config["n_gpu_layers"] = gpuLayersForModel(model)
	} else if system.HasMetal {
		config["backend"] = "metal"
		config["n_gpu_layers"] = gpuLayersForModel(model)
	} else {
		config["backend"] = "cpu"
		config["threads"] = "auto"
	}

	return config
}

func (a *LlamaCppAdapter) GenerateFallbacks(system *LLMFitSystem, model *LLMFitModel) []string {
	fallbacks := []string{
		"reduce ctx_size to 4096",
		"switch to a lower quantization (e.g., Q4_K_M -> Q3_K_M)",
	}
	if system.HasGPU {
		fallbacks = append(fallbacks, "reduce n_gpu_layers and offload more to CPU")
	}
	return fallbacks
}

func gpuLayersForModel(model *LLMFitModel) int {
	if model.ParamsB >= 70 {
		return 80
	}
	if model.ParamsB >= 30 {
		return 60
	}
	return 99
}

func modelFamily(model *LLMFitModel) string {
	f := inferFamily(model.Name)
	if f == "unknown" {
		return ""
	}
	return strings.ToUpper(f[0:1]) + f[1:]
}
