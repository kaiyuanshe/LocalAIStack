package express

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhuangbiaowei/LocalAIStack/internal/recipe"
)

func TestBuildRunPlanUsesComposeArtifact(t *testing.T) {
	path := filepath.Join("..", "..", "recipes", "express", "inference", "nvidia", "qwen14b-vllm-nvidia24g", "recipe.yaml")
	record, err := recipe.LoadRecord(path)
	if err != nil {
		t.Fatalf("LoadRecord returned error: %v", err)
	}
	plan := BuildRunPlan(record)
	joined := strings.Join(plan.Command, " ")
	if plan.Mode != "docker-compose" {
		t.Fatalf("expected docker-compose mode, got %+v", plan)
	}
	if !strings.Contains(joined, "docker compose") || !strings.Contains(joined, "docker-compose.yaml") {
		t.Fatalf("unexpected command: %v", plan.Command)
	}
}
