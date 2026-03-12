package modelmanager

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stubProvider struct {
	name        ModelSource
	downloadErr error
}

func (p *stubProvider) Name() ModelSource {
	return p.name
}

func (p *stubProvider) Search(ctx context.Context, query string, limit int) ([]ModelInfo, error) {
	return nil, nil
}

func (p *stubProvider) Download(ctx context.Context, modelID string, destPath string, progress func(downloaded, total int64), opts DownloadOptions) error {
	return p.downloadErr
}

func (p *stubProvider) Delete(ctx context.Context, modelID string) error {
	return nil
}

func (p *stubProvider) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
	return nil, nil
}

func TestManagerDownloadModel_FallsBackToModelScopeCLI(t *testing.T) {
	modelscopeDir := t.TempDir()
	modelscopePath := filepath.Join(modelscopeDir, "modelscope")
	script := "#!/bin/sh\n" +
		"local_dir=\"\"\n" +
		"while [ $# -gt 0 ]; do\n" +
		"  case \"$1\" in\n" +
		"    --local_dir)\n" +
		"      local_dir=\"$2\"\n" +
		"      shift 2\n" +
		"      ;;\n" +
		"    *)\n" +
		"      shift\n" +
		"      ;;\n" +
		"  esac\n" +
		"done\n" +
		"mkdir -p \"$local_dir\"\n" +
		"printf 'weights' > \"$local_dir/model.bin\"\n"
	if err := os.WriteFile(modelscopePath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create fake modelscope CLI: %v", err)
	}

	t.Setenv("PATH", modelscopeDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	modelDir := t.TempDir()
	mgr := NewManager(modelDir)
	if err := mgr.RegisterProvider(&stubProvider{
		name:        SourceHuggingFace,
		downloadErr: ErrModelNotFound,
	}); err != nil {
		t.Fatalf("failed to register huggingface provider: %v", err)
	}

	downloadedFrom, err := mgr.DownloadModel(SourceHuggingFace, "tclf90/Qwen3.5-27B-AWQ", nil, DownloadOptions{
		AllowModelScopeFallback: true,
	})
	if err != nil {
		t.Fatalf("DownloadModel returned error: %v", err)
	}
	if downloadedFrom != SourceModelScope {
		t.Fatalf("unexpected source, got %s want %s", downloadedFrom, SourceModelScope)
	}

	localPath := filepath.Join(modelDir, "tclf90_Qwen3.5-27B-AWQ")
	if _, err := os.Stat(filepath.Join(localPath, "model.bin")); err != nil {
		t.Fatalf("expected fallback download output in %s: %v", localPath, err)
	}

	metadata, err := os.ReadFile(filepath.Join(localPath, "metadata.json"))
	if err != nil {
		t.Fatalf("failed to read metadata: %v", err)
	}
	if !strings.Contains(string(metadata), `"source": "modelscope"`) {
		t.Fatalf("metadata does not record modelscope source: %s", string(metadata))
	}
}

func TestManagerDownloadModel_FallsBackWhenHuggingFaceIsUnavailable(t *testing.T) {
	modelscopeDir := t.TempDir()
	modelscopePath := filepath.Join(modelscopeDir, "modelscope")
	script := "#!/bin/sh\n" +
		"local_dir=\"\"\n" +
		"while [ $# -gt 0 ]; do\n" +
		"  case \"$1\" in\n" +
		"    --local_dir)\n" +
		"      local_dir=\"$2\"\n" +
		"      shift 2\n" +
		"      ;;\n" +
		"    *)\n" +
		"      shift\n" +
		"      ;;\n" +
		"  esac\n" +
		"done\n" +
		"mkdir -p \"$local_dir\"\n" +
		"printf 'weights' > \"$local_dir/model.bin\"\n"
	if err := os.WriteFile(modelscopePath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create fake modelscope CLI: %v", err)
	}

	t.Setenv("PATH", modelscopeDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	modelDir := t.TempDir()
	mgr := NewManager(modelDir)
	if err := mgr.RegisterProvider(&stubProvider{
		name:        SourceHuggingFace,
		downloadErr: ErrSourceUnavailable,
	}); err != nil {
		t.Fatalf("failed to register huggingface provider: %v", err)
	}

	downloadedFrom, err := mgr.DownloadModel(SourceHuggingFace, "tclf90/Qwen3.5-27B-AWQ", nil, DownloadOptions{
		AllowModelScopeFallback: true,
	})
	if err != nil {
		t.Fatalf("DownloadModel returned error: %v", err)
	}
	if downloadedFrom != SourceModelScope {
		t.Fatalf("unexpected source, got %s want %s", downloadedFrom, SourceModelScope)
	}
}

func TestModelScopeProviderDownload_UsesCLI(t *testing.T) {
	modelscopeDir := t.TempDir()
	modelscopePath := filepath.Join(modelscopeDir, "modelscope")
	script := "#!/bin/sh\n" +
		"local_dir=\"\"\n" +
		"while [ $# -gt 0 ]; do\n" +
		"  case \"$1\" in\n" +
		"    --local_dir)\n" +
		"      local_dir=\"$2\"\n" +
		"      shift 2\n" +
		"      ;;\n" +
		"    *)\n" +
		"      shift\n" +
		"      ;;\n" +
		"  esac\n" +
		"done\n" +
		"mkdir -p \"$local_dir\"\n" +
		"printf 'weights' > \"$local_dir/model.bin\"\n"
	if err := os.WriteFile(modelscopePath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create fake modelscope CLI: %v", err)
	}

	t.Setenv("PATH", modelscopeDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	dest := t.TempDir()
	provider := NewModelScopeProvider("")
	if err := provider.Download(context.Background(), "tclf90/Qwen3.5-27B-AWQ", dest, nil, DownloadOptions{}); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	localPath := filepath.Join(dest, "tclf90_Qwen3.5-27B-AWQ")
	if _, err := os.Stat(filepath.Join(localPath, "model.bin")); err != nil {
		t.Fatalf("expected CLI download output in %s: %v", localPath, err)
	}
}

func TestListDownloadedModels_IncludesDirectorySymlink(t *testing.T) {
	modelDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "model-cache")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	metadata := `{
  "id": "tclf90/Qwen3.5-27B-AWQ",
  "source": "modelscope",
  "downloaded_at": 1710000000
}`
	if err := os.WriteFile(filepath.Join(targetDir, "metadata.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "model.bin"), []byte("weights"), 0644); err != nil {
		t.Fatalf("failed to write model file: %v", err)
	}

	linkPath := filepath.Join(modelDir, "tclf90_Qwen3.5-27B-AWQ")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	mgr := NewManager(modelDir)
	models, err := mgr.ListDownloadedModels()
	if err != nil {
		t.Fatalf("ListDownloadedModels returned error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("unexpected model count, got %d want 1", len(models))
	}
	if models[0].ID != "tclf90/Qwen3.5-27B-AWQ" {
		t.Fatalf("unexpected model id: %s", models[0].ID)
	}

	size, err := mgr.GetModelSize(models[0].ID)
	if err != nil {
		t.Fatalf("GetModelSize returned error: %v", err)
	}
	if size == 0 {
		t.Fatalf("expected non-zero model size for symlinked model")
	}
}

func TestFindSafetensorsFiles_FollowsDirectorySymlink(t *testing.T) {
	targetDir := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "model-00001-of-00002.safetensors"), []byte("weights"), 0644); err != nil {
		t.Fatalf("failed to write safetensors file: %v", err)
	}

	linkPath := filepath.Join(t.TempDir(), "model-link")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	files, err := FindSafetensorsFiles(linkPath)
	if err != nil {
		t.Fatalf("FindSafetensorsFiles returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("unexpected safetensors file count, got %d want 1", len(files))
	}
}
