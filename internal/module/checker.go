package module

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/llm"
)

func Check(name string) error {
	normalized := strings.ToLower(name)
	moduleDir, err := resolveModuleDir(normalized)
	if err != nil {
		return err
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	return runModuleCheck(name, moduleDir, cfg.LLM)
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

func runModuleCheck(name, moduleDir string, llmConfig config.LLMConfig) error {
	installSpecPath := filepath.Join(moduleDir, "INSTALL.yaml")
	installSpec, err := os.ReadFile(installSpecPath)
	if err != nil {
		if os.IsNotExist(err) {
			return i18n.Errorf("module %q missing INSTALL.yaml", name)
		}
		return i18n.Errorf("failed to read module config for %q: %w", name, err)
	}

	verifyScript, err := resolveVerifyScript(string(installSpec), llmConfig)
	if err != nil {
		return err
	}
	verifyScript = filepath.Clean(verifyScript)
	if !filepath.IsAbs(verifyScript) {
		verifyScript = filepath.Join(moduleDir, verifyScript)
	}
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

func resolveVerifyScript(spec string, llmConfig config.LLMConfig) (string, error) {
	registry, err := llm.NewRegistryFromConfig(llmConfig)
	if err != nil {
		return "", err
	}
	provider, err := registry.Provider(llmConfig.Provider)
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`You are reading an install specification YAML. Extract the verification script path.
Return only valid JSON in this exact shape: {"script":"<path>"}
YAML:
%s`, spec)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(llmConfig.TimeoutSeconds)*time.Second)
	defer cancel()

	resp, err := provider.Generate(ctx, llm.Request{
		Model:   llmConfig.Model,
		Prompt:  prompt,
		Timeout: llmConfig.TimeoutSeconds,
	})
	if err != nil {
		return "", err
	}

	script, err := parseVerifyScript(resp.Text)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(script) == "" {
		return "", i18n.Errorf("module verification script not found in config")
	}
	return script, nil
}

func parseVerifyScript(text string) (string, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return "", i18n.Errorf("invalid LLM response for verification script")
	}
	var payload struct {
		Script string `json:"script"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &payload); err != nil {
		return "", i18n.Errorf("invalid LLM response for verification script: %w", err)
	}
	return payload.Script, nil
}
