package system

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/system/info"
)

func WriteBaseInfo(outputPath, format string, force, appendMode bool) error {
	resolvedPath, err := resolveOutputPath(outputPath)
	if err != nil {
		return err
	}
	if !force && !appendMode {
		defaultPath, err := resolveOutputPath("")
		if err != nil {
			return err
		}
		if resolvedPath == defaultPath {
			force = true
		}
	}
	if err := ensureWritable(resolvedPath, force, appendMode); err != nil {
		return err
	}

	report := info.CollectBaseInfo(context.Background())
	content, err := formatBaseInfo(report, format)
	if err != nil {
		return err
	}

	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC

	file, err := os.OpenFile(resolvedPath, flags, 0o644)
	if err != nil {
		return i18n.Errorf("open output file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return i18n.Errorf("write output file: %w", err)
	}

	return nil
}

func formatBaseInfo(report info.BaseInfo, format string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(format))
	if normalized != "" && normalized != "json" {
		return "", i18n.Errorf("unsupported format: %s", format)
	}
	payload := struct {
		CPU struct {
			Model string `json:"model"`
			Cores int    `json:"cores"`
		} `json:"cpu"`
		GPU    string `json:"gpu"`
		Memory string `json:"memory"`
		Disk   struct {
			Total     string `json:"total"`
			Available string `json:"available"`
		} `json:"disk"`
	}{}
	payload.CPU.Model = report.CPUModel
	payload.CPU.Cores = report.CPUCores
	payload.GPU = report.GPU
	payload.Memory = report.MemoryTotal
	payload.Disk.Total = report.DiskTotal
	payload.Disk.Available = report.DiskAvailable

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", i18n.Errorf("marshal json: %w", err)
	}
	return string(raw) + "\n", nil
}

func resolveOutputPath(path string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", i18n.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(home, ".localaistack")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", i18n.Errorf("create base directory: %w", err)
	}
	if path == "" {
		return filepath.Join(baseDir, "base_info.json"), nil
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		if path == "~" {
			return baseDir, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	if !filepath.IsAbs(path) {
		return filepath.Join(baseDir, path), nil
	}
	return path, nil
}

func ensureWritable(path string, force, appendMode bool) error {
	if path == "" {
		return i18n.Errorf("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return i18n.Errorf("create output directory: %w", err)
	}
	if appendMode {
		return nil
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return i18n.Errorf("output file exists: %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return i18n.Errorf("stat output file: %w", err)
		}
	}
	return nil
}
