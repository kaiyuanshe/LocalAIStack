package express

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
)

type HardwareFacts struct {
	OS               string    `json:"os"`
	Arch             string    `json:"arch"`
	CPUs             int       `json:"cpus"`
	RAMGB            int       `json:"ram_gb"`
	GPUs             []GPUFact `json:"gpus,omitempty"`
	BaseInfoPath     string    `json:"base_info_path,omitempty"`
	BaseInfoLoaded   bool      `json:"base_info_loaded"`
	DockerAvailable  bool      `json:"docker_available"`
	ComposeAvailable bool      `json:"compose_available"`
	MetalAvailable   bool      `json:"metal_available"`
}

type GPUFact struct {
	Vendor        string `json:"vendor"`
	Name          string `json:"name"`
	VRAMGB        int    `json:"vram_gb"`
	DriverVersion string `json:"driver_version,omitempty"`
}

func RefreshHardwareFacts() (HardwareFacts, error) {
	path, err := RefreshBaseInfo()
	if err != nil {
		return HardwareFacts{}, err
	}
	return LoadHardwareFacts(path)
}

func RefreshBaseInfo() (string, error) {
	if err := system.WriteBaseInfo("", "json", true, false); err != nil {
		return "", err
	}
	return defaultBaseInfoPath(), nil
}

func ReadBaseInfoRaw(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func LoadHardwareFacts(path string) (HardwareFacts, error) {
	summary, err := system.LoadBaseInfoSummary(path)
	if err != nil {
		return HardwareFacts{}, err
	}
	facts := HardwareFacts{
		OS:               firstNonEmpty(summary.OS, runtime.GOOS),
		Arch:             firstNonEmpty(summary.Arch, runtime.GOARCH),
		CPUs:             summary.CPUCores,
		RAMGB:            int(summary.MemoryKB / (1024 * 1024)),
		BaseInfoPath:     path,
		BaseInfoLoaded:   true,
		DockerAvailable:  summary.DockerAvailable,
		ComposeAvailable: summary.DockerComposeAvailable,
		MetalAvailable:   summary.MetalAvailable,
	}
	if facts.CPUs == 0 {
		facts.CPUs = runtime.NumCPU()
	}
	for i, name := range summary.GPUs {
		gpu := GPUFact{
			Vendor: guessGPUVendor(name),
			Name:   name,
		}
		if i < len(summary.GPUDetails) {
			detail := summary.GPUDetails[i]
			gpu.Vendor = firstNonEmpty(detail.Vendor, gpu.Vendor)
			gpu.Name = firstNonEmpty(detail.Name, gpu.Name)
			gpu.VRAMGB = detail.VRAMGB
			gpu.DriverVersion = detail.DriverVersion
		}
		if gpu.VRAMGB == 0 {
			gpu.VRAMGB = inferVRAMGBFromName(gpu.Name)
		}
		facts.GPUs = append(facts.GPUs, gpu)
	}
	return facts, nil
}

func DefaultBaseInfoPath() string {
	return defaultBaseInfoPath()
}

func (f HardwareFacts) JSON() ([]byte, error) {
	return json.MarshalIndent(f, "", "  ")
}

func defaultBaseInfoPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "base_info.json"
	}
	return filepath.Join(home, ".localaistack", "base_info.json")
}

func guessGPUVendor(name string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "nvidia") || strings.Contains(lower, "tesla") || strings.Contains(lower, "rtx") || strings.Contains(lower, "gtx") {
		return "nvidia"
	}
	if strings.Contains(lower, "apple") || strings.Contains(lower, "metal") {
		return "apple"
	}
	if strings.Contains(lower, "amd") || strings.Contains(lower, "radeon") {
		return "amd"
	}
	return ""
}

func inferVRAMGBFromName(name string) int {
	fields := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '(' || r == ')' || r == ','
	})
	for _, field := range fields {
		if strings.HasSuffix(field, "gb") {
			value := strings.TrimSuffix(field, "gb")
			if parsed, err := strconv.Atoi(value); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (f HardwareFacts) Summary() string {
	gpuSummary := "none"
	if len(f.GPUs) > 0 {
		parts := make([]string, 0, len(f.GPUs))
		for _, gpu := range f.GPUs {
			if gpu.VRAMGB > 0 {
				parts = append(parts, fmt.Sprintf("%s %dGB", gpu.Name, gpu.VRAMGB))
			} else {
				parts = append(parts, gpu.Name)
			}
		}
		gpuSummary = strings.Join(parts, "; ")
	}
	return fmt.Sprintf("os=%s arch=%s cpus=%d ram=%dGB gpu=%s docker=%t compose=%t metal=%t base_info=%s",
		f.OS, f.Arch, f.CPUs, f.RAMGB, gpuSummary, f.DockerAvailable, f.ComposeAvailable, f.MetalAvailable, f.BaseInfoPath)
}
