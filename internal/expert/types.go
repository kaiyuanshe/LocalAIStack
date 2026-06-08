package expert

import (
	"time"
)

// ExpertInput represents user input for expert plan command.
type ExpertInput struct {
	// Hardware - "auto" to detect, or manual specification
	Hardware HardwareSpec `json:"hardware,omitempty"`
	// Model - HuggingFace model ID or "qwen/llama/deepseek + size"
	Model string `json:"model,omitempty"`
	// Workload type: openai-api, rag, agent, chat
	Workload string `json:"workload,omitempty"`
	// Context length: 8192, 16384, 32768, 65536
	ContextLength int `json:"context_length,omitempty"`
	// Concurrency: expected concurrent requests
	Concurrency int `json:"concurrency,omitempty"`
	// Constraints - user constraints
	Constraints Constraints `json:"constraints,omitempty"`
}

// HardwareSpec defines hardware specification.
type HardwareSpec struct {
	AutoDetect bool   `json:"auto_detect,omitempty"` // "auto"
	Vendor     string `json:"vendor,omitempty"`      // nvidia, apple, cpu
	VRAMGB     int    `json:"vram_gb,omitempty"`
	RAMGB      int    `json:"ram_gb,omitempty"`
	GPUModel   string `json:"gpu_model,omitempty"` // e.g. RTX 4090
}

// Constraints defines runtime constraints.
type Constraints struct {
	MaxVRAMGB int  `json:"max_vram_gb,omitempty"`
	LocalOnly bool `json:"local_only,omitempty"` // no cloud API fallback
	Docker    bool `json:"docker,omitempty"`     // prefer Docker
}

// CandidateRecipe represents a candidate recipe output.
type CandidateRecipe struct {
	// Engine: vllm, sglang, llamacpp
	Engine string `json:"engine"`
	// Confidence: high, medium, low
	Confidence string `json:"confidence"`
	// Reason: why this engine was recommended
	Reason string `json:"reason"`
	// Risks: potential issues
	Risks []string `json:"risks,omitempty"`
	// Fallbacks: suggestions when OOM or other issues
	Fallbacks []string `json:"fallbacks,omitempty"`
	// Recipe - the generated recipe YAML
	Recipe map[string]any `json:"recipe,omitempty"`
	// Artifacts - generated file paths for a built candidate
	Artifacts map[string]string `json:"artifacts,omitempty"`
	// BuildCommand - command to build artifacts
	BuildCommand string `json:"build_command,omitempty"`
	// Notes - additional notes
	Notes []string `json:"notes,omitempty"`
}

// ExpertPlan represents the result of expert plan.
type ExpertPlan struct {
	// Input - what user specified
	Input ExpertInput `json:"input"`
	// Candidates - generated candidates
	Candidates []CandidateRecipe `json:"candidates"`
	// HardwareFacts - detected or parsed hardware facts
	HardwareFacts map[string]any `json:"hardware_facts,omitempty"`
	// ModelFacts - detected model metadata
	ModelFacts map[string]any `json:"model_facts,omitempty"`
	// CreatedAt - when plan was created
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// LLMFitModel represents model info from llmfit.
type LLMFitModel struct {
	Name           string  `json:"name"`
	Provider       string  `json:"provider"`
	ParameterCount string  `json:"parameter_count"` // e.g. "32B"
	ParamsB        float64 `json:"params_b"`
	ContextLength  int     `json:"context_length"` // in tokens
	UseCase        string  `json:"use_case"`       // general, coding, reasoning, chat
	FitLevel       string  `json:"fit_level"`      // perfect, good, marginal, too_tight
	RunMode        string  `json:"run_mode"`       // gpu, cpu_offload, cpu_only
	Score          float64 `json:"score"`
	EstimatedTPS   float64 `json:"estimated_tps"`
	Runtime        string  `json:"runtime"` // llamacpp, mlx, vllm
	BestQuant      string  `json:"best_quant"`
	MemoryRequired float64 `json:"memory_required_gb"`
	UtilizationPct float64 `json:"utilization_pct"`
}

// LLMFitSystem represents system info from llmfit.
type LLMFitSystem struct {
	AvailableRAMGB  float64 `json:"available_ram_gb"`
	AvailableVRAMGB float64 `json:"available_vram_gb"`
	HasGPU          bool    `json:"has_gpu"`
	GPUName         string  `json:"gpu_name"`
	GPUVRAMGB       float64 `json:"gpu_vram_gb"`
	HasCUDA         bool    `json:"has_cuda"`
	HasMetal        bool    `json:"has_metal"`
	CPUName         string  `json:"cpu_name"`
	OS              string  `json:"os"`
	ARCH            string  `json:"arch"`
}

// EngineAdapter defines interface for generating engine-specific configs.
type EngineAdapter interface {
	// Name returns the engine name (vllm, sglang, llamacpp)
	Name() string
	// CanHandle returns true if this adapter can handle the given hardware/model
	CanHandle(system *LLMFitSystem, model *LLMFitModel) bool
	// Generate returns engine config for the given constraints
	Generate(system *LLMFitSystem, model *LLMFitModel, contextLength int, concurrency int) map[string]any
	// GenerateFallbacks returns suggested fallbacks when OOM
	GenerateFallbacks(system *LLMFitSystem, model *LLMFitModel) []string
}
