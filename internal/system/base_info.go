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

	report, rawOutputs := info.CollectBaseInfoWithRaw(context.Background())
	content, err := formatBaseInfo(report, rawOutputs, format)
	if err != nil {
		return err
	}

	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

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

func formatBaseInfo(report info.BaseInfo, rawOutputs []info.RawCommandOutput, format string) (string, error) {
	switch strings.ToLower(format) {
	case "md", "markdown":
		return info.RenderBaseInfoMarkdown(report, rawOutputs), nil
	case "json":
		payload, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return "", i18n.Errorf("marshal json: %w", err)
		}
		return string(payload) + "\n", nil
	default:
		return "", i18n.Errorf("unsupported format: %s", format)
	}
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
		return filepath.Join(baseDir, "base_info.md"), nil
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
