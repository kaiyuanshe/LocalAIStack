package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/expert"
	"github.com/zhuangbiaowei/LocalAIStack/internal/modelmanager"
)

func RegisterExpertCommands(rootCmd *cobra.Command) {
	expertCmd := &cobra.Command{
		Use:   "expert",
		Short: "Expert recipe planning and management",
	}
	expertCmd.PersistentFlags().String("recipes-root", "", "recipes root directory")

	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Interactive expert recipe planning",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExpertPlanInteractive(cmd)
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List saved expert recipes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listExpertRecipes(cmd)
		},
	}

	showCmd := &cobra.Command{
		Use:   "show [recipe-id]",
		Short: "Show expert recipe details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showExpertRecipe(cmd, args[0])
		},
	}

	buildCmd := &cobra.Command{
		Use:   "build [plan-id-or-plan-json]",
		Short: "Build expert recipe artifacts from a saved plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return buildExpertRecipe(cmd, args[0])
		},
	}
	buildCmd.Flags().Int("candidate", 1, "candidate number to build, starting at 1")
	buildCmd.Flags().String("output-dir", "", "directory for generated expert recipe artifacts")

	upCmd := &cobra.Command{
		Use:   "up [recipe-id]",
		Short: "Start a generated expert recipe with Docker Compose",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			recipeID := ""
			if len(args) > 0 {
				recipeID = args[0]
			}
			return upExpertRecipe(cmd, recipeID)
		},
	}
	upCmd.Flags().Bool("dry-run", false, "print the Docker Compose command without running it")
	upCmd.Flags().Bool("skip-preflight", false, "skip generated preflight.sh before starting")

	expertCmd.AddCommand(planCmd)
	expertCmd.AddCommand(buildCmd)
	expertCmd.AddCommand(upCmd)
	expertCmd.AddCommand(listCmd)
	expertCmd.AddCommand(showCmd)
	rootCmd.AddCommand(expertCmd)
}

func runExpertPlanInteractive(cmd *cobra.Command) error {
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════════════╗")
	fmt.Println("  ║   LocalAIStack Expert Alpha - Recipe Planner  ║")
	fmt.Println("  ╚══════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("  This wizard helps you generate optimized inference recipes")
	fmt.Println("  for your hardware and chosen model.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	if !expert.IsLLMFitAvailable() {
		fmt.Println("  ⚠ llmfit not found. Falling back to basic hardware detection.")
		fmt.Println("  Install llmfit for better model recommendations:")
		fmt.Println("    curl -fsSL https://llmfit.org/install.sh | bash")
		fmt.Println()
		return runFallbackPlan(cmd, reader)
	}

	modelRecommendationLimit := 15

	fmt.Println("  ── Step 1: Hardware Detection ──")
	fmt.Println()
	sysInfo, err := expert.CallLLMFitSystem_cmd()
	if err != nil {
		fmt.Printf("  ⚠ Could not detect hardware via llmfit: %v\n", err)
		fmt.Println("  Falling back to basic detection...")
		return runFallbackPlan(cmd, reader)
	}

	if sysInfo.HasGPU {
		fmt.Printf("  ✓ GPU detected: %s (%.0f GB VRAM)\n", sysInfo.GPUName, sysInfo.GPUVRAMGB)
	} else {
		fmt.Println("  ✓ CPU-only mode")
	}
	fmt.Printf("  ✓ System RAM: %.0f GB\n", sysInfo.AvailableRAMGB)
	fmt.Println()

	fmt.Println("  ── Step 2: Select Use Case ──")
	fmt.Println()
	useCases := []string{"general", "coding", "reasoning", "chat"}
	useCase := promptChoice(reader, "Use case", useCases, "general")

	fmt.Println()
	fmt.Println("  ── Step 3: Model Selection ──")
	fmt.Println()
	fmt.Printf("  Querying llmfit for top %d models (%s)...\n", modelRecommendationLimit, useCase)

	models, _, err := expert.CallLLMFitRecommend_cmd(modelRecommendationLimit, useCase, "marginal")
	if err != nil {
		fmt.Printf("  ⚠ llmfit query failed: %v\n", err)
		fmt.Println("  Falling back to basic mode...")
		return runFallbackPlan(cmd, reader)
	}

	if len(models) == 0 {
		fmt.Println("  ⚠ No models found matching your criteria.")
		return runFallbackPlan(cmd, reader)
	}

	localModels, _ := createModelManager().ListDownloadedModels()
	llmfitSet := buildLLMFitIndex(models)

	type selectableModel struct {
		displayID string
		modelID   string
		hasLLMFit bool
		llmfit    expert.LLMFitModel
		localInfo *modelmanager.DownloadedModel
	}

	var localEntries []selectableModel
	listedLocal := make(map[string]bool)
	for i := range localModels {
		lm := &localModels[i]
		entry := selectableModel{
			displayID: lm.ID,
			modelID:   lm.ID,
			localInfo: lm,
		}
		if m, ok := llmfitSet[strings.ToLower(lm.ID)]; ok {
			entry.hasLLMFit = true
			entry.llmfit = m
		}
		localEntries = append(localEntries, entry)
		listedLocal[strings.ToLower(lm.ID)] = true
	}

	var remoteEntries []selectableModel
	for _, m := range models {
		if !listedLocal[strings.ToLower(m.Name)] {
			remoteEntries = append(remoteEntries, selectableModel{
				displayID: m.Name,
				modelID:   m.Name,
				hasLLMFit: true,
				llmfit:    m,
			})
		}
	}

	combined := append(localEntries, remoteEntries...)

	fmt.Println()
	if len(localEntries) > 0 {
		fmt.Println("  ── 📦 Downloaded (ready to use) ──")
		for i, e := range localEntries {
			if e.hasLLMFit {
				fmt.Printf("  %-3d %s\n", i+1, formatModelLine(e.llmfit))
			} else {
				fmt.Printf("  %-3d %-50s %-8s %-10s\n",
					i+1, truncateStr(e.displayID, 50),
					string(e.localInfo.Source), string(e.localInfo.Format))
			}
		}
		fmt.Println()
	}

	if len(remoteEntries) > 0 {
		offset := len(localEntries)
		fmt.Println("  ── 🌐 Recommended (download required) ──")
		for i, e := range remoteEntries {
			fmt.Printf("  %-3d %s\n", offset+i+1, formatModelLine(e.llmfit))
		}
		fmt.Println()
		fmt.Println("  💡 To download: ./build/las model download <model_id> <file_name>")
		fmt.Println()
	}

	modelIdx := promptInt(reader, fmt.Sprintf("Select model (1-%d)", len(combined)), 1, 1, len(combined))
	selectedEntry := combined[modelIdx-1]
	selectedModel := selectedEntry.llmfit
	selectedModel.Name = selectedEntry.modelID

	modelFacts := expert.ResolveModelFacts(&selectedModel)
	fmt.Printf("\n  ✓ Selected: %s\n", selectedModel.Name)
	fmt.Printf("    Family: %v, Format: %v, Quant: %v\n",
		modelFacts["family"], modelFacts["format"], modelFacts["quantization"])
	fmt.Printf("    Fit level: %s, Est. TPS: %.1f\n", selectedModel.FitLevel, selectedModel.EstimatedTPS)

	fmt.Println()
	fmt.Println("  ── Step 4: Workload Configuration ──")
	fmt.Println()
	workloads := []string{"openai-api", "rag", "agent", "chat"}
	workload := promptChoice(reader, "Workload type", workloads, workloads[0])

	ctxOptions := []string{"8192", "16384", "32768", "65536", "131072"}
	ctxStr := promptChoice(reader, "Context length (tokens)", ctxOptions, "16384")
	contextLength, _ := strconv.Atoi(ctxStr)

	concurrency := promptInt(reader, "Expected concurrent requests", 1, 1, 64)

	fmt.Println()
	fmt.Println("  ── Step 5: Generating Candidate Recipes ──")
	fmt.Println()

	input := expert.ExpertInput{
		Model:         selectedModel.Name,
		Workload:      workload,
		ContextLength: contextLength,
		Concurrency:   concurrency,
		Constraints: expert.Constraints{
			LocalOnly: true,
			Docker:    sysInfo.HasCUDA || sysInfo.HasMetal,
		},
	}
	if sysInfo.HasGPU {
		input.Hardware = expert.HardwareSpec{
			Vendor:   "nvidia",
			VRAMGB:   int(sysInfo.AvailableVRAMGB),
			GPUModel: sysInfo.GPUName,
		}
	} else {
		input.Hardware = expert.HardwareSpec{
			Vendor: "cpu",
			VRAMGB: 0,
		}
	}

	candidates := generateCandidates(&selectedModel, sysInfo, input)

	if len(candidates) == 0 {
		fmt.Println("  ⚠ No engine adapters could handle this configuration.")
		return fmt.Errorf("no viable engine candidates found")
	}

	plan := &expert.ExpertPlan{
		Input:         input,
		Candidates:    candidates,
		HardwareFacts: map[string]any{"system": sysInfo},
		ModelFacts:    modelFacts,
	}

	for i, c := range candidates {
		fmt.Printf("  ── Candidate %d: %s ──\n", i+1, c.Engine)
		fmt.Printf("  Confidence: %s\n", c.Confidence)
		fmt.Printf("  Reason: %s\n", c.Reason)
		if len(c.Risks) > 0 {
			fmt.Println("  Risks:")
			for _, r := range c.Risks {
				fmt.Printf("    • %s\n", r)
			}
		}
		if len(c.Fallbacks) > 0 {
			fmt.Println("  Fallbacks:")
			for _, f := range c.Fallbacks {
				fmt.Printf("    • %s\n", f)
			}
		}
		if len(c.Notes) > 0 {
			for _, n := range c.Notes {
				fmt.Printf("  Note: %s\n", n)
			}
		}
		fmt.Println()
	}

	fmt.Println("  ── Step 6: Confirm and Build ──")
	fmt.Println()
	fmt.Println("  Select which candidate to build:")
	for i, c := range candidates {
		fmt.Printf("    %d: %s (confidence: %s)\n", i+1, c.Engine, c.Confidence)
	}
	fmt.Println("    0: Cancel - don't build anything")
	fmt.Println()

	candidateIdx := promptInt(reader, fmt.Sprintf("Build candidate (1-%d, or 0 to cancel)", len(candidates)), 1, 0, len(candidates))

	if candidateIdx == 0 {
		fmt.Println("\n  ✗ Planning cancelled. No files written.")
		return nil
	}

	selectedIdx := candidateIdx - 1
	expertDir, err := expertRecipesDir()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  Building artifacts for: %s (%s)...\n", candidates[selectedIdx].Engine, candidates[selectedIdx].Confidence)

	result, err := expert.BuildArtifacts(plan, selectedIdx, expertDir)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	planID := filepath.Base(result.PlanDir)
	if saveErr := SaveExpertPlan(plan, planID); saveErr != nil {
		fmt.Printf("  ⚠ Could not save plan metadata: %v\n", saveErr)
	}

	fmt.Println()
	fmt.Println("  ✓ Expert recipe built successfully!")
	fmt.Println()
	for _, f := range result.FilesWritten {
		fmt.Printf("    → %s\n", f)
	}
	fmt.Println()
	fmt.Printf("  Next steps:\n")
	fmt.Printf("    View details:   ./build/las expert show %s\n", planID)
	fmt.Printf("    View recipes:   ./build/las expert list\n")
	fmt.Println()

	return nil
}

func generateCandidates(model *expert.LLMFitModel, system *expert.LLMFitSystem, input expert.ExpertInput) []expert.CandidateRecipe {
	adapters := expert.AllAdapters()
	candidates := make([]expert.CandidateRecipe, 0, 2)

	for _, adapter := range adapters {
		if !adapter.CanHandle(system, model) {
			continue
		}

		engineConfig := adapter.Generate(system, model, input.ContextLength, input.Concurrency)
		fallbacks := adapter.GenerateFallbacks(system, model)

		c := expert.CandidateRecipe{
			Engine:     adapter.Name(),
			Confidence: computeCandidateConfidence(model, adapter.Name(), system),
			Recipe:     engineConfig,
			Fallbacks:  fallbacks,
		}

		c.Reason = buildReason(model, adapter.Name(), system)
		c.Risks = buildRisks(model, adapter.Name(), system, input)
		c.Notes = buildNotes(model, adapter.Name(), system)

		candidates = append(candidates, c)

		if len(candidates) >= 2 {
			break
		}
	}

	return candidates
}

func computeCandidateConfidence(model *expert.LLMFitModel, engine string, system *expert.LLMFitSystem) string {
	switch model.FitLevel {
	case "perfect":
		return "high"
	case "good":
		return "medium"
	case "marginal":
		return "low"
	default:
		return "low"
	}
}

func buildReason(model *expert.LLMFitModel, engine string, system *expert.LLMFitSystem) string {
	switch engine {
	case "vllm":
		if system.HasCUDA {
			return fmt.Sprintf("NVIDIA GPU with CUDA detected (%s, %.0fGB VRAM). vLLM provides high-throughput OpenAI-compatible API serving for safetensors models.", system.GPUName, system.GPUVRAMGB)
		}
		return "vLLM supports high-throughput GPU inference."
	case "sglang":
		return fmt.Sprintf("SGLang excels at agent workloads, prefix caching, and structured generation. Good for multi-turn conversations with shared system prompts.")
	case "llamacpp":
		if system.HasGPU {
			return fmt.Sprintf("llama.cpp provides GGUF support with CUDA offloading. Works well for quantized models on mid-range GPUs.")
		}
		return "llama.cpp provides efficient CPU-only inference with GGUF quantization. Best option without a GPU."
	default:
		return fmt.Sprintf("%s engine selected based on hardware and model compatibility.", engine)
	}
}

func buildRisks(model *expert.LLMFitModel, engine string, system *expert.LLMFitSystem, input expert.ExpertInput) []string {
	risks := make([]string, 0)

	switch engine {
	case "vllm":
		if system.AvailableVRAMGB < 12 {
			risks = append(risks, "VRAM may be insufficient for vLLM. Consider llama.cpp with GGUF instead.")
		}
	case "sglang":
		if system.AvailableVRAMGB < 12 {
			risks = append(risks, "VRAM may be tight for SGLang.")
		}
	case "llamacpp":
		if !system.HasGPU && model.ParamsB > 14 {
			risks = append(risks, fmt.Sprintf("CPU-only inference with %s params may be slow. Consider a smaller model.", model.ParameterCount))
		}
	}

	if model.FitLevel == "marginal" {
		risks = append(risks, fmt.Sprintf("Model fit is %s on this hardware. May need reduced context length.", model.FitLevel))
	}

	if input.ContextLength > 32768 {
		risks = append(risks, fmt.Sprintf("Long context (%d tokens) may cause OOM. Reduce context if startup fails.", input.ContextLength))
	}

	if len(risks) == 0 {
		risks = append(risks, "No major compatibility risks identified; verify with generated healthcheck and benchmark before production use.")
	}
	return risks
}

func buildNotes(model *expert.LLMFitModel, engine string, system *expert.LLMFitSystem) []string {
	notes := make([]string, 0)

	if model.BestQuant != "" && model.BestQuant != "none" {
		notes = append(notes, fmt.Sprintf("Recommended quantization: %s", model.BestQuant))
	}
	if model.EstimatedTPS > 0 {
		notes = append(notes, fmt.Sprintf("Estimated throughput: %.1f tokens/sec", model.EstimatedTPS))
	}
	notes = append(notes, fmt.Sprintf("Model requires ~%.1f GB of %s", model.MemoryRequired,
		func() string {
			if system.HasGPU && engine != "llamacpp" {
				return "VRAM"
			}
			return "RAM"
		}()))

	return notes
}

func runFallbackPlan(cmd *cobra.Command, reader *bufio.Reader) error {
	fmt.Println("  Entering basic mode: manual model specification.")

	fmt.Print("\n  Model ID (e.g. Qwen/Qwen2.5-14B-Instruct): ")
	modelID, _ := reader.ReadString('\n')
	modelID = strings.TrimSpace(modelID)

	workloads := []string{"openai-api", "rag", "agent", "chat"}
	workload := promptChoice(reader, "Workload type", workloads, workloads[0])

	ctxOptions := []string{"8192", "16384", "32768"}
	ctxStr := promptChoice(reader, "Context length", ctxOptions, "16384")
	contextLength, _ := strconv.Atoi(ctxStr)

	concurrency := promptInt(reader, "Expected concurrent requests", 1, 1, 64)

	vendors := []string{"nvidia", "cpu", "apple"}
	vendor := promptChoice(reader, "Hardware vendor", vendors, "nvidia")

	engines := []string{"vllm", "sglang", "llamacpp"}
	engineStr := promptChoice(reader, "Preferred engine", engines, engines[0])

	candidate := expert.CandidateRecipe{
		Engine:     engineStr,
		Confidence: "medium",
		Reason:     fmt.Sprintf("Manual specification for %s on %s hardware.", modelID, vendor),
		Risks:      []string{"Manual mode uses limited model metadata; verify generated runtime arguments before long-running use."},
		Recipe:     generateBasicEngineConfig(engineStr, vendor, contextLength, concurrency),
		Fallbacks:  []string{"reduce context length", "switch to smaller model"},
	}

	plan := &expert.ExpertPlan{
		Input: expert.ExpertInput{
			Model:         modelID,
			Workload:      workload,
			ContextLength: contextLength,
			Concurrency:   concurrency,
			Hardware: expert.HardwareSpec{
				Vendor: vendor,
			},
		},
		Candidates: []expert.CandidateRecipe{candidate},
	}

	fmt.Println()
	fmt.Println("  ── Generated Candidate ──")
	fmt.Printf("  Engine: %s (%s)\n", candidate.Engine, candidate.Confidence)
	fmt.Printf("  Reason: %s\n", candidate.Reason)
	fmt.Println()

	fmt.Print("  Build this recipe? (yes/no): ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "yes" && confirm != "y" {
		fmt.Println("\n  ✗ Build cancelled.")
		return nil
	}

	expertDir, err := expertRecipesDir()
	if err != nil {
		return err
	}

	result, err := expert.BuildArtifacts(plan, 0, expertDir)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	planID := filepath.Base(result.PlanDir)
	_ = SaveExpertPlan(plan, planID)

	fmt.Println()
	fmt.Println("  ✓ Expert recipe built!")
	for _, f := range result.FilesWritten {
		fmt.Printf("    → %s\n", f)
	}
	fmt.Println()

	return nil
}

func generateBasicEngineConfig(engine, vendor string, ctxLen, concurrency int) map[string]any {
	switch engine {
	case "vllm":
		return map[string]any{
			"max_model_len":          ctxLen,
			"gpu_memory_utilization": 0.90,
			"tensor_parallel_size":   1,
			"enable_prefix_caching":  true,
		}
	case "sglang":
		return map[string]any{
			"host":               "0.0.0.0",
			"port":               30000,
			"context_length":     ctxLen,
			"enable_radix_cache": true,
		}
	case "llamacpp":
		config := map[string]any{
			"ctx_size":   ctxLen,
			"threads":    "auto",
			"batch_size": 512,
		}
		if vendor == "nvidia" {
			config["backend"] = "cuda"
			config["n_gpu_layers"] = 99
		} else if vendor == "apple" {
			config["backend"] = "metal"
			config["n_gpu_layers"] = 99
		} else {
			config["backend"] = "cpu"
		}
		return config
	}
	return map[string]any{}
}

func promptChoice(reader *bufio.Reader, prompt string, options []string, defaultVal string) string {
	fmt.Printf("  %s:\n", prompt)
	for i, opt := range options {
		marker := " "
		if opt == defaultVal {
			marker = "*"
		}
		fmt.Printf("    %s %d: %s\n", marker, i+1, opt)
	}

	for {
		fmt.Printf("\n  Choice [%s]: ", defaultVal)
		input, err := reader.ReadString('\n')
		if err != nil {
			return defaultVal
		}
		input = strings.TrimSpace(input)

		if input == "" {
			return defaultVal
		}

		if idx, err := strconv.Atoi(input); err == nil && idx >= 1 && idx <= len(options) {
			return options[idx-1]
		}

		for _, opt := range options {
			if strings.EqualFold(input, opt) {
				return opt
			}
		}

		fmt.Printf("  Invalid choice. Please enter 1-%d or a value.\n", len(options))
	}
}

func promptInt(reader *bufio.Reader, prompt string, defaultVal int, minVal, maxVal int) int {
	for {
		fmt.Printf("  %s [%d]: ", prompt, defaultVal)
		input, err := reader.ReadString('\n')
		if err != nil {
			return defaultVal
		}
		input = strings.TrimSpace(input)

		if input == "" {
			return defaultVal
		}

		val, err := strconv.Atoi(input)
		if err != nil {
			fmt.Printf("  Please enter a number between %d and %d.\n", minVal, maxVal)
			continue
		}

		if val < minVal || val > maxVal {
			fmt.Printf("  Value must be between %d and %d.\n", minVal, maxVal)
			continue
		}

		return val
	}
}

func buildLLMFitIndex(models []expert.LLMFitModel) map[string]expert.LLMFitModel {
	index := make(map[string]expert.LLMFitModel, len(models))
	for _, m := range models {
		index[strings.ToLower(m.Name)] = m
	}
	return index
}

func buildLocalModelIndex(localModels []modelmanager.DownloadedModel) map[string]*modelmanager.DownloadedModel {
	index := make(map[string]*modelmanager.DownloadedModel, len(localModels))
	for i := range localModels {
		index[strings.ToLower(localModels[i].ID)] = &localModels[i]
	}
	return index
}

func formatModelLine(m expert.LLMFitModel) string {
	return fmt.Sprintf("%-50s %-8s %-10s %-8.0f  %s",
		truncateStr(m.Name, 50), m.ParameterCount, m.FitLevel, m.Score, m.Runtime)
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func listExpertRecipes(cmd *cobra.Command) error {
	expertDir, err := expertRecipesDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(expertDir)
	if err != nil {
		if os.IsNotExist(err) {
			cmd.Println("No expert recipes saved yet.")
			return nil
		}
		return err
	}
	if len(entries) == 0 {
		cmd.Println("No expert recipes saved yet.")
		return nil
	}
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tCREATED\tCANDIDATES")
	for _, entry := range entries {
		if entry.IsDir() {
			plan, err := loadExpertPlan(filepath.Join(expertDir, entry.Name()))
			if err != nil {
				continue
			}
			created := ""
			if !plan.CreatedAt.IsZero() {
				created = plan.CreatedAt.Format("2006-01-02 15:04")
			}
			candidates := fmt.Sprintf("%d", len(plan.Candidates))
			fmt.Fprintf(writer, "%s\t%s\t%s\n", entry.Name(), created, candidates)
		}
	}
	writer.Flush()
	return nil
}

func showExpertRecipe(cmd *cobra.Command, id string) error {
	expertDir, err := expertRecipesDir()
	if err != nil {
		return err
	}
	plan, err := loadExpertPlan(filepath.Join(expertDir, id))
	if err != nil {
		return fmt.Errorf("expert recipe %q not found", id)
	}
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(raw))
	return nil
}

func buildExpertRecipe(cmd *cobra.Command, planRef string) error {
	plan, _, err := loadExpertPlanRef(planRef)
	if err != nil {
		return err
	}

	candidateNumber, err := cmd.Flags().GetInt("candidate")
	if err != nil {
		return err
	}
	if candidateNumber < 1 || candidateNumber > len(plan.Candidates) {
		return fmt.Errorf("candidate must be between 1 and %d", len(plan.Candidates))
	}

	outputDir, err := cmd.Flags().GetString("output-dir")
	if err != nil {
		return err
	}
	if strings.TrimSpace(outputDir) == "" {
		outputDir, err = expertRecipesDir()
		if err != nil {
			return err
		}
	}

	result, err := expert.BuildArtifacts(plan, candidateNumber-1, outputDir)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	generatedID := filepath.Base(result.PlanDir)
	if saveErr := saveExpertPlanInDir(plan, result.PlanDir); saveErr != nil {
		cmd.Printf("Warning: could not save plan metadata: %v\n", saveErr)
	}

	cmd.Printf("Expert recipe built: %s\n", generatedID)
	for _, f := range result.FilesWritten {
		cmd.Printf("%s\n", f)
	}
	return nil
}

func upExpertRecipe(cmd *cobra.Command, recipeID string) error {
	expertDir, err := expertRecipesDir()
	if err != nil {
		return err
	}

	planDir := ""
	if strings.TrimSpace(recipeID) != "" {
		planDir = filepath.Join(expertDir, recipeID)
		if _, err := os.Stat(filepath.Join(planDir, "docker-compose.yaml")); err != nil {
			return fmt.Errorf("expert recipe %q does not have docker-compose.yaml", recipeID)
		}
	} else {
		selectedDir, err := promptExpertRecipeDir(cmd, expertDir)
		if err != nil {
			return err
		}
		planDir = selectedDir
		recipeID = filepath.Base(planDir)
	}

	if err := migrateExpertRecipeForUp(planDir); err != nil {
		return err
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}
	skipPreflight, err := cmd.Flags().GetBool("skip-preflight")
	if err != nil {
		return err
	}

	composeFile := filepath.Join(planDir, "docker-compose.yaml")
	if dryRun {
		if !skipPreflight {
			cmd.Printf("Would run: %s\n", filepath.Join(planDir, "preflight.sh"))
		}
		cmd.Printf("Would run: docker compose -f %s --env-file %s up -d\n", composeFile, filepath.Join(planDir, ".env"))
		return nil
	}

	if !skipPreflight {
		preflight := filepath.Join(planDir, "preflight.sh")
		if _, err := os.Stat(preflight); err == nil {
			if err := runExpertCommand(cmd, planDir, preflight); err != nil {
				return fmt.Errorf("preflight failed: %w", err)
			}
		}
	}

	args := []string{"compose", "-f", composeFile}
	envFile := filepath.Join(planDir, ".env")
	if _, err := os.Stat(envFile); err == nil {
		args = append(args, "--env-file", envFile)
	}
	args = append(args, "up", "-d")
	if err := runExpertCommand(cmd, planDir, "docker", args...); err != nil {
		return err
	}
	cmd.Printf("Expert recipe started: %s\n", recipeID)
	cmd.Printf("Healthcheck: %s\n", filepath.Join(planDir, "healthcheck.sh"))
	cmd.Printf("Benchmark:   %s\n", filepath.Join(planDir, "benchmark.sh"))
	return nil
}

func promptExpertRecipeDir(cmd *cobra.Command, expertDir string) (string, error) {
	entries, err := os.ReadDir(expertDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no expert recipes saved yet")
		}
		return "", err
	}

	dirs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		planDir := filepath.Join(expertDir, entry.Name())
		if _, err := os.Stat(filepath.Join(planDir, "docker-compose.yaml")); err == nil {
			dirs = append(dirs, planDir)
		}
	}
	if len(dirs) == 0 {
		return "", fmt.Errorf("no generated expert recipes with docker-compose.yaml found")
	}

	cmd.Println("Select expert recipe to start:")
	for i, dir := range dirs {
		cmd.Printf("  %d: %s\n", i+1, filepath.Base(dir))
	}
	reader := bufio.NewReader(cmd.InOrStdin())
	choice := promptInt(reader, fmt.Sprintf("Recipe (1-%d)", len(dirs)), 1, 1, len(dirs))
	return dirs[choice-1], nil
}

func runExpertCommand(cmd *cobra.Command, dir string, name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()
	c.Stdin = cmd.InOrStdin()
	return c.Run()
}

func migrateExpertRecipeForUp(planDir string) error {
	composePath := filepath.Join(planDir, "docker-compose.yaml")
	data, err := os.ReadFile(composePath)
	if err != nil {
		return err
	}
	compose := string(data)
	if !strings.Contains(compose, "las-expert-vllm") {
		return ensureExpertEnvDefaults(planDir, nil)
	}

	plan, _ := loadExpertPlan(planDir)
	modelID := "${MODEL_ID}"
	gpuModel := ""
	vramGB := 0
	if plan != nil {
		if strings.TrimSpace(plan.Input.Model) != "" {
			modelID = plan.Input.Model
		}
		gpuModel = plan.Input.Hardware.GPUModel
		vramGB = plan.Input.Hardware.VRAMGB
	}

	v100Like := isLegacyNVIDIAGPU(gpuModel)
	args := []string{"--model", modelID}
	if v100Like && vramGB <= 16 {
		args = append(args, "--max-model-len", "8192", "--gpu-memory-utilization", "0.85", "--max-num-seqs", "1")
	}
	if v100Like {
		args = append(args, "--dtype", "half")
	}
	args = append(args, "--tensor-parallel-size", "1")

	var b strings.Builder
	b.WriteString("# Generated by LocalAIStack Expert Alpha\n# Engine: vllm\n\n")
	b.WriteString("services:\n")
	b.WriteString("  las-expert-vllm:\n")
	b.WriteString("    image: ${VLLM_IMAGE:-vllm/vllm-openai:v0.6.6}\n")
	b.WriteString("    network_mode: host\n")
	b.WriteString("    ipc: host\n")
	b.WriteString("    environment:\n")
	b.WriteString("      VLLM_ENABLE_CUDA_COMPATIBILITY: ${VLLM_ENABLE_CUDA_COMPATIBILITY:-1}\n")
	b.WriteString("      NVIDIA_DISABLE_REQUIRE: ${NVIDIA_DISABLE_REQUIRE:-true}\n")
	b.WriteString("      HF_ENDPOINT: ${HF_ENDPOINT:-https://hf-mirror.com}\n")
	b.WriteString("      HTTP_PROXY: ${HTTP_PROXY:-}\n")
	b.WriteString("      HTTPS_PROXY: ${HTTPS_PROXY:-}\n")
	b.WriteString("      ALL_PROXY: ${ALL_PROXY:-}\n")
	b.WriteString("      NO_PROXY: ${NO_PROXY:-localhost,127.0.0.1}\n")
	b.WriteString("      HF_HUB_ETAG_TIMEOUT: ${HF_HUB_ETAG_TIMEOUT:-60}\n")
	b.WriteString("      HF_HUB_DOWNLOAD_TIMEOUT: ${HF_HUB_DOWNLOAD_TIMEOUT:-600}\n")
	b.WriteString("    volumes:\n")
	b.WriteString("      - ${HOME}/.cache/huggingface:/root/.cache/huggingface\n")
	b.WriteString("    deploy:\n")
	b.WriteString("      resources:\n")
	b.WriteString("        reservations:\n")
	b.WriteString("          devices:\n")
	b.WriteString("            - driver: nvidia\n")
	b.WriteString("              count: 1\n")
	b.WriteString("              capabilities: [gpu]\n")
	b.WriteString("    command: >-\n")
	b.WriteString(fmt.Sprintf("      %s\n", strings.Join(args, " ")))

	if err := os.WriteFile(composePath, []byte(b.String()), 0644); err != nil {
		return err
	}
	return ensureExpertEnvDefaults(planDir, plan)
}

func ensureExpertEnvDefaults(planDir string, plan *expert.ExpertPlan) error {
	envPath := filepath.Join(planDir, ".env")
	data, _ := os.ReadFile(envPath)
	env := string(data)
	additions := map[string]string{
		"VLLM_IMAGE":                     "vllm/vllm-openai:v0.6.6",
		"VLLM_ENABLE_CUDA_COMPATIBILITY": "1",
		"NVIDIA_DISABLE_REQUIRE":         "true",
		"HF_ENDPOINT":                    "https://hf-mirror.com",
		"HF_HUB_ETAG_TIMEOUT":            "60",
		"HF_HUB_DOWNLOAD_TIMEOUT":        "600",
	}
	for key, value := range additions {
		if !envContainsKey(env, key) {
			env += fmt.Sprintf("\n%s=%s\n", key, value)
		}
	}
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY"} {
		if envContainsKey(env, key) {
			continue
		}
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			env += fmt.Sprintf("\n%s=%s\n", key, value)
		}
	}
	if plan != nil && !envContainsKey(env, "MODEL_ID") && strings.TrimSpace(plan.Input.Model) != "" {
		env += fmt.Sprintf("\nMODEL_ID=%s\n", plan.Input.Model)
	}
	return os.WriteFile(envPath, []byte(env), 0644)
}

func envContainsKey(env, key string) bool {
	for _, line := range strings.Split(env, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			return true
		}
	}
	return false
}

func isLegacyNVIDIAGPU(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "v100") ||
		strings.Contains(lower, "t4") ||
		strings.Contains(lower, "tesla p") ||
		strings.Contains(lower, "gtx")
}

func loadExpertPlanRef(planRef string) (*expert.ExpertPlan, string, error) {
	if strings.TrimSpace(planRef) == "" {
		return nil, "", fmt.Errorf("plan reference is required")
	}

	if info, err := os.Stat(planRef); err == nil {
		if info.IsDir() {
			plan, err := loadExpertPlan(planRef)
			return plan, filepath.Base(planRef), err
		}
		plan, err := loadExpertPlanFile(planRef)
		return plan, "", err
	}

	expertDir, err := expertRecipesDir()
	if err != nil {
		return nil, "", err
	}
	planDir := filepath.Join(expertDir, planRef)
	plan, err := loadExpertPlan(planDir)
	if err != nil {
		return nil, "", fmt.Errorf("expert plan %q not found", planRef)
	}
	return plan, planRef, nil
}

func expertRecipesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".localaistack", "expert")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func loadExpertPlan(dir string) (*expert.ExpertPlan, error) {
	metaFile := filepath.Join(dir, "plan.json")
	return loadExpertPlanFile(metaFile)
}

func loadExpertPlanFile(metaFile string) (*expert.ExpertPlan, error) {
	data, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, err
	}
	var plan expert.ExpertPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func SaveExpertPlan(plan *expert.ExpertPlan, id string) error {
	expertDir, err := expertRecipesDir()
	if err != nil {
		return err
	}
	planDir := filepath.Join(expertDir, id)
	return saveExpertPlanInDir(plan, planDir)
}

func saveExpertPlanInDir(plan *expert.ExpertPlan, planDir string) error {
	if err := os.MkdirAll(planDir, 0755); err != nil {
		return err
	}
	plan.CreatedAt = time.Now()
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(planDir, "plan.json"), data, 0644)
}
