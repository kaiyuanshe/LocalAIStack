package module

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"gopkg.in/yaml.v3"
)

type updateSpec struct {
	Update uninstallSpec `yaml:"update"`
}

// Update upgrades an installed module. If no dedicated update script is
// provided in INSTALL.yaml, it falls back to Install().
func Update(name string) error {
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

	var spec updateSpec
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		return i18n.Errorf("failed to parse install plan for module %q: %w", normalized, err)
	}
	script := strings.TrimSpace(spec.Update.Script)
	if script == "" {
		return Install(normalized)
	}

	scriptPath := script
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(moduleDir, scriptPath)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		if os.IsNotExist(err) {
			return i18n.Errorf("update script not found for module %q", normalized)
		}
		return i18n.Errorf("failed to read update script for module %q: %w", normalized, err)
	}

	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = moduleDir
	cmd.Env = commandEnv(nil)
	var buffer bytes.Buffer
	writer := io.MultiWriter(&buffer, os.Stdout)
	cmd.Stdout = writer
	cmd.Stderr = writer
	err = cmd.Run()
	if err != nil {
		message := strings.TrimSpace(buffer.String())
		if message == "" {
			return i18n.Errorf("module %q update failed: %w", normalized, err)
		}
		return i18n.Errorf("module %q update failed: %s", normalized, message)
	}
	return nil
}
