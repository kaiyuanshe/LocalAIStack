package commands

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRecipeListCommand(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	RegisterRecipeCommands(rootCmd)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{
		"recipes",
		"--root",
		filepath.Join("..", "..", "..", "recipes"),
		"list",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\n%s", err, out.String())
	}
	output := out.String()
	if !strings.Contains(output, "express.nvidia.qwen14b.vllm.nvidia24g") {
		t.Fatalf("expected vLLM recipe in output: %s", output)
	}
	if !strings.Contains(output, "express.rag.local-docs.qdrant") {
		t.Fatalf("expected RAG recipe in output: %s", output)
	}
}

func TestRecipeValidateCommand(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	RegisterRecipeCommands(rootCmd)

	recipePath := filepath.Join("..", "..", "..", "recipes", "express", "rag", "local-docs-qdrant", "recipe.yaml")
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"recipes", "validate", recipePath})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Recipe express.rag.local-docs.qdrant is valid") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
