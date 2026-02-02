package modelmanager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type ModelFormat string

const (
	FormatGGUF        ModelFormat = "gguf"
	FormatSafetensors ModelFormat = "safetensors"
	FormatOllama      ModelFormat = "ollama"
	FormatUnknown     ModelFormat = "unknown"
)

type ModelSource string

const (
	SourceOllama      ModelSource = "ollama"
	SourceHuggingFace ModelSource = "huggingface"
	SourceModelScope  ModelSource = "modelscope"
)

type ModelInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Source      ModelSource       `json:"source"`
	Format      ModelFormat       `json:"format"`
	Size        int64             `json:"size"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
}

type DownloadedModel struct {
	ModelInfo
	LocalPath    string `json:"local_path"`
	DownloadedAt int64  `json:"downloaded_at"`
}

type Provider interface {
	Name() ModelSource
	Search(ctx context.Context, query string, limit int) ([]ModelInfo, error)
	Download(ctx context.Context, modelID string, destPath string, progress func(downloaded, total int64), opts DownloadOptions) error
	Delete(ctx context.Context, modelID string) error
	GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error)
}

type Manager struct {
	providers map[ModelSource]Provider
	modelDir  string
}

type DownloadOptions struct {
	FileHint string
}

func NewManager(modelDir string) *Manager {
	if modelDir == "" {
		home, _ := os.UserHomeDir()
		modelDir = filepath.Join(home, ".localaistack", "models")
	}
	return &Manager{
		providers: make(map[ModelSource]Provider),
		modelDir:  modelDir,
	}
}

func (m *Manager) RegisterProvider(provider Provider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}
	source := provider.Name()
	if _, exists := m.providers[source]; exists {
		return fmt.Errorf("provider %s already registered", source)
	}
	m.providers[source] = provider
	return nil
}

func (m *Manager) GetProvider(source ModelSource) (Provider, error) {
	provider, ok := m.providers[source]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", source)
	}
	return provider, nil
}

func (m *Manager) GetModelDir() string {
	return m.modelDir
}

func (m *Manager) EnsureModelDir() error {
	return os.MkdirAll(m.modelDir, 0755)
}

func DetectFormat(filename string) ModelFormat {
	ext := filepath.Ext(filename)
	switch ext {
	case ".gguf":
		return FormatGGUF
	case ".safetensors":
		return FormatSafetensors
	case ".bin":
		return FormatGGUF
	default:
		return FormatUnknown
	}
}
