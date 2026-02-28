package configplanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
)

type Change struct {
	Scope  string `json:"scope"`
	Key    string `json:"key"`
	Value  any    `json:"value"`
	Reason string `json:"reason"`
}

type PlannerMeta struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Mode    string `json:"mode"`
}

type Context struct {
	CPUCores int    `json:"cpu_cores"`
	MemoryKB int64  `json:"memory_kb"`
	GPUName  string `json:"gpu_name,omitempty"`
	GPUCount int    `json:"gpu_count"`
}

type Plan struct {
	SchemaVersion string      `json:"schema_version"`
	Planner       PlannerMeta `json:"planner"`
	Module        string      `json:"module"`
	Model         string      `json:"model,omitempty"`
	Source        string      `json:"source"`
	Reason        string      `json:"reason"`
	GeneratedAt   string      `json:"generated_at"`
	Context       Context     `json:"context"`
	Changes       []Change    `json:"changes"`
	Warnings      []string    `json:"warnings,omitempty"`
}

func BuildStaticPlan(moduleName, modelID string, info system.BaseInfoSummary) (Plan, error) {
	name := strings.ToLower(strings.TrimSpace(moduleName))
	if name == "" {
		return Plan{}, fmt.Errorf("module name is required")
	}

	plan := Plan{
		SchemaVersion: "las.configplan/v0.1.0",
		Planner: PlannerMeta{
			Name:    "module-config-planner",
			Version: "p3-a",
			Mode:    "static",
		},
		Module:      name,
		Model:       strings.TrimSpace(modelID),
		Source:      "static",
		Reason:      "hardware-aware static planner",
		GeneratedAt: time.Now().Format(time.RFC3339),
		Context: Context{
			CPUCores: info.CPUCores,
			MemoryKB: info.MemoryKB,
			GPUName:  strings.TrimSpace(info.GPUName),
			GPUCount: info.GPUCount,
		},
	}

	switch name {
	case "llama.cpp":
		threads := info.CPUCores
		if threads <= 0 {
			threads = 4
		}
		ctxSize := 2048
		switch {
		case info.MemoryKB >= 64*1024*1024:
			ctxSize = 8192
		case info.MemoryKB >= 32*1024*1024:
			ctxSize = 4096
		}
		nGPULayers := estimateLlamaGPULayers(parseVRAMFromGPUName(info.GPUName))

		plan.Changes = append(plan.Changes,
			Change{Scope: "model.run.llama.cpp", Key: "threads", Value: threads, Reason: "match available CPU cores"},
			Change{Scope: "model.run.llama.cpp", Key: "ctx_size", Value: ctxSize, Reason: "fit system memory tier"},
			Change{Scope: "model.run.llama.cpp", Key: "n_gpu_layers", Value: nGPULayers, Reason: "fit detected GPU memory"},
		)
	case "vllm":
		vram := parseVRAMFromGPUName(info.GPUName)
		maxModelLen := 2048
		switch {
		case vram >= 80:
			maxModelLen = 32768
		case vram >= 48:
			maxModelLen = 24576
		case vram >= 24:
			maxModelLen = 16384
		case vram >= 16:
			maxModelLen = 8192
		case vram >= 12:
			maxModelLen = 6144
		case vram > 0:
			maxModelLen = 4096
		}
		gpuMemUtil := 0.88
		switch {
		case vram >= 80:
			gpuMemUtil = 0.92
		case vram >= 48:
			gpuMemUtil = 0.90
		case vram >= 24:
			gpuMemUtil = 0.88
		case vram >= 16:
			gpuMemUtil = 0.86
		case vram > 0:
			gpuMemUtil = 0.82
		default:
			gpuMemUtil = 0.0
		}

		plan.Changes = append(plan.Changes,
			Change{Scope: "model.run.vllm", Key: "max_model_len", Value: maxModelLen, Reason: "fit detected GPU/host memory"},
			Change{Scope: "model.run.vllm", Key: "gpu_memory_utilization", Value: gpuMemUtil, Reason: "keep memory pressure stable"},
		)
	case "ollama":
		parallel := 1
		if info.CPUCores >= 16 {
			parallel = 2
		}
		keepAlive := "10m"
		plan.Changes = append(plan.Changes,
			Change{Scope: "module.runtime.ollama", Key: "num_parallel", Value: parallel, Reason: "avoid oversubscription on host CPU"},
			Change{Scope: "module.runtime.ollama", Key: "keep_alive", Value: keepAlive, Reason: "reduce model reload overhead"},
		)
	default:
		return Plan{}, fmt.Errorf("no static config planner available for module %q", name)
	}

	return plan, nil
}

func ApplyPlan(plan Plan) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".localaistack", "config-plans")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s.json", strings.ReplaceAll(plan.Module, "/", "_"))
	target := filepath.Join(dir, filename)
	payload, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(target, payload, 0o600); err != nil {
		return "", err
	}
	return target, nil
}

func ResolveBaseInfoPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "base_info.json")
	}
	primary := filepath.Join(home, ".localaistack", "base_info.json")
	if _, err := os.Stat(primary); err == nil {
		return primary
	}
	alternate := filepath.Join(home, ".localiastack", "base_info.json")
	if _, err := os.Stat(alternate); err == nil {
		return alternate
	}
	return primary
}

func parseVRAMFromGPUName(name string) int {
	if strings.TrimSpace(name) == "" {
		return 0
	}
	re := regexp.MustCompile(`(?i)(\d+)\s*gb`)
	match := re.FindStringSubmatch(name)
	if len(match) < 2 {
		return 0
	}
	value := strings.TrimSpace(match[1])
	n := 0
	for i := 0; i < len(value); i++ {
		n = n*10 + int(value[i]-'0')
	}
	return n
}

func estimateLlamaGPULayers(vram int) int {
	switch {
	case vram >= 80:
		return 80
	case vram >= 48:
		return 60
	case vram >= 24:
		return 40
	case vram >= 16:
		return 20
	case vram >= 12:
		return 12
	case vram > 0:
		return 8
	default:
		return 0
	}
}
