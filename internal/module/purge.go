package module

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"gopkg.in/yaml.v3"
)

type purgeSpec struct {
	SupportedPlatforms []string      `yaml:"supported_platforms"`
	Purge              uninstallSpec `yaml:"purge"`
}

// Purge runs the destructive cleanup script defined in INSTALL.yaml.
func Purge(name string) error {
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

	var spec purgeSpec
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		return i18n.Errorf("failed to parse install plan for module %q: %w", normalized, err)
	}
	if err := ensurePlatformSupported(spec.SupportedPlatforms); err != nil {
		return err
	}
	script := strings.TrimSpace(spec.Purge.Script)
	if script == "" {
		return i18n.Errorf("module %q does not define a purge script", normalized)
	}

	scriptPath := absoluteModuleScriptPath(moduleDir, script)
	if _, err := os.Stat(scriptPath); err != nil {
		if _, resolveErr := resolveModuleScriptPath(scriptPath); os.IsNotExist(err) && resolveErr != nil {
			return i18n.Errorf("purge script not found for module %q", normalized)
		}
		return i18n.Errorf("failed to read purge script for module %q: %w", normalized, err)
	}

	output, err := runModuleScript(scriptPath, moduleDir, nil, nil, false)
	if err != nil {
		message := strings.TrimSpace(output)
		if message == "" {
			return i18n.Errorf("module %q purge failed: %w", normalized, err)
		}
		return i18n.Errorf("module %q purge failed: %s", normalized, message)
	}
	return nil
}
