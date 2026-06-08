package express

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHardwareFacts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "base_info.json")
	content := `{"os":"linux","arch":"amd64","cpu":{"model":"Intel Xeon","cores":36},"gpu":"Tesla V100-SXM2-16GB\nTesla V100-SXM2-16GB","gpu_details":[{"vendor":"nvidia","name":"Tesla V100-SXM2-16GB","vram_gb":16},{"vendor":"nvidia","name":"Tesla V100-SXM2-16GB","vram_gb":16}],"memory":"32691192 千字节","disk":{"total":"937.3 GB","available":"55.4 GB"},"runtime":{"docker_available":true,"docker_compose_available":true,"metal_available":false}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	facts, err := LoadHardwareFacts(path)
	if err != nil {
		t.Fatal(err)
	}

	if !facts.BaseInfoLoaded {
		t.Fatal("expected base info loaded")
	}
	if facts.CPUs != 36 {
		t.Fatalf("expected 36 CPUs, got %d", facts.CPUs)
	}
	if facts.RAMGB != 31 {
		t.Fatalf("expected 31GB RAM, got %d", facts.RAMGB)
	}
	if len(facts.GPUs) != 2 {
		t.Fatalf("expected 2 GPUs, got %+v", facts.GPUs)
	}
	if facts.GPUs[0].Vendor != "nvidia" {
		t.Fatalf("expected nvidia vendor, got %+v", facts.GPUs[0])
	}
	if facts.GPUs[0].VRAMGB != 16 {
		t.Fatalf("expected inferred 16GB VRAM, got %+v", facts.GPUs[0])
	}
	if !facts.DockerAvailable || !facts.ComposeAvailable {
		t.Fatalf("expected runtime facts from base_info, got %+v", facts)
	}
}
