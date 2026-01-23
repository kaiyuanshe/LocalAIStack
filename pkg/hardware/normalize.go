package hardware

type NormalizedProfile struct {
	CPUArch           string
	CPUCores          int
	CPUThreads        int
	GPUCount          int
	MaxGPUVRAMBytes   uint64
	TotalGPUVRAMBytes uint64
	HasNVLink         bool
	MultiGPU          bool
	MemoryTotalBytes  uint64
	StorageTotalBytes uint64
	StorageFreeBytes  uint64
}

func NormalizeProfile(profile *HardwareProfile) NormalizedProfile {
	if profile == nil {
		return NormalizedProfile{}
	}

	normalized := NormalizedProfile{
		CPUArch:          profile.CPU.Arch,
		CPUCores:         profile.CPU.Cores,
		CPUThreads:       profile.CPU.Threads,
		GPUCount:         len(profile.GPUs),
		MemoryTotalBytes: profile.Memory.Total,
	}

	var totalVRAM uint64
	var maxVRAM uint64
	var hasNVLink bool
	var multiGPU bool
	for _, gpu := range profile.GPUs {
		totalVRAM += gpu.VRAMTotal
		if gpu.VRAMTotal > maxVRAM {
			maxVRAM = gpu.VRAMTotal
		}
		if gpu.NVLink {
			hasNVLink = true
		}
		if gpu.MultiGPU {
			multiGPU = true
		}
	}

	if normalized.GPUCount > 1 {
		multiGPU = true
	}

	normalized.TotalGPUVRAMBytes = totalVRAM
	normalized.MaxGPUVRAMBytes = maxVRAM
	normalized.HasNVLink = hasNVLink
	normalized.MultiGPU = multiGPU

	for _, storage := range profile.Storage {
		normalized.StorageTotalBytes += storage.Total
		normalized.StorageFreeBytes += storage.Free
	}

	return normalized
}
