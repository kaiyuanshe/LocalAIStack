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
	"sort"
	"strings"
	"time"

	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/failure"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/llm"
	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
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
	Mode         string   `json:"mode"`
	Steps        []string `json:"steps"`
	Reason       string   `json:"reason,omitempty"`
	RiskLevel    string   `json:"risk_level,omitempty"`
	FallbackHint string   `json:"fallback_hint,omitempty"`
}

type installPlannerInput struct {
	ModuleName         string                        `json:"module_name"`
	CurrentMode        string                        `json:"current_mode"`
	AvailableModes     []string                      `json:"available_modes"`
	CurrentModeSteps   []installPlannerStepHint      `json:"current_mode_steps"`
	CurrentModeSummary installPlannerCategorySummary `json:"current_mode_summary"`
	ModeCatalog        []installPlannerModeHint      `json:"mode_catalog"`
	Preconditions      []installPlannerConditionHint `json:"preconditions,omitempty"`
	PlannerVersion     string                        `json:"planner_version"`
}

type installPlannerStepHint struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Tool     string `json:"tool"`
	Intent   string `json:"intent,omitempty"`
	Command  string `json:"command,omitempty"`
}

type installPlannerCategorySummary struct {
	Dependency    int `json:"dependency"`
	Download      int `json:"download"`
	BinaryInstall int `json:"binary_install"`
	SourceBuild   int `json:"source_build"`
	Configure     int `json:"configure"`
	Service       int `json:"service"`
	Verify        int `json:"verify"`
}

type installPlannerModeHint struct {
	Mode     string                        `json:"mode"`
	Steps    []installPlannerStepHint      `json:"steps"`
	Summary  installPlannerCategorySummary `json:"summary"`
	StepIDs  []string                      `json:"step_ids"`
	StepSize int                           `json:"step_size"`
}

type installPlannerConditionHint struct {
	ID      string `json:"id"`
	Intent  string `json:"intent,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Command string `json:"command,omitempty"`
}

func Install(name string) (retErr error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return i18n.Errorf("module name is required")
	}
	plannerProvider := ""
	plannerModel := ""
	defer func() {
		if retErr == nil {
			return
		}
		failure.RecordBestEffort(failure.Event{
			Phase:    inferInstallFailurePhase(retErr),
			Module:   normalized,
			Model:    plannerModel,
			Provider: plannerProvider,
			Error:    retErr.Error(),
			Message:  "module install failed",
			Context: map[string]any{
				"entry": "module.Install",
			},
		})
	}()
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

	mode, env := selectInstallModeForSystem(normalized, spec)
	steps, ok := spec.Install[mode]
	if !ok || len(steps) == 0 {
		return i18n.Errorf("install plan for module %q has no steps for mode %q", normalized, mode)
	}

	planSteps := steps
	plannerSource := "static"
	plannerErr := ""
	plannerMode := mode
	if cfg, cfgErr := config.LoadConfig(); cfgErr == nil {
		plannerProvider = strings.TrimSpace(cfg.LLM.Provider)
		plannerModel = strings.TrimSpace(cfg.LLM.Model)
		if llmPlan, err := interpretInstallPlanWithLLM(cfg.LLM, normalized, spec, mode, steps); err == nil {
			resolvedMode, resolvedSteps, applyErr := applyLLMInstallPlan(spec, mode, steps, env, llmPlan)
			if applyErr == nil {
				plannerMode = resolvedMode
				planSteps = resolvedSteps
				plannerSource = "llm"
			} else {
				plannerErr = applyErr.Error()
			}
		} else {
			plannerErr = err.Error()
		}
	} else {
		plannerErr = cfgErr.Error()
	}
	planSteps = ensureServiceSteps(planSteps, steps)
	if isInstallPlannerDebugEnabled() {
		fmt.Printf("Install planner: source=%s mode=%s steps=%s\n", plannerSource, plannerMode, strings.Join(stepIDs(planSteps), ","))
		if strings.TrimSpace(plannerErr) != "" {
			fmt.Printf("Install planner fallback reason: %s\n", plannerErr)
		}
	}
	if isInstallPlannerStrictEnabled() && plannerSource != "llm" {
		if strings.TrimSpace(plannerErr) == "" {
			plannerErr = "LLM install planner did not produce a valid plan"
		}
		return i18n.Errorf("install planner strict mode: %s", plannerErr)
	}

	vars := flattenDefaults(spec.Configuration.Defaults)
	for _, step := range planSteps {
		if err := runInstallStep(normalized, moduleDir, step, vars, env); err != nil {
			return err
		}
	}
	return nil
}

func inferInstallFailurePhase(err error) string {
	if err == nil {
		return failure.PhaseModuleInstall
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(message, "install planner") {
		return failure.PhaseInstallPlanner
	}
	return failure.PhaseModuleInstall
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
				if normalizedOutput(output) != normalizedOutput(pre.Expected.Equals) {
					return i18n.Errorf("precondition %s failed: expected %q but got %q", pre.ID, normalizedOutput(pre.Expected.Equals), normalizedOutput(output))
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

func selectInstallModeForSystem(moduleName string, spec moduleInstallSpec) (string, map[string]string) {
	mode := selectInstallMode(spec)
	env := map[string]string{}

	if moduleName != "llama.cpp" {
		return mode, env
	}

	baseInfoPath := resolveBaseInfoPath()
	baseInfo, err := system.LoadBaseInfoSummary(baseInfoPath)
	if err != nil {
		return mode, env
	}

	if strings.TrimSpace(baseInfo.GPUName) != "" {
		mode = "source"
		env["LLAMA_CUDA"] = "1"
		if archs := detectCudaArchs(baseInfo.GPUName); archs != "" {
			env["LLAMA_CUDA_ARCHS"] = archs
		}
	}

	return mode, env
}

func interpretInstallPlanWithLLM(cfg config.LLMConfig, moduleName string, spec moduleInstallSpec, mode string, steps []installStep) (llmInstallPlan, error) {
	registry, err := llm.NewRegistryFromConfig(cfg)
	if err != nil {
		return llmInstallPlan{}, err
	}
	provider, err := registry.Provider(cfg.Provider)
	if err != nil {
		return llmInstallPlan{}, err
	}

	input := buildInstallPlannerInput(moduleName, spec, mode, steps)
	inputPayload, err := json.Marshal(input)
	if err != nil {
		return llmInstallPlan{}, err
	}

	prompt := fmt.Sprintf(`You are an install planner for LocalAIStack.
Only return valid JSON and nothing else.
Required JSON schema:
{"mode":"<mode>","steps":["<step-id>"],"reason":"<short reason>","risk_level":"low|medium|high","fallback_hint":"<optional>"}
Rules:
- mode must be one of available_modes.
- steps must only contain IDs listed for that selected mode.
- preserve service-related steps when relevant.
- prefer safe, idempotent execution.
- prioritize complete install path if applicable: dependency -> download -> binary_install or source_build -> configure -> verify.
Planner input:
%s`, string(inputPayload))

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
	payload := extractFirstJSONObject(text)
	if payload == "" {
		return llmInstallPlan{}, i18n.Errorf("LLM response did not include JSON")
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return llmInstallPlan{}, err
	}

	var plan llmInstallPlan
	_ = json.Unmarshal(raw["mode"], &plan.Mode)
	if err := json.Unmarshal(raw["steps"], &plan.Steps); err != nil {
		var selected []string
		if aliasErr := json.Unmarshal(raw["selected_steps"], &selected); aliasErr == nil {
			plan.Steps = selected
		} else {
			return llmInstallPlan{}, err
		}
	}
	_ = json.Unmarshal(raw["reason"], &plan.Reason)
	_ = json.Unmarshal(raw["risk_level"], &plan.RiskLevel)
	_ = json.Unmarshal(raw["fallback_hint"], &plan.FallbackHint)
	plan.Mode = strings.TrimSpace(plan.Mode)
	plan.Steps = dedupeStepIDs(plan.Steps)
	plan.RiskLevel = normalizeRiskLevel(plan.RiskLevel)
	return plan, nil
}

func buildInstallPlannerInput(moduleName string, spec moduleInstallSpec, mode string, steps []installStep) installPlannerInput {
	availableModes := collectAvailableInstallModes(spec)
	hints := make([]installPlannerStepHint, 0, len(steps))
	for _, step := range steps {
		hints = append(hints, installPlannerStepHint{
			ID:       strings.TrimSpace(step.ID),
			Category: classifyInstallStep(step),
			Tool:     strings.TrimSpace(step.Tool),
			Intent:   strings.TrimSpace(step.Intent),
			Command:  strings.TrimSpace(step.Command),
		})
	}
	return installPlannerInput{
		ModuleName:         moduleName,
		CurrentMode:        mode,
		AvailableModes:     availableModes,
		CurrentModeSteps:   hints,
		CurrentModeSummary: summarizePlannerStepHints(hints),
		ModeCatalog:        buildInstallPlannerModeCatalog(spec),
		Preconditions:      buildPlannerPreconditionHints(spec.Preconditions),
		PlannerVersion:     "p1.1",
	}
}

func collectAvailableInstallModes(spec moduleInstallSpec) []string {
	modeSet := make(map[string]bool)
	for _, mode := range spec.InstallModes {
		trimmed := strings.TrimSpace(mode)
		if trimmed != "" {
			modeSet[trimmed] = true
		}
	}
	for mode := range spec.Install {
		trimmed := strings.TrimSpace(mode)
		if trimmed != "" {
			modeSet[trimmed] = true
		}
	}
	modes := make([]string, 0, len(modeSet))
	for mode := range modeSet {
		modes = append(modes, mode)
	}
	sort.Strings(modes)
	return modes
}

func buildInstallPlannerModeCatalog(spec moduleInstallSpec) []installPlannerModeHint {
	modes := collectAvailableInstallModes(spec)
	catalog := make([]installPlannerModeHint, 0, len(modes))
	for _, mode := range modes {
		modeSteps := spec.Install[mode]
		hints := make([]installPlannerStepHint, 0, len(modeSteps))
		stepIDs := make([]string, 0, len(modeSteps))
		for _, step := range modeSteps {
			hint := installPlannerStepHint{
				ID:       strings.TrimSpace(step.ID),
				Category: classifyInstallStep(step),
				Tool:     strings.TrimSpace(step.Tool),
				Intent:   strings.TrimSpace(step.Intent),
				Command:  strings.TrimSpace(step.Command),
			}
			hints = append(hints, hint)
			stepIDs = append(stepIDs, hint.ID)
		}
		catalog = append(catalog, installPlannerModeHint{
			Mode:     mode,
			Steps:    hints,
			Summary:  summarizePlannerStepHints(hints),
			StepIDs:  stepIDs,
			StepSize: len(stepIDs),
		})
	}
	return catalog
}

func summarizePlannerStepHints(steps []installPlannerStepHint) installPlannerCategorySummary {
	summary := installPlannerCategorySummary{}
	for _, step := range steps {
		switch step.Category {
		case "dependency":
			summary.Dependency++
		case "download":
			summary.Download++
		case "binary_install":
			summary.BinaryInstall++
		case "source_build":
			summary.SourceBuild++
		case "service":
			summary.Service++
		case "verify":
			summary.Verify++
		default:
			summary.Configure++
		}
	}
	return summary
}

func buildPlannerPreconditionHints(preconditions []installPrecondition) []installPlannerConditionHint {
	if len(preconditions) == 0 {
		return nil
	}
	hints := make([]installPlannerConditionHint, 0, len(preconditions))
	for _, pre := range preconditions {
		hints = append(hints, installPlannerConditionHint{
			ID:      strings.TrimSpace(pre.ID),
			Intent:  strings.TrimSpace(pre.Intent),
			Tool:    strings.TrimSpace(pre.Tool),
			Command: strings.TrimSpace(pre.Command),
		})
	}
	return hints
}

func classifyInstallStep(step installStep) string {
	if isServiceStep(step) {
		return "service"
	}
	if strings.EqualFold(strings.TrimSpace(step.Tool), "template") {
		return "configure"
	}

	combined := strings.ToLower(strings.TrimSpace(step.ID + " " + step.Intent + " " + step.Command))
	switch {
	case strings.Contains(combined, "apt-get install"),
		strings.Contains(combined, "apt install"),
		strings.Contains(combined, "yum install"),
		strings.Contains(combined, "dnf install"),
		strings.Contains(combined, "apk add"),
		strings.Contains(combined, "pip install"),
		strings.Contains(combined, "uv pip install"),
		strings.Contains(combined, "npm install"),
		strings.Contains(combined, "pnpm install"),
		strings.Contains(combined, "brew install"),
		strings.Contains(combined, "pacman -s"),
		strings.Contains(combined, "install deps"),
		strings.Contains(combined, "dependency"):
		return "dependency"
	case strings.Contains(combined, "wget "),
		strings.Contains(combined, "curl "),
		strings.Contains(combined, "git clone"),
		strings.Contains(combined, "git fetch"),
		strings.Contains(combined, "gh release download"),
		strings.Contains(combined, "aria2c"),
		strings.Contains(combined, "rsync"),
		strings.Contains(combined, "hf download"),
		strings.Contains(combined, "ollama pull"),
		strings.Contains(combined, "modelscope download"),
		strings.Contains(combined, "download"):
		return "download"
	case strings.Contains(combined, "cmake"),
		strings.Contains(combined, "make "),
		strings.Contains(combined, "go build"),
		strings.Contains(combined, "cargo build"),
		strings.Contains(combined, "cargo install"),
		strings.Contains(combined, "python -m build"),
		strings.Contains(combined, "python setup.py"),
		strings.Contains(combined, "meson compile"),
		strings.Contains(combined, "ninja"),
		strings.Contains(combined, "source install"),
		strings.Contains(combined, "install source"),
		strings.Contains(combined, "source"):
		return "source_build"
	case strings.Contains(combined, "binary"),
		strings.Contains(combined, ".deb"),
		strings.Contains(combined, ".rpm"),
		strings.Contains(combined, "dpkg -i"),
		strings.Contains(combined, "rpm -i"),
		strings.Contains(combined, "install_binary"),
		strings.Contains(combined, "install binary"):
		return "binary_install"
	case strings.Contains(combined, "verify"),
		strings.Contains(combined, "health"),
		strings.Contains(combined, "check "),
		strings.Contains(combined, "is-active"):
		return "verify"
	default:
		return "configure"
	}
}

func applyLLMInstallPlan(spec moduleInstallSpec, defaultMode string, defaultSteps []installStep, env map[string]string, llmPlan llmInstallPlan) (string, []installStep, error) {
	resolvedMode := defaultMode
	candidateMode := strings.TrimSpace(llmPlan.Mode)
	if candidateMode != "" {
		if env["LLAMA_CUDA"] == "1" && candidateMode != defaultMode {
			candidateMode = defaultMode
		}
		if _, ok := spec.Install[candidateMode]; !ok {
			return defaultMode, defaultSteps, i18n.Errorf("LLM returned unsupported mode %q", candidateMode)
		}
		resolvedMode = candidateMode
	}

	stepsForMode := defaultSteps
	if resolvedMode != defaultMode {
		modeSteps := spec.Install[resolvedMode]
		if len(modeSteps) == 0 {
			return defaultMode, defaultSteps, i18n.Errorf("resolved mode %q has no steps", resolvedMode)
		}
		stepsForMode = modeSteps
	}

	if unknown := unknownStepIDs(stepsForMode, llmPlan.Steps); len(unknown) > 0 {
		return defaultMode, defaultSteps, i18n.Errorf("LLM returned unknown step IDs: %s", strings.Join(unknown, ", "))
	}

	selected := filterStepsByID(stepsForMode, llmPlan.Steps)
	if len(selected) == 0 {
		selected = stepsForMode
	}
	selected = ensureServiceSteps(selected, stepsForMode)

	return resolvedMode, selected, nil
}

func unknownStepIDs(steps []installStep, ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	allowed := make(map[string]bool, len(steps))
	for _, step := range steps {
		allowed[strings.TrimSpace(step.ID)] = true
	}
	unknown := make([]string, 0)
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if !allowed[trimmed] {
			unknown = append(unknown, trimmed)
		}
	}
	return unknown
}

func dedupeStepIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}

func normalizeRiskLevel(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(risk))
	default:
		return "medium"
	}
}

func stepIDs(steps []installStep) []string {
	if len(steps) == 0 {
		return nil
	}
	ids := make([]string, 0, len(steps))
	for _, step := range steps {
		id := strings.TrimSpace(step.ID)
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func isInstallPlannerDebugEnabled() bool {
	return isTruthyEnv("LOCALAISTACK_INSTALL_PLANNER_DEBUG")
}

func isInstallPlannerStrictEnabled() bool {
	return isTruthyEnv("LOCALAISTACK_INSTALL_PLANNER_STRICT")
}

func isTruthyEnv(key string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on", "y":
		return true
	default:
		return false
	}
}

func extractFirstJSONObject(text string) string {
	start := -1
	depth := 0
	inString := false
	escaped := false

	for i := 0; i < len(text); i++ {
		ch := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				return text[start : i+1]
			}
		}
	}

	return ""
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

func runInstallStep(moduleName, moduleDir string, step installStep, vars map[string]string, env map[string]string) error {
	switch strings.TrimSpace(step.Tool) {
	case "shell":
		output, exitCode, err := runShellCommandWithEnv(step.Command, moduleDir, true, env)
		if err != nil {
			return i18n.Errorf("install step %s failed: %w", step.ID, err)
		}
		if step.Expected.ExitCode != nil && exitCode != *step.Expected.ExitCode {
			return i18n.Errorf("install step %s failed: expected exit code %d but got %d", step.ID, *step.Expected.ExitCode, exitCode)
		}
		if step.Expected.Equals != "" {
			if normalizedOutput(output) != normalizedOutput(step.Expected.Equals) {
				return i18n.Errorf("install step %s failed: expected %q but got %q", step.ID, normalizedOutput(step.Expected.Equals), normalizedOutput(output))
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
	return runShellCommandWithEnv(command, moduleDir, stream, nil)
}

func runShellCommandWithEnv(command, moduleDir string, stream bool, env map[string]string) (string, int, error) {
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = moduleDir
	cmd.Env = commandEnv(env)
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
			message := normalizedOutput(output)
			if message == "" {
				return "", exitCode, err
			}
			return message, exitCode, i18n.Errorf("%s", message)
		}
		return output, exitCode, nil
	}

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		message := normalizedOutput(string(output))
		if message == "" {
			return "", exitCode, err
		}
		return message, exitCode, i18n.Errorf("%s", message)
	}
	return string(output), exitCode, nil
}

func commandEnv(extra map[string]string) []string {
	env := append([]string{}, os.Environ()...)
	if !hasEnvKey(env, "NO_COLOR") {
		env = append(env, "NO_COLOR=1")
	}
	if !hasEnvKey(env, "CLICOLOR") {
		env = append(env, "CLICOLOR=0")
	}
	if !hasEnvKey(env, "CLICOLOR_FORCE") {
		env = append(env, "CLICOLOR_FORCE=0")
	}
	if len(extra) > 0 {
		env = append(env, formatEnv(extra)...)
	}
	return env
}

func hasEnvKey(env []string, key string) bool {
	for _, item := range env {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 && parts[0] == key {
			return true
		}
	}
	return false
}

func normalizedOutput(output string) string {
	cleaned := stripANSIEscapes(output)
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")
	return strings.TrimSpace(cleaned)
}

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x1b\x07]*(?:\x07|\x1b\\)|\x1b[@-_]`)

func stripANSIEscapes(s string) string {
	return ansiEscapePattern.ReplaceAllString(s, "")
}

func formatEnv(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	env := make([]string, 0, len(values))
	for key, value := range values {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}

func resolveBaseInfoPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "base_info.md")
	}
	primary := filepath.Join(home, ".localaistack", "base_info.md")
	if _, err := os.Stat(primary); err == nil {
		return primary
	}
	alternate := filepath.Join(home, ".localiastack", "base_info.md")
	if _, err := os.Stat(alternate); err == nil {
		return alternate
	}
	return primary
}

func detectCudaArchs(gpuName string) string {
	name := strings.ToLower(strings.TrimSpace(gpuName))
	switch {
	case strings.Contains(name, "v100"):
		return "70"
	case strings.Contains(name, "a100"):
		return "80"
	case strings.Contains(name, "h100"):
		return "90"
	case strings.Contains(name, "a10"):
		return "86"
	case strings.Contains(name, "4090"), strings.Contains(name, "4080"), strings.Contains(name, "4070"):
		return "89"
	case strings.Contains(name, "3090"), strings.Contains(name, "3080"), strings.Contains(name, "3070"):
		return "86"
	}
	return ""
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
		bin := strings.TrimSpace(expect.Bin)
		if filepath.IsAbs(bin) {
			if _, err := os.Stat(bin); err != nil {
				// Some installers place binaries in a different PATH location (e.g. /usr/bin).
				// Fall back to PATH lookup for the same command name.
				if _, lookupErr := exec.LookPath(filepath.Base(bin)); lookupErr != nil {
					return err
				}
			}
		} else {
			if _, err := exec.LookPath(bin); err != nil {
				return err
			}
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
