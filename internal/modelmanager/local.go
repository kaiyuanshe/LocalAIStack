package modelmanager

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (m *Manager) ResolveLocalModelDir(source ModelSource, modelID string) (string, error) {
	modelDir := modelID
	switch source {
	case SourceHuggingFace, SourceModelScope:
		modelDir = strings.ReplaceAll(modelID, "/", "_")
	}

	modelPath := filepath.Join(m.modelDir, modelDir)
	if _, err := os.Stat(modelPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("model %s not found locally", modelID)
		}
		return "", err
	}

	return modelPath, nil
}

func FindGGUFFiles(modelPath string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(modelPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".gguf") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
