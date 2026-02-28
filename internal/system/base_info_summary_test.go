package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBaseInfoSummary_StandardMarkdown(t *testing.T) {
	content := `# LocalAIStack Base Info

- Timestamp: 2026-02-13T09:00:00Z

## System Information

### CPU
- Model: Intel Xeon
- Cores: 32

### GPU
- GPU: Tesla V100-SXM2-16GB
Tesla V100-SXM2-16GB

### Memory
- Total: 12345678 kB
`

	path := writeTempBaseInfo(t, content)
	summary, err := LoadBaseInfoSummary(path)
	if err != nil {
		t.Fatalf("LoadBaseInfoSummary returned error: %v", err)
	}

	if summary.CPUCores != 32 {
		t.Fatalf("expected CPUCores=32, got %d", summary.CPUCores)
	}
	if summary.MemoryKB != 12345678 {
		t.Fatalf("expected MemoryKB=12345678, got %d", summary.MemoryKB)
	}
	if summary.GPUName != "Tesla V100-SXM2-16GB" {
		t.Fatalf("expected GPUName=Tesla V100-SXM2-16GB, got %q", summary.GPUName)
	}
	if summary.GPUCount != 2 {
		t.Fatalf("expected GPUCount=2, got %d", summary.GPUCount)
	}
}

func TestLoadBaseInfoSummary_MalformedInlineMarkdown(t *testing.T) {
	content := `# LocalAIStack Base Info- Timestamp: 2026-02-13T09:00:00Z## System Information### CPU- Cores: 36### GPU- GPU: Tesla V100-SXM2-16GB
Tesla V100-SXM2-16GB### Memory- Total: 32691208 kB### Disk- Total: 1 TB`

	path := writeTempBaseInfo(t, content)
	summary, err := LoadBaseInfoSummary(path)
	if err != nil {
		t.Fatalf("LoadBaseInfoSummary returned error: %v", err)
	}

	if summary.CPUCores != 36 {
		t.Fatalf("expected CPUCores=36, got %d", summary.CPUCores)
	}
	if summary.MemoryKB != 32691208 {
		t.Fatalf("expected MemoryKB=32691208, got %d", summary.MemoryKB)
	}
	if summary.GPUName != "Tesla V100-SXM2-16GB" {
		t.Fatalf("expected GPUName=Tesla V100-SXM2-16GB, got %q", summary.GPUName)
	}
	if summary.GPUCount != 2 {
		t.Fatalf("expected GPUCount=2, got %d", summary.GPUCount)
	}
}

func TestLoadBaseInfoSummary_LocalizedChineseMarkdown(t *testing.T) {
	content := `# LocalAIStack 基本信息- 时间戳：2026-02-13T21:47:02+08:00## 系统信息### CPU- 模型：Intel(R) Xeon(R) CPU E5-2686 v4 @ 2.30GHz
- 核心数量：36### GPU（图形处理器）- GPU：Tesla V100-SXM2-16GB
Tesla V100-SXM2-16GB### 内存- 总计：32691216 千字节## 原始命令的输出结果
processor : 0
processor : 1
processor : 2`

	path := writeTempBaseInfo(t, content)
	summary, err := LoadBaseInfoSummary(path)
	if err != nil {
		t.Fatalf("LoadBaseInfoSummary returned error: %v", err)
	}

	if summary.CPUCores != 36 {
		t.Fatalf("expected CPUCores=36, got %d", summary.CPUCores)
	}
	if summary.MemoryKB != 32691216 {
		t.Fatalf("expected MemoryKB=32691216, got %d", summary.MemoryKB)
	}
	if summary.GPUName != "Tesla V100-SXM2-16GB" {
		t.Fatalf("expected GPUName=Tesla V100-SXM2-16GB, got %q", summary.GPUName)
	}
	if summary.GPUCount != 2 {
		t.Fatalf("expected GPUCount=2, got %d", summary.GPUCount)
	}
}

func TestLoadBaseInfoSummary_CompactJSON(t *testing.T) {
	content := `{"cpu":{"model":"Intel Xeon","cores":24},"gpu":"Tesla V100-SXM2-16GB; Tesla V100-SXM2-16GB","memory":"33554432 kB","disk":{"total":"1.0 TB","available":"800.0 GB"}}`

	path := writeTempBaseInfo(t, content)
	summary, err := LoadBaseInfoSummary(path)
	if err != nil {
		t.Fatalf("LoadBaseInfoSummary returned error: %v", err)
	}

	if summary.CPUCores != 24 {
		t.Fatalf("expected CPUCores=24, got %d", summary.CPUCores)
	}
	if summary.MemoryKB != 33554432 {
		t.Fatalf("expected MemoryKB=33554432, got %d", summary.MemoryKB)
	}
	if summary.GPUName != "Tesla V100-SXM2-16GB" {
		t.Fatalf("expected GPUName=Tesla V100-SXM2-16GB, got %q", summary.GPUName)
	}
	if summary.GPUCount != 2 {
		t.Fatalf("expected GPUCount=2, got %d", summary.GPUCount)
	}
}

func writeTempBaseInfo(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "base_info.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp base_info.md: %v", err)
	}
	return path
}
