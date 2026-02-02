package modelmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
		if !entry.IsDir() {
			continue
		}

		modelPath := filepath.Join(m.modelDir, entry.Name())
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

func (m *Manager) DownloadModel(source ModelSource, modelID string, progress func(downloaded, total int64), opts DownloadOptions) error {
	provider, err := m.GetProvider(source)
	if err != nil {
		return err
	}

	if err := m.EnsureModelDir(); err != nil {
		return err
	}

	return provider.Download(context.Background(), modelID, m.modelDir, progress, opts)
}

func (m *Manager) GetModelInfo(source ModelSource, modelID string) (*ModelInfo, error) {
	provider, err := m.GetProvider(source)
	if err != nil {
		return nil, err
	}

	return provider.GetModelInfo(context.Background(), modelID)
}

func (m *Manager) GetModelSize(modelID string) (int64, error) {
	modelPath := filepath.Join(m.modelDir, modelID)

	var totalSize int64
	err := filepath.Walk(modelPath, func(path string, info os.FileInfo, err error) error {
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
