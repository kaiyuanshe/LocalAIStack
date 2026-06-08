package recipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRecordInferenceRecipe(t *testing.T) {
	path := filepath.Join("..", "..", "recipes", "express", "inference", "nvidia", "qwen14b-vllm-nvidia24g", "recipe.yaml")
	record, err := LoadRecord(path)
	if err != nil {
		t.Fatalf("LoadRecord returned error: %v", err)
	}
	if record.Recipe.ID != "express.nvidia.qwen14b.vllm.nvidia24g" {
		t.Fatalf("unexpected recipe id: %s", record.Recipe.ID)
	}
	if record.Recipe.Kind != KindInference {
		t.Fatalf("unexpected recipe kind: %s", record.Recipe.Kind)
	}
	if record.Checksum == "" {
		t.Fatal("expected checksum")
	}
}

func TestLoadRegistryFromDir(t *testing.T) {
	root := filepath.Join("..", "..", "recipes")
	registry, err := LoadRegistryFromDir(root)
	if err != nil {
		t.Fatalf("LoadRegistryFromDir returned error: %v", err)
	}
	records := registry.All()
	if len(records) < 3 {
		t.Fatalf("expected at least 3 recipes, got %d", len(records))
	}
	if _, ok := registry.Get("express.rag.local-docs.qdrant"); !ok {
		t.Fatal("expected local docs rag recipe")
	}
}

func TestValidateRejectsInvalidRecipe(t *testing.T) {
	err := Validate(Recipe{
		ID:     "bad recipe",
		Kind:   KindInference,
		Tier:   TierExpress,
		Status: StatusExperimental,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	msg := err.Error()
	for _, want := range []string{"id must not contain spaces", "model.id is required", "validation.healthcheck is required"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in error %q", want, msg)
		}
	}
}

func TestLoadRegistryRejectsDuplicateID(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "a")
	dirB := filepath.Join(root, "b")
	if err := os.MkdirAll(dirA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirB, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`
id: duplicate.recipe
kind: application
tier: express
status: experimental
llm:
  recipe: express.nvidia.qwen14b.vllm.nvidia24g
embedding:
  model: BAAI/bge-m3
retrieval:
  vector_store: qdrant
`)
	if err := os.WriteFile(filepath.Join(dirA, "recipe.yaml"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "recipe.yaml"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRegistryFromDir(root); err == nil || !strings.Contains(err.Error(), "duplicate recipe id") {
		t.Fatalf("expected duplicate recipe id error, got %v", err)
	}
}
