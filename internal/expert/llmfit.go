package expert

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func CallLLMFitRecommend_cmd(limit int, useCase string, minFit string) ([]LLMFitModel, *LLMFitSystem, error) {
	return callLLMFitRecommend(limit, useCase, minFit)
}

func CallLLMFitSystem_cmd() (*LLMFitSystem, error) {
	return callLLMFitSystem()
}

func callLLMFitRecommend(limit int, useCase string, minFit string) ([]LLMFitModel, *LLMFitSystem, error) {
	args := []string{"recommend", "--json", "--limit", strconv.Itoa(limit)}
	if useCase != "" {
		args = append(args, "--use-case", useCase)
	}
	if minFit != "" {
		args = append(args, "--min-fit", minFit)
	}

	out, err := exec.Command("llmfit", args...).Output()
	if err != nil {
		return nil, nil, fmt.Errorf(
			"llmfit recommend failed: %w. Is llmfit installed? Run: curl -fsSL https://llmfit.org/install.sh | bash",
			err,
		)
	}

	var models []llmfitRecommendEntry
	if err := json.Unmarshal(out, &models); err != nil {
		var wrapper llmfitRecommendWrapper
		if err2 := json.Unmarshal(out, &wrapper); err2 != nil {
			return nil, nil, fmt.Errorf(
				"failed to parse llmfit recommend output: %w (raw=%s)",
				err,
				string(out[:min(len(out), 200)]),
			)
		}
		models = wrapper.Models
	}

	result := make([]LLMFitModel, 0, len(models))
	for _, m := range models {
		result = append(result, m.toLLMFitModel())
	}

	sysInfo, err := callLLMFitSystem()
	if err != nil {
		sysInfo = &LLMFitSystem{}
	}

	return result, sysInfo, nil
}

func callLLMFitSystem() (*LLMFitSystem, error) {
	out, err := exec.Command("llmfit", "system", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("llmfit system failed: %w", err)
	}

	// Detect format: try new {"system": {...}} wrapper first
	var raw map[string]json.RawMessage
	if json.Unmarshal(out, &raw) == nil {
		if sysRaw, ok := raw["system"]; ok {
			// New format: fields are nested under "system"
			var sysV2 llmfitSystemOutputV2
			if err := json.Unmarshal(sysRaw, &sysV2); err != nil {
				return nil, fmt.Errorf("failed to parse llmfit system output (v2): %w", err)
			}
			return &LLMFitSystem{
				AvailableRAMGB:  sysV2.AvailableRAMGB,
				AvailableVRAMGB: sysV2.GPUVRAMGB,
				HasGPU:          sysV2.HasGPU,
				GPUName:         sysV2.GPUName,
				GPUVRAMGB:       sysV2.GPUVRAMGB,
				HasCUDA:         sysV2.Backend == "CUDA",
				HasMetal:        sysV2.Backend == "Metal" || sysV2.Backend == "METAL",
				CPUName:         sysV2.CPUName,
			}, nil
		}
	}

	// Fallback: old flat format
	var sys llmfitSystemOutput
	if err := json.Unmarshal(out, &sys); err != nil {
		return nil, fmt.Errorf("failed to parse llmfit system output: %w", err)
	}

	return &LLMFitSystem{
		AvailableRAMGB:  sys.RAMGB,
		AvailableVRAMGB: sys.VRAMGB,
		HasGPU:          sys.GPUCount > 0,
		GPUName:         sys.GPUName,
		GPUVRAMGB:       sys.VRAMGB,
		HasCUDA:         sys.HasCUDA,
		HasMetal:        sys.HasMetal,
		CPUName:         sys.CPUName,
		OS:              sys.OS,
		ARCH:            sys.Arch,
	}, nil
}

func callLLMFitPlan(modelID string, contextLength int) (string, error) {
	args := []string{"plan", modelID, "--context", strconv.Itoa(contextLength), "--json"}
	out, err := exec.Command("llmfit", args...).Output()
	if err != nil {
		return "", fmt.Errorf("llmfit plan failed: %w", err)
	}
	return string(out), nil
}

func IsLLMFitAvailable() bool {
	_, err := exec.LookPath("llmfit")
	return err == nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type llmfitRecommendWrapper struct {
	Models []llmfitRecommendEntry `json:"models"`
}

type llmfitRecommendEntry struct {
	Name           string  `json:"name"`
	Provider       string  `json:"provider"`
	ParameterCount string  `json:"parameter_count"`
	ParamsB        float64 `json:"params_b"`
	ContextLength  int     `json:"context_length"`
	UseCase        string  `json:"use_case"`
	FitLevel       string  `json:"fit_level"`
	RunMode        string  `json:"run_mode"`
	Score          float64 `json:"score"`
	EstimatedTPS   float64 `json:"estimated_tps"`
	Runtime        string  `json:"runtime"`
	BestQuant      string  `json:"best_quant"`
	MemoryRequired float64 `json:"memory_required_gb"`
	UtilizationPct float64 `json:"utilization_pct"`
	Quantization   string  `json:"quantization,omitempty"`
}

func (e llmfitRecommendEntry) toLLMFitModel() LLMFitModel {
	return LLMFitModel{
		Name:           e.Name,
		Provider:       e.Provider,
		ParameterCount: e.ParameterCount,
		ParamsB:        e.ParamsB,
		ContextLength:  e.ContextLength,
		UseCase:        e.UseCase,
		FitLevel:       e.FitLevel,
		RunMode:        e.RunMode,
		Score:          e.Score,
		EstimatedTPS:   e.EstimatedTPS,
		Runtime:        e.Runtime,
		BestQuant:      e.BestQuant,
		MemoryRequired: e.MemoryRequired,
		UtilizationPct: e.UtilizationPct,
	}
}

type llmfitSystemOutput struct {
	OS       string  `json:"os"`
	Arch     string  `json:"arch"`
	RAMGB    float64 `json:"ram_gb"`
	VRAMGB   float64 `json:"vram_gb"`
	GPUName  string  `json:"gpu_name"`
	GPUCount int     `json:"gpu_count"`
	CPUName  string  `json:"cpu_name"`
	HasCUDA  bool    `json:"has_cuda"`
	HasMetal bool    `json:"has_metal"`
}

type llmfitSystemOutputV2 struct {
	AvailableRAMGB float64 `json:"available_ram_gb"`
	CPUName        string  `json:"cpu_name"`
	GPUCount       int     `json:"gpu_count"`
	GPUName        string  `json:"gpu_name"`
	GPUVRAMGB      float64 `json:"gpu_vram_gb"`
	HasGPU         bool    `json:"has_gpu"`
	Backend        string  `json:"backend"`
}

func ResolveModelFacts(model *LLMFitModel) map[string]any {
	return map[string]any{
		"model_id":          model.Name,
		"provider":          model.Provider,
		"parameter_count":   model.ParameterCount,
		"params_b":          model.ParamsB,
		"context_length":    model.ContextLength,
		"use_case":          model.UseCase,
		"runtime":           model.Runtime,
		"recommended_quant": model.BestQuant,
		"estimated_tps":     model.EstimatedTPS,
		"memory_required":   model.MemoryRequired,
		"family":            inferFamily(model.Name),
		"format":            inferFormat(model.Name, model.Runtime),
		"quantization":      model.BestQuant,
	}
}

func inferFamily(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "qwen"):
		return "qwen"
	case strings.Contains(lower, "llama"):
		return "llama"
	case strings.Contains(lower, "deepseek"):
		return "deepseek"
	case strings.Contains(lower, "mistral"):
		return "mistral"
	case strings.Contains(lower, "mixtral"):
		return "mixtral"
	case strings.Contains(lower, "phi"):
		return "phi"
	case strings.Contains(lower, "gemma"):
		return "gemma"
	case strings.Contains(lower, "yi"):
		return "yi"
	default:
		return "unknown"
	}
}

func inferFormat(name, runtime string) string {
	lowerName := strings.ToLower(name)
	switch {
	case strings.Contains(lowerName, "gguf"):
		return "gguf"
	case strings.Contains(lowerName, "awq"):
		return "awq"
	case strings.Contains(lowerName, "gptq"):
		return "gptq"
	}
	switch strings.ToLower(runtime) {
	case "llamacpp", "llama.cpp", "llama_cpp":
		return "safetensors"
	case "vllm":
		return "safetensors"
	case "mlx":
		return "mlx"
	default:
		return "safetensors"
	}
}
