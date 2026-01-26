package module

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/llm"
	"gopkg.in/yaml.v3"
)

type moduleInstallSpec struct {
	InstallModes   []string                 `yaml:"install_modes"`
	DecisionMatrix installDecisionMatrix    `yaml:"decision_matrix"`
	Preconditions  []installPrecondition    `yaml:"preconditions"`
	Install        map[string][]installStep `yaml:"install"`
	Configuration  installConfiguration     `yaml:"configuration"`
}

type installDecisionMatrix struct {
	Default string `yaml:"default"`
}

type installConfiguration struct {
	Defaults map[string]any `yaml:"defaults"`
}

type installPrecondition struct {
	ID       string        `yaml:"id"`
	Intent   string        `yaml:"intent"`
	Tool     string        `yaml:"tool"`
	Command  string        `yaml:"command"`
	Expected installExpect `yaml:"expected"`
}

type installStep struct {
	ID         string        `yaml:"id"`
	Intent     string        `yaml:"intent"`
	Tool       string        `yaml:"tool"`
	Command    string        `yaml:"command"`
	Edit       installEdit   `yaml:"edit"`
	Expected   installExpect `yaml:"expected"`
	Idempotent bool          `yaml:"idempotent"`
}

type installEdit struct {
	Template    string `yaml:"template"`
	Destination string `yaml:"destination"`
}

type installExpect struct {
	Equals   string `yaml:"equals"`
	ExitCode *int   `yaml:"exit_code"`
	Bin      string `yaml:"bin"`
	Unit     string `yaml:"unit"`
	Service  string `yaml:"service"`
}

type llmInstallPlan struct {
	Mode  string   `json:"mode"`
	Steps []string `json:"steps"`
}

func Install(name string) error {
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

	var spec moduleInstallSpec
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		return i18n.Errorf("failed to parse install plan for module %q: %w", normalized, err)
	}

	if err := runPreconditions(spec.Preconditions, moduleDir); err != nil {
		return err
	}

	mode := selectInstallMode(spec)
	steps, ok := spec.Install[mode]
	if !ok || len(steps) == 0 {
		return i18n.Errorf("install plan for module %q has no steps for mode %q", normalized, mode)
	}

	planMode := mode
	planSteps := steps
	if cfg, cfgErr := config.LoadConfig(); cfgErr == nil {
		if llmPlan, err := interpretInstallPlanWithLLM(cfg.LLM, normalized, string(raw), mode, steps); err == nil {
			if strings.TrimSpace(llmPlan.Mode) != "" {
				planMode = strings.TrimSpace(llmPlan.Mode)
			}
			if planMode != mode {
				if modeSteps, ok := spec.Install[planMode]; ok && len(modeSteps) > 0 {
					steps = modeSteps
				}
			}
			planSteps = filterStepsByID(steps, llmPlan.Steps)
			if len(planSteps) == 0 {
				planSteps = steps
			}
		}
	}
	planSteps = ensureServiceSteps(planSteps, steps)

	vars := flattenDefaults(spec.Configuration.Defaults)
	for _, step := range planSteps {
		if err := runInstallStep(normalized, moduleDir, step, vars); err != nil {
			return err
		}
	}
	return nil
}

func runPreconditions(preconditions []installPrecondition, moduleDir string) error {
	for _, pre := range preconditions {
		tool := strings.TrimSpace(pre.Tool)
		if tool == "" {
			continue
		}
		switch tool {
		case "shell":
			output, exitCode, err := runShellCommand(pre.Command, moduleDir, false)
			if err != nil {
				return i18n.Errorf("precondition %s failed: %w", pre.ID, err)
			}
			if pre.Expected.ExitCode != nil && exitCode != *pre.Expected.ExitCode {
				return i18n.Errorf("precondition %s failed: expected exit code %d but got %d", pre.ID, *pre.Expected.ExitCode, exitCode)
			}
			if pre.Expected.Equals != "" {
				if strings.TrimSpace(output) != strings.TrimSpace(pre.Expected.Equals) {
					return i18n.Errorf("precondition %s failed: expected %q but got %q", pre.ID, pre.Expected.Equals, strings.TrimSpace(output))
				}
			}
		default:
			return i18n.Errorf("precondition %s uses unsupported tool %q", pre.ID, tool)
		}
	}
	return nil
}

func selectInstallMode(spec moduleInstallSpec) string {
	if strings.TrimSpace(spec.DecisionMatrix.Default) != "" {
		return strings.TrimSpace(spec.DecisionMatrix.Default)
	}
	if len(spec.InstallModes) > 0 {
		return strings.TrimSpace(spec.InstallModes[0])
	}
	for mode := range spec.Install {
		return mode
	}
	return ""
}

func interpretInstallPlanWithLLM(cfg config.LLMConfig, moduleName, installYAML, mode string, steps []installStep) (llmInstallPlan, error) {
	registry, err := llm.NewRegistryFromConfig(cfg)
	if err != nil {
		return llmInstallPlan{}, err
	}
	provider, err := registry.Provider(cfg.Provider)
	if err != nil {
		return llmInstallPlan{}, err
	}

	stepIDs := make([]string, 0, len(steps))
	for _, step := range steps {
		stepIDs = append(stepIDs, step.ID)
	}

	prompt := fmt.Sprintf(`You are installing module %s.
Given the following INSTALL.yaml, choose the install mode and step IDs to execute.
Only return JSON with keys "mode" and "steps". "steps" must be an array of step IDs.
Current selected mode: %s
Available step IDs: %s
INSTALL.yaml:
%s`, moduleName, mode, strings.Join(stepIDs, ", "), installYAML)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	resp, err := provider.Generate(ctx, llm.Request{Prompt: prompt, Model: cfg.Model, Timeout: cfg.TimeoutSeconds})
	if err != nil {
		return llmInstallPlan{}, err
	}

	parsed, err := parseLLMInstallPlan(resp.Text)
	if err != nil {
		return llmInstallPlan{}, err
	}
	if strings.TrimSpace(parsed.Mode) == "" {
		parsed.Mode = mode
	}
	return parsed, nil
}

func parseLLMInstallPlan(text string) (llmInstallPlan, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return llmInstallPlan{}, i18n.Errorf("LLM response did not include JSON")
	}
	payload := text[start : end+1]
	var plan llmInstallPlan
	if err := json.Unmarshal([]byte(payload), &plan); err != nil {
		return llmInstallPlan{}, err
	}
	return plan, nil
}

func filterStepsByID(steps []installStep, ids []string) []installStep {
	if len(ids) == 0 {
		return nil
	}
	allowed := make(map[string]bool, len(ids))
	for _, id := range ids {
		allowed[strings.TrimSpace(id)] = true
	}
	filtered := make([]installStep, 0, len(steps))
	for _, step := range steps {
		if allowed[step.ID] {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

func ensureServiceSteps(selected, all []installStep) []installStep {
	if len(all) == 0 {
		return selected
	}
	included := make(map[string]bool, len(selected))
	for _, step := range selected {
		included[step.ID] = true
	}
	for _, step := range all {
		if !included[step.ID] && isServiceStep(step) {
			selected = append(selected, step)
		}
	}
	return selected
}

func isServiceStep(step installStep) bool {
	return strings.TrimSpace(step.Expected.Unit) != "" || strings.TrimSpace(step.Expected.Service) != ""
}

func runInstallStep(moduleName, moduleDir string, step installStep, vars map[string]string) error {
	switch strings.TrimSpace(step.Tool) {
	case "shell":
		output, exitCode, err := runShellCommand(step.Command, moduleDir, true)
		if err != nil {
			return i18n.Errorf("install step %s failed: %w", step.ID, err)
		}
		if step.Expected.ExitCode != nil && exitCode != *step.Expected.ExitCode {
			return i18n.Errorf("install step %s failed: expected exit code %d but got %d", step.ID, *step.Expected.ExitCode, exitCode)
		}
		if step.Expected.Equals != "" {
			if strings.TrimSpace(output) != strings.TrimSpace(step.Expected.Equals) {
				return i18n.Errorf("install step %s failed: expected %q but got %q", step.ID, step.Expected.Equals, strings.TrimSpace(output))
			}
		}
	case "template":
		if err := runTemplateStep(moduleDir, step.Edit, vars); err != nil {
			return i18n.Errorf("install step %s failed: %w", step.ID, err)
		}
	default:
		return i18n.Errorf("install step %s uses unsupported tool %q", step.ID, step.Tool)
	}

	if err := validateExpected(moduleName, moduleDir, step.Expected); err != nil {
		return i18n.Errorf("install step %s failed: %w", step.ID, err)
	}
	return nil
}

func runShellCommand(command, moduleDir string, stream bool) (string, int, error) {
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = moduleDir
	if stream {
		var buffer bytes.Buffer
		writer := io.MultiWriter(&buffer, os.Stdout)
		cmd.Stdout = writer
		cmd.Stderr = writer
		err := cmd.Run()
		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		output := buffer.String()
		if err != nil {
			message := strings.TrimSpace(output)
			if message == "" {
				return "", exitCode, err
			}
			return message, exitCode, i18n.Errorf(message)
		}
		return output, exitCode, nil
	}

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", exitCode, err
		}
		return message, exitCode, i18n.Errorf(message)
	}
	return string(output), exitCode, nil
}

func runTemplateStep(moduleDir string, edit installEdit, vars map[string]string) error {
	templatePath := strings.TrimSpace(edit.Template)
	if templatePath == "" {
		return i18n.Errorf("template path is required")
	}
	if !filepath.IsAbs(templatePath) {
		templatePath = filepath.Join(moduleDir, templatePath)
	}
	contents, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}
	rendered, err := renderTemplate(string(contents), vars)
	if err != nil {
		return err
	}
	destPath := strings.TrimSpace(edit.Destination)
	if destPath == "" {
		return i18n.Errorf("template destination is required")
	}
	if !filepath.IsAbs(destPath) {
		destPath = filepath.Join(moduleDir, destPath)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destPath, []byte(rendered), 0o644)
}

func renderTemplate(content string, vars map[string]string) (string, error) {
	pattern := regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\|\s*default\("([^"]*)"\)\s*\}\}`)
	result := pattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := pattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		key := parts[1]
		fallback := parts[2]
		if value, ok := vars[key]; ok && strings.TrimSpace(value) != "" {
			return value
		}
		return fallback
	})

	simplePattern := regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)
	result = simplePattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := simplePattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		if value, ok := vars[parts[1]]; ok {
			return value
		}
		return ""
	})
	return result, nil
}

func flattenDefaults(defaults map[string]any) map[string]string {
	vars := make(map[string]string)
	for key, value := range defaults {
		if value == nil {
			continue
		}
		vars[key] = fmt.Sprint(value)
	}
	return vars
}

func validateExpected(moduleName, moduleDir string, expect installExpect) error {
	if expect.Bin != "" {
		if !filepath.IsAbs(expect.Bin) {
			return i18n.Errorf("expected bin path must be absolute")
		}
		if _, err := os.Stat(expect.Bin); err != nil {
			return err
		}
	}
	if expect.Unit != "" {
		unitPath := expect.Unit
		if !filepath.IsAbs(unitPath) {
			unitPath = filepath.Join(moduleDir, unitPath)
		}
		if _, err := os.Stat(unitPath); err != nil {
			return err
		}
	}
	if strings.TrimSpace(expect.Service) != "" {
		statusCmd := exec.Command("systemctl", "is-active", moduleName)
		output, err := statusCmd.CombinedOutput()
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(output)) != strings.TrimSpace(expect.Service) {
			return i18n.Errorf("expected service state %q but got %q", expect.Service, strings.TrimSpace(string(output)))
		}
	}
	return nil
}
