package system

import (
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type BaseInfoSummary struct {
	OS                     string
	Arch                   string
	CPUCores               int
	MemoryKB               int64
	GPUName                string
	GPUCount               int
	GPUs                   []string
	GPUDetails             []BaseInfoGPU
	DockerAvailable        bool
	DockerComposeAvailable bool
	MetalAvailable         bool
}

type BaseInfoGPU struct {
	Vendor        string `json:"vendor,omitempty"`
	Name          string `json:"name"`
	VRAMGB        int    `json:"vram_gb,omitempty"`
	DriverVersion string `json:"driver_version,omitempty"`
}

func LoadBaseInfoSummary(path string) (BaseInfoSummary, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return BaseInfoSummary{}, err
	}

	if summary, ok := parseJSONBaseInfoSummary(raw); ok {
		return summary, nil
	}

	var summary BaseInfoSummary

	content := string(raw)
	cpuSection := extractMarkdownSection(content, "CPU")
	memorySection := extractMarkdownSection(content, "Memory")
	gpuSection := extractMarkdownSection(content, "GPU")

	if cores, ok := extractFirstInt(cpuSection, `-\s*(?:Cores|核心数量)\s*[:：]\s*(\d+)`); ok {
		summary.CPUCores = cores
	}
	if summary.CPUCores == 0 {
		if cores, ok := extractFirstInt(content, `-\s*(?:Cores|核心数量)\s*[:：]\s*(\d+)`); ok {
			summary.CPUCores = cores
		}
	}

	if totalKB, ok := extractFirstInt64(memorySection, `-\s*(?:Total|总计)\s*[:：]\s*(\d+)\s*(?:kB|KB|千字节)`); ok {
		summary.MemoryKB = totalKB
	}
	if summary.MemoryKB == 0 {
		if totalKB, ok := extractFirstInt64(content, `-\s*(?:Total|总计)\s*[:：]\s*(\d+)\s*(?:kB|KB|千字节)`); ok {
			summary.MemoryKB = totalKB
		}
	}

	gpuEntries := extractGPUEntries(gpuSection)
	if len(gpuEntries) == 0 {
		gpuEntries = extractExplicitGPUEntries(content)
	}
	if len(gpuEntries) > 0 {
		summary.GPUName = gpuEntries[0]
		summary.GPUCount = len(gpuEntries)
		summary.GPUs = gpuEntries
	}

	if summary.GPUCount == 0 && summary.GPUName != "" {
		summary.GPUCount = 1
	}
	return summary, nil
}

func parseJSONBaseInfoSummary(raw []byte) (BaseInfoSummary, bool) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return BaseInfoSummary{}, false
	}

	var compact struct {
		OS   string `json:"os"`
		Arch string `json:"arch"`
		CPU  struct {
			Cores int `json:"cores"`
		} `json:"cpu"`
		GPU        string        `json:"gpu"`
		GPUDetails []BaseInfoGPU `json:"gpu_details"`
		Memory     string        `json:"memory"`
		Runtime    struct {
			DockerAvailable        bool `json:"docker_available"`
			DockerComposeAvailable bool `json:"docker_compose_available"`
			MetalAvailable         bool `json:"metal_available"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal(raw, &compact); err == nil {
		summary := BaseInfoSummary{
			OS:                     strings.TrimSpace(compact.OS),
			Arch:                   strings.TrimSpace(compact.Arch),
			CPUCores:               compact.CPU.Cores,
			MemoryKB:               parseMemoryToKB(compact.Memory),
			GPUDetails:             compact.GPUDetails,
			DockerAvailable:        compact.Runtime.DockerAvailable,
			DockerComposeAvailable: compact.Runtime.DockerComposeAvailable,
			MetalAvailable:         compact.Runtime.MetalAvailable,
		}
		if len(compact.GPUDetails) > 0 {
			entries := make([]string, 0, len(compact.GPUDetails))
			for _, gpu := range compact.GPUDetails {
				if strings.TrimSpace(gpu.Name) != "" {
					entries = append(entries, strings.TrimSpace(gpu.Name))
				}
			}
			setGPUEntries(&summary, entries)
		}
		if summary.GPUCount == 0 {
			setGPUSummary(&summary, compact.GPU)
		}
		if summary.CPUCores > 0 || summary.MemoryKB > 0 || summary.GPUCount > 0 || strings.TrimSpace(compact.GPU) != "" {
			return summary, true
		}
	}

	var legacy struct {
		CPUCores    int    `json:"cpu_cores"`
		GPU         string `json:"gpu"`
		MemoryTotal string `json:"memory_total"`
	}
	if err := json.Unmarshal(raw, &legacy); err == nil {
		summary := BaseInfoSummary{
			CPUCores: legacy.CPUCores,
			MemoryKB: parseMemoryToKB(legacy.MemoryTotal),
		}
		setGPUSummary(&summary, legacy.GPU)
		if summary.CPUCores > 0 || summary.MemoryKB > 0 || summary.GPUCount > 0 || strings.TrimSpace(legacy.GPU) != "" {
			return summary, true
		}
	}

	return BaseInfoSummary{}, false
}

func parseMemoryToKB(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	re := regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*([kmgt]?i?b|bytes?|千字节)`)
	match := re.FindStringSubmatch(trimmed)
	if len(match) < 3 {
		return 0
	}

	number, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToLower(strings.TrimSpace(match[2]))
	switch unit {
	case "b", "byte", "bytes":
		return int64(number / 1024)
	case "kb", "kib", "千字节":
		return int64(number)
	case "mb", "mib":
		return int64(number * 1024)
	case "gb", "gib":
		return int64(number * 1024 * 1024)
	case "tb", "tib":
		return int64(number * 1024 * 1024 * 1024)
	default:
		return 0
	}
}

func setGPUSummary(summary *BaseInfoSummary, raw string) {
	entries := parseGPUEntries(raw)
	if len(entries) == 0 {
		return
	}
	setGPUEntries(summary, entries)
}

func setGPUEntries(summary *BaseInfoSummary, entries []string) {
	if len(entries) == 0 {
		return
	}
	summary.GPUName = entries[0]
	summary.GPUCount = len(entries)
	summary.GPUs = entries
}

func parseGPUEntries(raw string) []string {
	candidates := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ';'
	})
	entries := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		addGPUEntry(&entries, candidate)
	}
	return entries
}

func extractMarkdownSection(content, sectionName string) string {
	pattern := `(?is)###\s*` + regexp.QuoteMeta(sectionName) + `\b([\s\S]*?)(?:\n###\s|\n##\s)`
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(content)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	// Fallback for malformed markdown where line breaks are missing.
	inlinePattern := `(?is)###\s*` + regexp.QuoteMeta(sectionName) + `\b([\s\S]*?)(?:###|##|\z)`
	inlineRE := regexp.MustCompile(inlinePattern)
	inlineMatch := inlineRE.FindStringSubmatch(content)
	if len(inlineMatch) >= 2 {
		return strings.TrimSpace(inlineMatch[1])
	}
	return ""
}

func extractFirstInt(content, pattern string) (int, bool) {
	re := regexp.MustCompile(`(?i)` + pattern)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return 0, false
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	return value, true
}

func extractFirstInt64(content, pattern string) (int64, bool) {
	re := regexp.MustCompile(`(?i)` + pattern)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return 0, false
	}
	value, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func extractGPUEntries(content string) []string {
	if strings.TrimSpace(content) == "" {
		return nil
	}

	entries := make([]string, 0)

	// Support normal markdown lines and malformed "### GPU- GPU: xxx" layout.
	bulletRe := regexp.MustCompile(`(?im)-\s*GPU(?:\s*\([^)]+\)|（[^）]+）)?\s*[:：]\s*([^\n#]+)`)
	for _, match := range bulletRe.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		addGPUEntry(&entries, match[1])
	}

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		if strings.EqualFold(trimmed, "GPU") {
			continue
		}
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "gpu:") || strings.Contains(trimmed, "GPU：") {
			continue
		}
		addGPUEntry(&entries, trimmed)
	}

	return entries
}

func extractExplicitGPUEntries(content string) []string {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	entries := make([]string, 0)
	bulletRe := regexp.MustCompile(`(?im)-\s*GPU(?:\s*\([^)]+\)|（[^）]+）)?\s*[:：]\s*([^\n#]+)`)
	for _, match := range bulletRe.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		addGPUEntry(&entries, match[1])
	}
	return entries
}

func addGPUEntry(entries *[]string, value string) {
	name := strings.TrimSpace(value)
	if idx := strings.Index(name, "###"); idx >= 0 {
		name = strings.TrimSpace(name[:idx])
	}
	if idx := strings.Index(name, "##"); idx >= 0 {
		name = strings.TrimSpace(name[:idx])
	}
	if name == "" || strings.HasPrefix(strings.ToLower(name), "unknown") {
		return
	}
	*entries = append(*entries, name)
}
