package module

import "time"

type Category string

const (
	CategoryLanguage    Category = "language"
	CategoryRuntime     Category = "runtime"
	CategoryFramework   Category = "framework"
	CategoryService     Category = "service"
	CategoryApplication Category = "application"
	CategoryTool        Category = "tool"
	CategoryModel       Category = "model"
)

type State string

const (
	StateAvailable  State = "available"
	StateResolved   State = "resolved"
	StateInstalled  State = "installed"
	StateRunning    State = "running"
	StateStopped    State = "stopped"
	StateFailed     State = "failed"
	StateDeprecated State = "deprecated"
)

type Manifest struct {
	Name         string          `yaml:"name"`
	Category     Category        `yaml:"category"`
	Version      string          `yaml:"version"`
	Description  string          `yaml:"description"`
	License      string          `yaml:"license,omitempty"`
	Hardware     HardwareReq     `yaml:"hardware"`
	Dependencies Dependencies    `yaml:"dependencies"`
	Runtime      RuntimeConfig   `yaml:"runtime"`
	Interfaces   InterfaceConfig `yaml:"interfaces"`
	Integrity    Integrity       `yaml:"integrity,omitempty"`
}

type HardwareReq struct {
	CPU    *CPUReq    `yaml:"cpu,omitempty"`
	Memory *MemoryReq `yaml:"memory,omitempty"`
	GPU    *GPUReq    `yaml:"gpu,omitempty"`
}

type CPUReq struct {
	CoresMin int `yaml:"cores_min,omitempty"`
}

type MemoryReq struct {
	RAMMin string `yaml:"ram_min,omitempty"`
}

type GPUReq struct {
	VRAMMin  string `yaml:"vram_min,omitempty"`
	MultiGPU bool   `yaml:"multi_gpu,omitempty"`
}

type Dependencies struct {
	System  []string `yaml:"system,omitempty"`
	Modules []string `yaml:"modules,omitempty"`
	Runtime []string `yaml:"runtime,omitempty"`
}

type RuntimeConfig struct {
	Modes     []string `yaml:"modes"`
	Preferred string   `yaml:"preferred,omitempty"`
}

type InterfaceConfig struct {
	Provides []string `yaml:"provides,omitempty"`
	Consumes []string `yaml:"consumes,omitempty"`
}

type Integrity struct {
	Checksum  string `yaml:"checksum,omitempty"`
	Signature string `yaml:"signature,omitempty"`
}

type Module struct {
	Manifest    Manifest
	State       State
	InstalledAt time.Time
	Version     string
}
