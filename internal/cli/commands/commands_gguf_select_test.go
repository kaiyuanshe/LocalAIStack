package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zhuangbiaowei/LocalAIStack/internal/modelmanager"
)

func TestResolveGGUFFile_QuantSelectorPrefersFirstShard(t *testing.T) {
	modelDir := t.TempDir()
	q4Dir := filepath.Join(modelDir, "Q4_K_M")
	if err := os.MkdirAll(q4Dir, 0o755); err != nil {
		t.Fatalf("mkdir q4 dir: %v", err)
	}
	q8Dir := filepath.Join(modelDir, "Q8_0")
	if err := os.MkdirAll(q8Dir, 0o755); err != nil {
		t.Fatalf("mkdir q8 dir: %v", err)
	}

	filesToCreate := []string{
		filepath.Join(q4Dir, "Qwen3.5-122B-A10B-Q4_K_M-00002-of-00003.gguf"),
		filepath.Join(q4Dir, "Qwen3.5-122B-A10B-Q4_K_M-00001-of-00003.gguf"),
		filepath.Join(q4Dir, "Qwen3.5-122B-A10B-Q4_K_M-00003-of-00003.gguf"),
		filepath.Join(q8Dir, "Qwen3.5-122B-A10B-Q8_0.gguf"),
	}
	for _, f := range filesToCreate {
		if err := os.WriteFile(f, []byte("test"), 0o644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}

	ggufFiles, err := modelmanager.FindGGUFFiles(modelDir)
	if err != nil {
		t.Fatalf("find gguf: %v", err)
	}

	chosen, autoSelected, err := resolveGGUFFile(modelDir, ggufFiles, "Q4_K_M")
	if err != nil {
		t.Fatalf("resolve gguf: %v", err)
	}
	if autoSelected {
		t.Fatalf("expected manual selection mode")
	}
	want := filepath.Join(q4Dir, "Qwen3.5-122B-A10B-Q4_K_M-00001-of-00003.gguf")
	if chosen != want {
		t.Fatalf("unexpected chosen file: got %s, want %s", chosen, want)
	}
}

