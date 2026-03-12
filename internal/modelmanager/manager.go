package modelmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (m *Manager) ListDownloadedModels() ([]DownloadedModel, error) {
	if err := m.EnsureModelDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(m.modelDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read model directory: %w", err)
	}

	var models []DownloadedModel
	for _, entry := range entries {
		modelPath := filepath.Join(m.modelDir, entry.Name())
		info, err := os.Stat(modelPath)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			continue
		}
		metadataPath := filepath.Join(modelPath, "metadata.json")

		metadata, err := os.ReadFile(metadataPath)
		if err != nil {
			continue
		}

		var model DownloadedModel
		if err := json.Unmarshal(metadata, &model); err != nil {
			continue
		}

		model.LocalPath = modelPath
		models = append(models, model)
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].DownloadedAt > models[j].DownloadedAt
	})

	return models, nil
}

func (m *Manager) RemoveModel(source ModelSource, modelID string) error {
	provider, err := m.GetProvider(source)
	if err != nil {
		return err
	}

	if err := provider.Delete(context.Background(), modelID); err != nil {
		return fmt.Errorf("failed to delete model from %s: %w", source, err)
	}

	modelPath := filepath.Join(m.modelDir, modelID)
	if _, err := os.Stat(modelPath); !os.IsNotExist(err) {
		if err := os.RemoveAll(modelPath); err != nil {
			return fmt.Errorf("failed to remove local metadata for %s: %w", modelID, err)
		}
	}

	return nil
}

func (m *Manager) GetModelPath(modelID string) (string, error) {
	modelPath := filepath.Join(m.modelDir, modelID)

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return "", fmt.Errorf("model %s not found", modelID)
	}

	return modelPath, nil
}

func (m *Manager) SearchAll(query string, limit int) (map[ModelSource][]ModelInfo, error) {
	results := make(map[ModelSource][]ModelInfo)

	for source, provider := range m.providers {
		models, err := provider.Search(context.Background(), query, limit)
		if err != nil {
			results[source] = []ModelInfo{}
			continue
		}
		results[source] = models
	}

	return results, nil
}

func (m *Manager) DownloadModel(source ModelSource, modelID string, progress func(downloaded, total int64), opts DownloadOptions) (ModelSource, error) {
	provider, err := m.GetProvider(source)
	if err != nil {
		return "", err
	}

	if err := m.EnsureModelDir(); err != nil {
		return "", err
	}

	err = provider.Download(context.Background(), modelID, m.modelDir, progress, opts)
	if err == nil {
		return source, nil
	}

	if source == SourceHuggingFace && opts.AllowModelScopeFallback && shouldFallbackToModelScope(err) {
		if fallbackErr := downloadModelWithModelScopeCLI(context.Background(), m.modelDir, modelID, opts); fallbackErr == nil {
			return SourceModelScope, nil
		} else {
			return "", fmt.Errorf("huggingface download failed and modelscope fallback also failed: %w", fallbackErr)
		}
	}

	return "", err
}

func shouldFallbackToModelScope(err error) bool {
	return errors.Is(err, ErrModelNotFound) || errors.Is(err, ErrSourceUnavailable)
}

func downloadModelWithModelScopeCLI(ctx context.Context, baseDir, modelID string, opts DownloadOptions) error {
	modelscopePath, err := exec.LookPath("modelscope")
	if err != nil {
		return fmt.Errorf("modelscope CLI not found in PATH: %w", err)
	}

	modelDir := filepath.Join(baseDir, strings.ReplaceAll(modelID, "/", "_"))
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create fallback model directory: %w", err)
	}

	args := []string{"download", "--model", modelID, "--local_dir", modelDir}
	if opts.FileHint != "" {
		args = append(args, "--include", opts.FileHint)
	}

	cmd := exec.CommandContext(ctx, modelscopePath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("modelscope download failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	metadata := map[string]interface{}{
		"id":            modelID,
		"source":        "modelscope",
		"downloaded_at": time.Now().Unix(),
	}

	metadataPath := filepath.Join(modelDir, "metadata.json")
	metadataFile, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer metadataFile.Close()

	encoder := json.NewEncoder(metadataFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

func (m *Manager) GetModelInfo(source ModelSource, modelID string) (*ModelInfo, error) {
	provider, err := m.GetProvider(source)
	if err != nil {
		return nil, err
	}

	return provider.GetModelInfo(context.Background(), modelID)
}

func (m *Manager) GetModelSize(modelID string) (int64, error) {
	modelPath, err := m.resolveModelStoragePath(modelID)
	if err != nil {
		return 0, err
	}

	var totalSize int64
	err = filepath.Walk(modelPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate model size: %w", err)
	}

	return totalSize, nil
}

func (m *Manager) resolveModelStoragePath(modelID string) (string, error) {
	candidates := []string{
		filepath.Join(m.modelDir, modelID),
		filepath.Join(m.modelDir, strings.ReplaceAll(modelID, "/", "_")),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("model %s not found", modelID)
}

func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func ParseModelID(input string) (ModelSource, string, error) {
	inputLower := strings.ToLower(input)

	if strings.HasPrefix(inputLower, "ollama:") {
		return SourceOllama, input[7:], nil
	}
	if strings.HasPrefix(inputLower, "huggingface:") {
		return SourceHuggingFace, input[12:], nil
	}
	if strings.HasPrefix(inputLower, "hf:") {
		return SourceHuggingFace, input[3:], nil
	}
	if strings.HasPrefix(inputLower, "modelscope:") {
		return SourceModelScope, input[11:], nil
	}

	if strings.Contains(input, ":") && !strings.Contains(input, "/") {
		return SourceOllama, input, nil
	}

	return SourceHuggingFace, input, nil
}
