package module

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

func Check(name string) error {
	normalized := strings.ToLower(name)
	moduleDir, err := resolveModuleDir(normalized)
	if err != nil {
		return err
	}
	return runModuleCheck(name, moduleDir)
}

func resolveModuleDir(name string) (string, error) {
	roots := []string{"."}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		roots = append(roots, exeDir, filepath.Dir(exeDir))
	}
	for _, root := range roots {
		moduleDir := filepath.Join(root, "modules", name)
		manifestPath := filepath.Join(moduleDir, "manifest.yaml")
		if _, err := os.Stat(manifestPath); err == nil {
			absDir, err := filepath.Abs(moduleDir)
			if err != nil {
				return "", i18n.Errorf("failed to resolve module path for %q: %w", name, err)
			}
			return absDir, nil
		} else if !os.IsNotExist(err) {
			return "", i18n.Errorf("failed to read module config for %q: %w", name, err)
		}
	}
	return "", i18n.Errorf("module %q not found", name)
}

func runModuleCheck(name, moduleDir string) error {
	verifyScript := platformCheckScriptPath(moduleDir)
	if _, err := resolveModuleScriptPath(verifyScript); err != nil {
		if os.IsNotExist(err) {
			return i18n.Errorf("module %q does not provide a check script", name)
		}
		return i18n.Errorf("failed to read module check script for %q: %w", name, err)
	}

	output, err := runModuleScript(verifyScript, moduleDir, nil, nil, false)
	if err != nil {
		message := strings.TrimSpace(output)
		if message == "" {
			return i18n.Errorf("module %q check failed: %v", name, err)
		}
		return i18n.Errorf("module %q check failed: %s", name, message)
	}
	return nil
}
