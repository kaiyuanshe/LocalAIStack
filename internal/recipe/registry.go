package recipe

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"gopkg.in/yaml.v3"
)

type Record struct {
	Recipe     Recipe
	SourcePath string
	Checksum   string
}

type Registry struct {
	records map[string]Record
}

func NewRegistry() *Registry {
	return &Registry{records: make(map[string]Record)}
}

func FindRecipesRoot() (string, error) {
	roots := []string{"."}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		roots = append(roots, exeDir, filepath.Dir(exeDir))
	}
	for _, root := range roots {
		recipeDir := filepath.Join(root, "recipes")
		info, err := os.Stat(recipeDir)
		if err == nil {
			if info.IsDir() {
				return filepath.Abs(recipeDir)
			}
			continue
		}
		if !os.IsNotExist(err) {
			return "", i18n.Errorf("failed to access recipes directory: %w", err)
		}
	}
	return "", i18n.Errorf("recipes directory not found")
}

func LoadRegistryFromDir(root string) (*Registry, error) {
	registry := NewRegistry()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !isRecipeFile(path) {
			return nil
		}
		record, err := LoadRecord(path)
		if err != nil {
			return i18n.Errorf("load recipe %s: %w", path, err)
		}
		if err := registry.Add(record); err != nil {
			return i18n.Errorf("register recipe %s: %w", record.Recipe.ID, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return registry, nil
}

func LoadRecord(path string) (Record, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	var r Recipe
	if err := yaml.Unmarshal(raw, &r); err != nil {
		return Record{}, err
	}
	if err := Validate(r); err != nil {
		return Record{}, err
	}
	return Record{
		Recipe:     r,
		SourcePath: path,
		Checksum:   ComputeChecksum(raw),
	}, nil
}

func ComputeChecksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (r *Registry) Add(record Record) error {
	if r.records == nil {
		r.records = make(map[string]Record)
	}
	id := strings.TrimSpace(record.Recipe.ID)
	if id == "" {
		return i18n.Errorf("recipe id is required")
	}
	if _, exists := r.records[id]; exists {
		return i18n.Errorf("duplicate recipe id %q", id)
	}
	r.records[id] = record
	return nil
}

func (r *Registry) Get(id string) (Record, bool) {
	record, ok := r.records[id]
	return record, ok
}

func (r *Registry) All() []Record {
	records := make([]Record, 0, len(r.records))
	for _, record := range r.records {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Recipe.ID < records[j].Recipe.ID
	})
	return records
}

func isRecipeFile(path string) bool {
	base := filepath.Base(path)
	if base != "recipe.yaml" && base != "recipe.yml" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
