package module

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"gopkg.in/yaml.v3"
)

type installSpec struct {
	Uninstall uninstallSpec `yaml:"uninstall"`
}

type uninstallSpec struct {
	Script    string   `yaml:"script"`
	Preserves []string `yaml:"preserves"`
}

func Uninstall(name string) error {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return i18n.Errorf("module name is required")
	}
	moduleDir, err := resolveModuleDir(normalized)
	if err != nil {
		return err
	}

	planPath := filepath.Join(moduleDir, "INSTALL.yaml")
	raw, err := os.ReadFile(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return i18n.Errorf("install plan not found for module %q", normalized)
		}
		return i18n.Errorf("failed to read install plan for module %q: %w", normalized, err)
	}

	var spec installSpec
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		return i18n.Errorf("failed to parse install plan for module %q: %w", normalized, err)
	}
	script := strings.TrimSpace(spec.Uninstall.Script)
	if script == "" {
		return i18n.Errorf("module %q does not define an uninstall script", normalized)
	}

	scriptPath := script
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(moduleDir, scriptPath)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		if os.IsNotExist(err) {
			return i18n.Errorf("uninstall script not found for module %q", normalized)
		}
		return i18n.Errorf("failed to read uninstall script for module %q: %w", normalized, err)
	}

	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return i18n.Errorf("module %q uninstall failed: %w", normalized, err)
		}
		return i18n.Errorf("module %q uninstall failed: %s", normalized, message)
	}
	return nil
}
