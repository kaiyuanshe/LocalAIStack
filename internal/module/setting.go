package module

import (
	"os"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

// Setting runs module-specific post-install settings script with arguments.
func Setting(name string, args []string) error {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return i18n.Errorf("module name is required")
	}
	if len(args) == 0 {
		return i18n.Errorf("setting arguments are required")
	}

	moduleDir, err := resolveModuleDir(normalized)
	if err != nil {
		return err
	}

	scriptPath := platformSettingScriptPath(moduleDir)
	if _, err := resolveModuleScriptPath(scriptPath); err != nil {
		if os.IsNotExist(err) {
			return i18n.Errorf("setting script not found for module %q", normalized)
		}
		return i18n.Errorf("failed to read setting script for module %q: %w", normalized, err)
	}

	output, err := runModuleScript(scriptPath, moduleDir, args, nil, false)
	if err != nil {
		message := normalizedOutput(output)
		if message == "" {
			return i18n.Errorf("module %q setting failed: %w", normalized, err)
		}
		return i18n.Errorf("module %q setting failed: %s", normalized, message)
	}
	return nil
}
