package module

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	moduleDir := filepath.Join("modules", name)
	manifestPath := filepath.Join(moduleDir, "manifest.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return "", i18n.Errorf("module %q not found", name)
		}
		return "", i18n.Errorf("failed to read module config for %q: %w", name, err)
	}
	return moduleDir, nil
}

func runModuleCheck(name, moduleDir string) error {
	verifyScript := filepath.Join(moduleDir, "verify.sh")
	if _, err := os.Stat(verifyScript); err != nil {
		if os.IsNotExist(err) {
			return i18n.Errorf("module %q does not provide a check script", name)
		}
		return i18n.Errorf("failed to read module check script for %q: %w", name, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", verifyScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return i18n.Errorf("module %q check failed: %w", name, err)
		}
		return i18n.Errorf("module %q check failed: %s", name, message)
	}
	return nil
}
