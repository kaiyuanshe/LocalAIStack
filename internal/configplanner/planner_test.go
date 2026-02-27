package configplanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
)

func TestBuildStaticPlanLlama(t *testing.T) {
	info := system.BaseInfoSummary{CPUCores: 12, MemoryKB: 64 * 1024 * 1024, GPUName: "NVIDIA RTX 4090 24GB"}
	plan, err := BuildStaticPlan("llama.cpp", "demo", info)
	if err != nil {
		t.Fatalf("BuildStaticPlan returned error: %v", err)
	}
	if plan.Source != "static" {
		t.Fatalf("expected source static, got %q", plan.Source)
	}
	if len(plan.Changes) < 3 {
		t.Fatalf("expected >=3 changes, got %d", len(plan.Changes))
	}
}

func TestBuildStaticPlanUnknownModule(t *testing.T) {
	_, err := BuildStaticPlan("unknown-module", "", system.BaseInfoSummary{})
	if err == nil {
		t.Fatalf("expected error for unknown module")
	}
}

func TestApplyPlan(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	plan := Plan{
		Module: "llama.cpp",
		Source: "static",
		Changes: []Change{
			{Scope: "model.run.llama.cpp", Key: "ctx_size", Value: 4096, Reason: "test"},
		},
	}
	path, err := ApplyPlan(plan)
	if err != nil {
		t.Fatalf("ApplyPlan returned error: %v", err)
	}
	if !strings.Contains(path, filepath.Join(".localaistack", "config-plans")) {
		t.Fatalf("unexpected plan path: %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected plan file to exist: %v", err)
	}
}
