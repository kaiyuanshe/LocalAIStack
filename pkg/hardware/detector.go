package hardware

import "github.com/zhuangbiaowei/LocalAIStack/internal/i18n"

type CPU struct {
	Arch      string
	Cores     int
	Threads   int
	ModelName string
	Vendor    string
}

type GPU struct {
	Index         int
	Name          string
	Vendor        string
	VRAMTotal     uint64
	VRAMFree      uint64
	CUDAVersion   string
	DriverVersion string
	MultiGPU      bool
	NVLink        bool
}

type Memory struct {
	Total     uint64
	Available uint64
	Free      uint64
}

type Storage struct {
	Path  string
	Total uint64
	Free  uint64
	Type  string
}

type HardwareProfile struct {
	CPU     CPU
	GPUs    []GPU
	Memory  Memory
	Storage []Storage
}

type Detector interface {
	DetectCPU() (CPU, error)
	DetectGPUs() ([]GPU, error)
	DetectMemory() (Memory, error)
	DetectStorage() ([]Storage, error)
	Detect() (*HardwareProfile, error)
}

type NativeDetector struct{}

func NewNativeDetector() Detector {
	return &NativeDetector{}
}

func (d *NativeDetector) DetectCPU() (CPU, error) {
	return CPU{
		Arch:      "x86_64",
		Cores:     8,
		Threads:   16,
		ModelName: "Placeholder",
		Vendor:    "Unknown",
	}, nil
}

func (d *NativeDetector) DetectGPUs() ([]GPU, error) {
	return []GPU{}, nil
}

func (d *NativeDetector) DetectMemory() (Memory, error) {
	return Memory{
		Total:     16 * 1024 * 1024 * 1024,
		Available: 8 * 1024 * 1024 * 1024,
		Free:      6 * 1024 * 1024 * 1024,
	}, nil
}

func (d *NativeDetector) DetectStorage() ([]Storage, error) {
	return []Storage{
		{
			Path:  "/",
			Total: 500 * 1024 * 1024 * 1024,
			Free:  300 * 1024 * 1024 * 1024,
			Type:  "ext4",
		},
	}, nil
}

func (d *NativeDetector) Detect() (*HardwareProfile, error) {
	cpu, err := d.DetectCPU()
	if err != nil {
		return nil, i18n.Errorf("failed to detect CPU: %w", err)
	}

	gpus, err := d.DetectGPUs()
	if err != nil {
		return nil, i18n.Errorf("failed to detect GPUs: %w", err)
	}

	memory, err := d.DetectMemory()
	if err != nil {
		return nil, i18n.Errorf("failed to detect memory: %w", err)
	}

	storage, err := d.DetectStorage()
	if err != nil {
		return nil, i18n.Errorf("failed to detect storage: %w", err)
	}

	return &HardwareProfile{
		CPU:     cpu,
		GPUs:    gpus,
		Memory:  memory,
		Storage: storage,
	}, nil
}
