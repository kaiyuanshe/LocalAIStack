package module

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type ModuleRecord struct {
	Manifest   Manifest
	Version    Version
	SourcePath string
	Checksum   string
	Signature  string
}

type Registry struct {
	records map[string][]ModuleRecord
}

func NewRegistry() *Registry {
	return &Registry{
		records: make(map[string][]ModuleRecord),
	}
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
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		record, err := LoadModuleRecord(path)
		if err != nil {
			return fmt.Errorf("load module manifest %s: %w", path, err)
		}
		if err := registry.Add(record); err != nil {
			return fmt.Errorf("register module %s: %w", record.Manifest.Name, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return registry, nil
}

func LoadModuleRecord(path string) (ModuleRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ModuleRecord{}, err
	}
	var manifest Manifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return ModuleRecord{}, err
	}
	if err := ValidateManifest(manifest); err != nil {
		return ModuleRecord{}, err
	}

	version, err := ParseVersion(manifest.Version)
	if err != nil {
		return ModuleRecord{}, err
	}

	checksum := ComputeChecksum(raw)
	if manifest.Integrity.Checksum != "" {
		expected := normalizeChecksum(manifest.Integrity.Checksum)
		if !strings.EqualFold(checksum, expected) {
			return ModuleRecord{}, fmt.Errorf("checksum mismatch for %s: expected %s got %s", manifest.Name, expected, checksum)
		}
	}

	return ModuleRecord{
		Manifest:   manifest,
		Version:    version,
		SourcePath: path,
		Checksum:   checksum,
		Signature:  manifest.Integrity.Signature,
	}, nil
}

func ComputeChecksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func normalizeChecksum(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "sha256:")
	trimmed = strings.TrimPrefix(trimmed, "SHA256:")
	return trimmed
}

func (r *Registry) Add(record ModuleRecord) error {
	if r.records == nil {
		r.records = make(map[string][]ModuleRecord)
	}
	name := record.Manifest.Name
	if name == "" {
		return fmt.Errorf("module name is required")
	}
	r.records[name] = append(r.records[name], record)
	sort.Slice(r.records[name], func(i, j int) bool {
		return r.records[name][i].Version.Compare(r.records[name][j].Version) > 0
	})
	return nil
}

func (r *Registry) Get(name string) []ModuleRecord {
	records := r.records[name]
	if len(records) == 0 {
		return nil
	}
	copied := make([]ModuleRecord, len(records))
	copy(copied, records)
	return copied
}

func (r *Registry) All() map[string][]ModuleRecord {
	result := make(map[string][]ModuleRecord, len(r.records))
	for name, records := range r.records {
		copied := make([]ModuleRecord, len(records))
		copy(copied, records)
		result[name] = copied
	}
	return result
}
