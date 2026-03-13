package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/configplanner"
	"github.com/zhuangbiaowei/LocalAIStack/internal/failure"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/llm"
	"github.com/zhuangbiaowei/LocalAIStack/internal/modelmanager"
	"github.com/zhuangbiaowei/LocalAIStack/internal/module"
	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
)

func init() {
	// Initialize commands package
}

var llmRegistryFactory = llm.NewRegistryFromConfig
var llamaRunRecommendationsLoader = loadLlamaRunRecommendations
var vllmRunRecommendationsLoader = loadVLLMRunRecommendations
var baseInfoPromptLoader = loadBaseInfoPrompt

const (
	llamaRunRecommendationsRelativePath = "llama.cpp/RUN_PARAMS_RECOMMENDATIONS.md"
	vllmRunRecommendationsRelativePath  = "vllm/RUN_PARAMS_RECOMMENDATIONS.md"
	llamaRunRecommendationsMaxBytes     = 16 * 1024
	baseInfoPromptMaxBytes              = 16 * 1024
	smartRunAdviceSchemaVersion         = 3
	smartRunFailureLogMaxBytes          = 16 * 1024
	smartRunErrorExtractModel           = "deepseek-ai/DeepSeek-V3.2"
	smartRunRetryPlannerModel           = "deepseek-ai/DeepSeek-V3.2"
	smartRunRecoveryTimeoutSeconds      = 90
)

func RegisterModuleCommands(rootCmd *cobra.Command) {
	moduleCmd := &cobra.Command{
		Use:     "module",
		Short:   "Manage software modules",
		Aliases: []string{"modules"},
	}

	installCmd := &cobra.Command{
		Use:   "install [module-name]",
		Short: "Install a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("%s\n", i18n.T("Installing module: %s", args[0]))
			if err := module.Install(args[0]); err != nil {
				cmd.Printf("%s\n", i18n.T("Module install failed: %s", err))
				return err
			}
			cmd.Printf("%s\n", i18n.T("Module %s installed successfully.", args[0]))
			return nil
		},
	}

	updateCmd := &cobra.Command{
		Use:   "update [module-name]",
		Short: "Update a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("%s\n", i18n.T("Updating module: %s", args[0]))
			if err := module.Update(args[0]); err != nil {
				cmd.Printf("%s\n", i18n.T("Module update failed: %s", err))
				return err
			}
			cmd.Printf("%s\n", i18n.T("Module %s updated successfully.", args[0]))
			return nil
		},
	}

	uninstallCmd := &cobra.Command{
		Use:   "uninstall [module-name]",
		Short: "Uninstall a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("%s\n", i18n.T("Uninstalling module: %s", args[0]))
			if err := module.Uninstall(args[0]); err != nil {
				cmd.Printf("%s\n", i18n.T("Module uninstall failed: %s", err))
				return err
			}
			cmd.Printf("%s\n", i18n.T("Module %s uninstalled successfully.", args[0]))
			return nil
		},
	}

	purgeCmd := &cobra.Command{
		Use:   "purge [module-name]",
		Short: "Purge a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("%s\n", i18n.T("Purging module: %s", args[0]))
			if err := module.Purge(args[0]); err != nil {
				cmd.Printf("%s\n", i18n.T("Module purge failed: %s", err))
				return err
			}
			cmd.Printf("%s\n", i18n.T("Module %s purged successfully.", args[0]))
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all available modules",
		Run: func(cmd *cobra.Command, args []string) {
			modulesRoot, err := module.FindModulesRoot()
			if err != nil {
				cmd.Printf("%s\n", i18n.T("Failed to locate modules directory: %v", err))
				return
			}
			registry, err := module.LoadRegistryFromDir(modulesRoot)
			if err != nil {
				cmd.Printf("%s\n", i18n.T("Failed to load modules from %s: %v", modulesRoot, err))
				return
			}

			all := registry.All()
			names := make([]string, 0, len(all))
			for name := range all {
				names = append(names, name)
			}
			sort.Strings(names)

			cmd.Println(i18n.T("Manageable modules:"))
			if len(names) == 0 {
				cmd.Println(i18n.T("- none"))
			}
			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			for _, name := range names {
				status := i18n.T("Not installed")
				if err := module.Check(name); err == nil {
					status = i18n.T("Installed")
				}
				_, _ = fmt.Fprintf(writer, "%s\n", i18n.T("- %s\t%s", name, status))
			}
			_ = writer.Flush()
		},
	}

	checkCmd := &cobra.Command{
		Use:   "check [module-name]",
		Short: "Check module installation status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := module.Check(args[0]); err != nil {
				cmd.Printf("%s\n", i18n.T("Module check failed: %s", err))
				return err
			}
			cmd.Printf("%s\n", i18n.T("Module %s is installed and healthy.", args[0]))
			return nil
		},
	}

	settingCmd := &cobra.Command{
		Use:   "setting [module-name] [setting-args...]",
		Short: "Run module-specific settings",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			settingArgs := args[1:]
			cmd.Printf("%s\n", i18n.T("Running module setting: %s %s", name, strings.Join(settingArgs, " ")))
			if err := module.Setting(name, settingArgs); err != nil {
				cmd.Printf("%s\n", i18n.T("Module setting failed: %s", err))
				return err
			}
			cmd.Printf("%s\n", i18n.T("Module setting finished: %s", name))
			return nil
		},
	}

	configPlanCmd := &cobra.Command{
		Use:   "config-plan [module-name]",
		Short: "Generate module configuration plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (retErr error) {
			moduleName := args[0]
			modelID, _ := cmd.Flags().GetString("model")
			apply, _ := cmd.Flags().GetBool("apply")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			plannerDebug, _ := cmd.Flags().GetBool("planner-debug")
			plannerStrict, _ := cmd.Flags().GetBool("planner-strict")
			outputFormat, _ := cmd.Flags().GetString("output")
			plannerProvider := ""
			plannerModel := ""
			defer func() {
				if retErr == nil {
					return
				}
				cls, advice, logPath := recordFailureWithResultBestEffort(failure.Event{
					Phase:    failure.PhaseConfigPlanner,
					Module:   moduleName,
					Model:    plannerModel,
					Provider: plannerProvider,
					Error:    retErr.Error(),
					Message:  "module config-plan failed",
					Context: map[string]any{
						"apply":          apply,
						"dry_run":        dryRun,
						"planner_strict": plannerStrict,
					},
				})
				if failure.FailureDebugEnabled() {
					cmd.Printf("Failure handling: phase=%s category=%s retryable=%t log=%s suggestion=%s\n",
						failure.PhaseConfigPlanner, cls.Category, advice.Retryable, fallbackString(logPath, "n/a"), advice.Suggestion)
				}
			}()

			baseInfoPath := configplanner.ResolveBaseInfoPath()
			baseInfo, err := system.LoadBaseInfoSummary(baseInfoPath)
			if err != nil {
				return fmt.Errorf("failed to read base info at %s (try `./build/las system init`): %w", baseInfoPath, err)
			}

			plan, err := configplanner.BuildStaticPlan(moduleName, modelID, baseInfo)
			if err != nil {
				source := "static"
				reason := err.Error()
				if plannerDebug {
					cmd.Printf("Config planner: source=%s reason=%s\n", source, reason)
				}
				if plannerStrict {
					return fmt.Errorf("config planner strict mode: %w", err)
				}
				return err
			}
			source := "static"
			reason := "static planner applied"

			if cfg, cfgErr := config.LoadConfig(); cfgErr == nil {
				plannerProvider = strings.TrimSpace(cfg.LLM.Provider)
				plannerModel = strings.TrimSpace(cfg.LLM.Model)
				if strings.TrimSpace(cfg.LLM.Provider) != "" && strings.TrimSpace(cfg.LLM.Model) != "" {
					if llmPlan, llmErr := configplanner.BuildLLMPlan(cmd.Context(), plan, baseInfo, cfg.LLM); llmErr == nil {
						plan = llmPlan
						source = "llm"
						reason = "llm planner applied"
					} else {
						source = "static"
						reason = llmErr.Error()
						if plannerStrict {
							if plannerDebug {
								cmd.Printf("Config planner: source=%s reason=%s\n", source, reason)
							}
							return fmt.Errorf("config planner strict mode: %w", llmErr)
						}
					}
				}
			} else if plannerStrict {
				if plannerDebug {
					cmd.Printf("Config planner: source=static reason=%s\n", cfgErr.Error())
				}
				return fmt.Errorf("config planner strict mode: %w", cfgErr)
			}

			if plannerDebug {
				cmd.Printf("Config planner: source=%s reason=%s\n", source, reason)
			}

			switch strings.ToLower(strings.TrimSpace(outputFormat)) {
			case "", "text":
				printConfigPlanText(cmd, plan)
			case "json":
				payload, err := json.MarshalIndent(plan, "", "  ")
				if err != nil {
					return err
				}
				cmd.Printf("%s\n", payload)
			default:
				return fmt.Errorf("unsupported output format %q (use text or json)", outputFormat)
			}

			if dryRun || !apply {
				return nil
			}
			path, err := configplanner.ApplyPlan(plan)
			if err != nil {
				return err
			}
			cmd.Printf("Config plan saved to %s\n", path)
			return nil
		},
	}
	configPlanCmd.Flags().String("model", "", "Optional model id to include in planning context")
	configPlanCmd.Flags().Bool("apply", false, "Persist config plan to ~/.localaistack/config-plans/<module>.json")
	configPlanCmd.Flags().Bool("dry-run", false, "Print generated plan without saving")
	configPlanCmd.Flags().Bool("planner-debug", false, "Print planner source and reason")
	configPlanCmd.Flags().Bool("planner-strict", false, "Fail immediately when planner cannot generate a valid plan")
	configPlanCmd.Flags().String("output", "text", "Output format for plan display (text|json)")

	moduleCmd.AddCommand(installCmd)
	moduleCmd.AddCommand(updateCmd)
	moduleCmd.AddCommand(uninstallCmd)
	moduleCmd.AddCommand(purgeCmd)
	moduleCmd.AddCommand(listCmd)
	moduleCmd.AddCommand(checkCmd)
	moduleCmd.AddCommand(settingCmd)
	moduleCmd.AddCommand(configPlanCmd)
	rootCmd.AddCommand(moduleCmd)
}

func RegisterServiceCommands(rootCmd *cobra.Command) {
	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Manage services",
	}

	startCmd := &cobra.Command{
		Use:   "start [service-name]",
		Short: "Start a service",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("%s\n", i18n.T("Starting service: %s", args[0]))
		},
	}

	stopCmd := &cobra.Command{
		Use:   "stop [service-name]",
		Short: "Stop a service",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("%s\n", i18n.T("Stopping service: %s", args[0]))
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status [service-name]",
		Short: "Get service status",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("%s\n", i18n.T("Service status: %s", args[0]))
		},
	}

	serviceCmd.AddCommand(startCmd)
	serviceCmd.AddCommand(stopCmd)
	serviceCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(serviceCmd)
}

func RegisterModelCommands(rootCmd *cobra.Command) {
	modelCmd := &cobra.Command{
		Use:   "model",
		Short: "Manage AI models",
	}

	searchCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for models",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			source, _ := cmd.Flags().GetString("source")
			limit, _ := cmd.Flags().GetInt("limit")

			mgr := createModelManager()

			if source != "" && source != "all" {
				var src modelmanager.ModelSource
				switch strings.ToLower(source) {
				case "ollama":
					src = modelmanager.SourceOllama
				case "huggingface", "hf":
					src = modelmanager.SourceHuggingFace
				case "modelscope":
					src = modelmanager.SourceModelScope
				default:
					return fmt.Errorf("unknown source: %s", source)
				}

				provider, err := mgr.GetProvider(src)
				if err != nil {
					return err
				}

				models, err := provider.Search(cmd.Context(), query, limit)
				if err != nil {
					return err
				}

				displaySearchResults(cmd, src, models)
			} else {
				results, err := mgr.SearchAll(query, limit)
				if err != nil {
					return err
				}

				for src, models := range results {
					displaySearchResults(cmd, src, models)
				}
			}

			return nil
		},
	}
	searchCmd.Flags().StringP("source", "s", "all", "Source to search (ollama, huggingface, modelscope, or all)")
	searchCmd.Flags().IntP("limit", "n", 10, "Maximum number of results per source")

	downloadCmd := &cobra.Command{
		Use:   "download [model-id] [file]",
		Short: "Download a model",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			modelID := args[0]
			fileHint := ""
			if len(args) > 1 {
				fileHint = args[1]
			}
			source, _ := cmd.Flags().GetString("source")
			flagFile, _ := cmd.Flags().GetString("file")
			if flagFile != "" {
				if fileHint != "" {
					return fmt.Errorf("file hint provided twice; use either positional [file] or --file")
				}
				fileHint = flagFile
			}

			mgr := createModelManager()

			var src modelmanager.ModelSource
			if source != "" {
				switch strings.ToLower(source) {
				case "ollama":
					src = modelmanager.SourceOllama
				case "huggingface", "hf":
					src = modelmanager.SourceHuggingFace
				case "modelscope":
					src = modelmanager.SourceModelScope
				default:
					return fmt.Errorf("unknown source: %s", source)
				}
			} else {
				var err error
				src, modelID, err = modelmanager.ParseModelID(modelID)
				if err != nil {
					return err
				}
			}

			cmd.Printf("Downloading model from %s: %s\n", src, modelID)

			progress := func(downloaded, total int64) {
				if total > 0 {
					percent := float64(downloaded) * 100 / float64(total)
					cmd.Printf("\rProgress: %.1f%% (%s / %s)", percent,
						modelmanager.FormatBytes(downloaded), modelmanager.FormatBytes(total))
				}
			}

			opts := modelmanager.DownloadOptions{
				FileHint:                fileHint,
				AllowModelScopeFallback: source == "" && src == modelmanager.SourceHuggingFace,
			}

			downloadedFrom, err := mgr.DownloadModel(src, modelID, progress, opts)
			if err != nil {
				return fmt.Errorf("failed to download model: %w", err)
			}

			if downloadedFrom != src {
				cmd.Printf("\nModel not found on %s, downloaded from %s instead.\n", src, downloadedFrom)
			}
			cmd.Println("\nModel downloaded successfully!")
			return nil
		},
	}
	downloadCmd.Flags().StringP("source", "s", "", "Source to download from (ollama, huggingface, modelscope)")
	downloadCmd.Flags().StringP("file", "f", "", "Specific model file to download (e.g. Q4_K_M.gguf)")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List downloaded models",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := createModelManager()

			models, err := mgr.ListDownloadedModels()
			if err != nil {
				return err
			}

			if len(models) == 0 {
				cmd.Println("No models downloaded yet.")
				return nil
			}

			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(writer, "NAME\tSOURCE\tFORMAT\tSIZE\tDOWNLOADED")

			for _, model := range models {
				size, _ := mgr.GetModelSize(model.ID)
				downloadTime := time.Unix(model.DownloadedAt, 0).Format("2006-01-02 15:04")
				fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
					model.ID, model.Source, model.Format,
					modelmanager.FormatBytes(size), downloadTime)
			}

			writer.Flush()
			return nil
		},
	}

	runCmd := &cobra.Command{
		Use:   "run [model-id] [gguf-file-or-quant]",
		Short: "Run a local model",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (retErr error) {
			modelID := args[0]
			source, _ := cmd.Flags().GetString("source")
			selectedFile, _ := cmd.Flags().GetString("file")
			if selectedFile == "" && len(args) == 2 {
				selectedFile = args[1]
			}
			threads, _ := cmd.Flags().GetInt("threads")
			ctxSize, _ := cmd.Flags().GetInt("ctx-size")
			gpuLayers, _ := cmd.Flags().GetInt("n-gpu-layers")
			batchSize, _ := cmd.Flags().GetInt("batch-size")
			ubatchSize, _ := cmd.Flags().GetInt("ubatch-size")
			autoBatch, _ := cmd.Flags().GetBool("auto-batch")
			smartRun, _ := cmd.Flags().GetBool("smart-run")
			smartRunDebug, _ := cmd.Flags().GetBool("smart-run-debug")
			smartRunRefresh, _ := cmd.Flags().GetBool("smart-run-refresh")
			smartRunStrict, _ := cmd.Flags().GetBool("smart-run-strict")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			temperature, _ := cmd.Flags().GetFloat64("temperature")
			topP, _ := cmd.Flags().GetFloat64("top-p")
			topK, _ := cmd.Flags().GetInt("top-k")
			minP, _ := cmd.Flags().GetFloat64("min-p")
			presencePenalty, _ := cmd.Flags().GetFloat64("presence-penalty")
			repeatPenalty, _ := cmd.Flags().GetFloat64("repeat-penalty")
			chatTemplateKwargs, _ := cmd.Flags().GetString("chat-template-kwargs")
			vllmMaxModelLen, _ := cmd.Flags().GetInt("vllm-max-model-len")
			vllmGpuMemUtil, _ := cmd.Flags().GetFloat64("vllm-gpu-memory-utilization")
			vllmTrustRemoteCode, _ := cmd.Flags().GetBool("vllm-trust-remote-code")
			textOnly, _ := cmd.Flags().GetBool("text-only")
			threadsChanged := cmd.Flags().Changed("threads")
			ctxSizeChanged := cmd.Flags().Changed("ctx-size")
			gpuLayersChanged := cmd.Flags().Changed("n-gpu-layers")
			tensorSplitChanged := cmd.Flags().Changed("tensor-split")
			batchSizeChanged := cmd.Flags().Changed("batch-size")
			ubatchSizeChanged := cmd.Flags().Changed("ubatch-size")
			temperatureChanged := cmd.Flags().Changed("temperature")
			topPChanged := cmd.Flags().Changed("top-p")
			topKChanged := cmd.Flags().Changed("top-k")
			minPChanged := cmd.Flags().Changed("min-p")
			presencePenaltyChanged := cmd.Flags().Changed("presence-penalty")
			repeatPenaltyChanged := cmd.Flags().Changed("repeat-penalty")
			chatTemplateKwargsChanged := cmd.Flags().Changed("chat-template-kwargs")
			vllmMaxModelLenChanged := cmd.Flags().Changed("vllm-max-model-len")
			vllmGpuMemUtilChanged := cmd.Flags().Changed("vllm-gpu-memory-utilization")
			vllmTrustRemoteCodeChanged := cmd.Flags().Changed("vllm-trust-remote-code")
			plannerProvider := ""
			plannerModel := ""
			defer func() {
				if retErr == nil {
					return
				}
				phase := failure.PhaseModelRun
				message := strings.ToLower(strings.TrimSpace(retErr.Error()))
				if strings.Contains(message, "smart-run") {
					phase = failure.PhaseSmartRun
				}
				cls, advice, logPath := recordFailureWithResultBestEffort(failure.Event{
					Phase:    phase,
					Model:    modelID,
					Provider: plannerProvider,
					Error:    retErr.Error(),
					Message:  "model run failed",
					Context: map[string]any{
						"source_flag":   source,
						"smart_run":     smartRun,
						"refresh":       smartRunRefresh,
						"strict":        smartRunStrict,
						"dry_run":       dryRun,
						"planner_model": plannerModel,
					},
				})
				if failure.FailureDebugEnabled() {
					cmd.Printf("Failure handling: phase=%s category=%s retryable=%t log=%s suggestion=%s\n",
						phase, cls.Category, advice.Retryable, fallbackString(logPath, "n/a"), advice.Suggestion)
				}
			}()

			var cfg *config.Config
			var cfgLoadErr error
			if smartRun {
				if loaded, err := config.LoadConfig(); err == nil {
					cfg = loaded
					plannerProvider = strings.TrimSpace(cfg.LLM.Provider)
					plannerModel = strings.TrimSpace(cfg.LLM.Model)
				} else {
					cfgLoadErr = err
				}
			}
			if smartRunStrict && !smartRun {
				return fmt.Errorf("smart-run-strict requires --smart-run")
			}
			if smartRunRefresh && !smartRun {
				return fmt.Errorf("smart-run-refresh requires --smart-run")
			}

			mgr := createModelManager()

			var src modelmanager.ModelSource
			if source != "" {
				switch strings.ToLower(source) {
				case "ollama":
					src = modelmanager.SourceOllama
				case "huggingface", "hf":
					src = modelmanager.SourceHuggingFace
				case "modelscope":
					src = modelmanager.SourceModelScope
				default:
					return fmt.Errorf("unknown source: %s", source)
				}
			} else {
				var err error
				src, modelID, err = modelmanager.ParseModelID(modelID)
				if err != nil {
					return err
				}
			}

			if src == modelmanager.SourceOllama {
				ollamaPath, err := exec.LookPath("ollama")
				if err != nil {
					return fmt.Errorf("ollama not found in PATH (install the ollama module first)")
				}
				cmd.Printf("Starting Ollama model: %s\n", modelID)
				ollamaArgs := []string{"run", modelID}
				if dryRun {
					printDryRunCommand(cmd, ollamaPath, ollamaArgs, nil)
					return nil
				}
				runCmd := exec.CommandContext(cmd.Context(), ollamaPath, ollamaArgs...)
				runCmd.Stdout = cmd.OutOrStdout()
				runCmd.Stderr = cmd.ErrOrStderr()
				runCmd.Stdin = cmd.InOrStdin()
				return runCmd.Run()
			}

			modelDir, err := mgr.ResolveLocalModelDir(src, modelID)
			if err != nil {
				return fmt.Errorf("local model not found: %w", err)
			}

			safetensorsFiles, err := modelmanager.FindSafetensorsFiles(modelDir)
			if err != nil {
				return err
			}
			ggufFiles, err := modelmanager.FindGGUFFiles(modelDir)
			if err != nil {
				return err
			}
			if len(safetensorsFiles) == 0 && len(ggufFiles) == 0 {
				return fmt.Errorf("no supported model files found for %s", modelID)
			}

			baseInfoPath := resolveBaseInfoPath()
			baseInfo, err := system.LoadBaseInfoSummary(baseInfoPath)
			if err != nil {
				return fmt.Errorf("failed to read base info at %s (try `./build/las system init`): %w", baseInfoPath, err)
			}

			if len(safetensorsFiles) > 0 {
				modelRef := modelDir
				if !hasVLLMConfig(modelDir) {
					meta, err := readModelMetadata(modelDir)
					if err != nil {
						return fmt.Errorf("vLLM requires a local config.json/params.json or a known repo id: %w", err)
					}
					if meta.ID == "" {
						return fmt.Errorf("metadata.json missing model id")
					}
					modelRef = meta.ID
				}
				vllmPath, err := exec.LookPath("vllm")
				if err != nil {
					return fmt.Errorf("vllm not found in PATH (install the vllm module first)")
				}
				vllmDefaults := defaultVLLMRunParams(baseInfo)
				if vllmMaxModelLen > 0 {
					vllmDefaults.maxModelLen = vllmMaxModelLen
				}
				if vllmGpuMemUtil > 0 {
					vllmDefaults.gpuMemUtil = vllmGpuMemUtil
				}
				enableTrustRemoteCode := vllmTrustRemoteCode || shouldAutoEnableVLLMTrustRemoteCode(modelDir)
				textOnlyModel := isLikelyTextOnlyVLLMModel(modelDir)
				if textOnly {
					vllmDefaults.skipMMProfiling = true
					vllmDefaults.limitMMPerPrompt = `{"image":0,"video":0}`
				}
				vllmSmartSource := ""
				vllmSmartReason := ""
				vllmSmartErr := error(nil)
				var vllmAdviceToPersist *smartRunAdviceEnvelope
				if smartRun {
					if !smartRunRefresh {
						if advice, err := loadSmartRunAdvice("vllm", modelID, modelRef); err == nil {
							applyVLLMAdvice(&vllmDefaults, &enableTrustRemoteCode, advice.VLLM, map[string]bool{
								"max_model_len":          vllmMaxModelLenChanged,
								"gpu_memory_utilization": vllmGpuMemUtilChanged,
								"trust_remote_code":      vllmTrustRemoteCodeChanged,
							})
							vllmSmartSource = "local"
							vllmSmartReason = "Reused last saved smart-run parameters"
						} else {
							loadErr := err
							if cfg != nil {
								advice, err := suggestVLLMAdvice(cmd.Context(), cfg.LLM, modelID, modelRef, baseInfo, vllmDefaults, enableTrustRemoteCode)
								if err == nil {
									applyVLLMAdvice(&vllmDefaults, &enableTrustRemoteCode, advice, map[string]bool{
										"max_model_len":          vllmMaxModelLenChanged,
										"gpu_memory_utilization": vllmGpuMemUtilChanged,
										"trust_remote_code":      vllmTrustRemoteCodeChanged,
									})
									vllmSmartSource = "llm"
									vllmSmartReason = "LLM advice applied"
									vllmAdviceToPersist = &smartRunAdviceEnvelope{VLLM: advice}
									if loadErr != nil && !errors.Is(loadErr, os.ErrNotExist) {
										vllmSmartReason = fmt.Sprintf("Saved params unavailable (%v); LLM advice applied", loadErr)
									}
								} else {
									vllmSmartErr = err
									if loadErr != nil && !errors.Is(loadErr, os.ErrNotExist) {
										vllmSmartErr = fmt.Errorf("load saved params: %v; llm advice failed: %w", loadErr, err)
									}
								}
							} else if cfgLoadErr != nil {
								vllmSmartErr = fmt.Errorf("load smart-run config: %w", cfgLoadErr)
								if loadErr != nil && !errors.Is(loadErr, os.ErrNotExist) {
									vllmSmartErr = fmt.Errorf("load saved params: %v; load smart-run config: %w", loadErr, cfgLoadErr)
								}
							} else {
								vllmSmartErr = loadErr
							}
						}
					} else if cfg != nil {
						advice, err := suggestVLLMAdvice(cmd.Context(), cfg.LLM, modelID, modelRef, baseInfo, vllmDefaults, enableTrustRemoteCode)
						if err == nil {
							applyVLLMAdvice(&vllmDefaults, &enableTrustRemoteCode, advice, map[string]bool{
								"max_model_len":          vllmMaxModelLenChanged,
								"gpu_memory_utilization": vllmGpuMemUtilChanged,
								"trust_remote_code":      vllmTrustRemoteCodeChanged,
							})
							vllmSmartSource = "llm"
							vllmSmartReason = "LLM advice applied (refresh requested)"
							vllmAdviceToPersist = &smartRunAdviceEnvelope{VLLM: advice}
						} else {
							vllmSmartErr = err
						}
					} else if cfgLoadErr != nil {
						vllmSmartErr = fmt.Errorf("load smart-run config: %w", cfgLoadErr)
					} else {
						vllmSmartErr = fmt.Errorf("smart-run refresh requires LLM configuration")
					}
				}
				vllmDefaults = finalizeVLLMRunParams(baseInfo, vllmDefaults, textOnlyModel)
				if vllmMaxModelLenChanged && vllmMaxModelLen > 0 {
					vllmDefaults.maxModelLen = clampInt(vllmMaxModelLen, 256, 131072)
				}
				if vllmGpuMemUtilChanged && vllmGpuMemUtil > 0 {
					vllmDefaults.gpuMemUtil = clampFloat(vllmGpuMemUtil, 0, 0.98)
				}
				if smartRun && textOnly {
					vllmDefaults.enforceEager = false
					vllmDefaults.optimizationLevel = -1
				}
				vllmSmartSource, vllmSmartReason, vllmSmartFatal := evaluateSmartRunOutcomeWithSource(smartRun, vllmSmartSource, vllmSmartReason, vllmSmartErr, smartRunStrict)
				if smartRunDebug {
					printSmartRunDebug(cmd, "vllm", vllmSmartSource, vllmSmartReason)
				}
				if vllmSmartFatal != nil {
					return vllmSmartFatal
				}
				cmd.Printf("Starting vLLM server for %s\n", modelID)
				servedModelName := suggestVLLMServedModelName(modelID)
				args := buildVLLMServeArgs(modelRef, servedModelName, host, port, vllmDefaults, enableTrustRemoteCode)
				if dryRun {
					printDryRunCommand(cmd, vllmPath, args, vllmDefaults.env)
					return nil
				}
				buildVLLMCmd := func(stdout, stderr io.Writer) (*exec.Cmd, error) {
					servedModelName := suggestVLLMServedModelName(modelID)
					args := buildVLLMServeArgs(modelRef, servedModelName, host, port, vllmDefaults, enableTrustRemoteCode)
					runCmd := exec.CommandContext(cmd.Context(), vllmPath, args...)
					if len(vllmDefaults.env) > 0 {
						runCmd.Env = append(os.Environ(), vllmDefaults.env...)
					}
					runCmd.Stdout = stdout
					runCmd.Stderr = stderr
					runCmd.Stdin = cmd.InOrStdin()
					return runCmd, nil
				}
				var recovery *smartRunRecoveryPlan
				if smartRun && cfg != nil {
					recovery = &smartRunRecoveryPlan{
						Enabled: true,
						Recover: func(ctx context.Context, startupLog string) (*smartRunAdviceEnvelope, error) {
							cmd.Printf("Smart-run recovery (vllm): startup_log_bytes=%d provider=%s base_url=%s extract_model=%s retry_model=%s timeout=%ds\n",
								len(startupLog),
								sanitizeLogValue(strings.TrimSpace(cfg.LLM.Provider)),
								sanitizeLogValue(strings.TrimSpace(cfg.LLM.BaseURL)),
								smartRunErrorExtractModel,
								smartRunRetryPlannerModel,
								withSmartRunRecoveryTimeout(cfg.LLM).TimeoutSeconds,
							)
							if smartRunDebug {
								cmd.Printf("Smart-run recovery (vllm): startup_log_excerpt=%s\n", summarizeForLog(startupLog, 600))
							}
							cmd.Printf("Smart-run recovery (vllm): extracting key errors with %s\n", smartRunErrorExtractModel)
							summary, err := extractSmartRunFailureSummary(ctx, cfg.LLM, "vllm", startupLog)
							if err != nil {
								cmd.Printf("Smart-run recovery (vllm): extraction failed: %v\n", err)
								return nil, err
							}
							cmd.Printf("Smart-run recovery (vllm): extracted_summary=%s\n", summarizeForLog(summary, 400))
							cmd.Printf("Smart-run recovery (vllm): requesting revised parameters with %s\n", smartRunRetryPlannerModel)
							advice, err := suggestVLLMAdviceFromFailure(ctx, cfg.LLM, modelID, modelRef, baseInfo, vllmDefaults, enableTrustRemoteCode, summary)
							if err != nil {
								cmd.Printf("Smart-run recovery (vllm): parameter replanning failed: %v\n", err)
								return nil, err
							}
							applyVLLMAdvice(&vllmDefaults, &enableTrustRemoteCode, advice, map[string]bool{
								"max_model_len":          vllmMaxModelLenChanged,
								"gpu_memory_utilization": vllmGpuMemUtilChanged,
								"trust_remote_code":      vllmTrustRemoteCodeChanged,
							})
							vllmDefaults = finalizeVLLMRunParams(baseInfo, vllmDefaults, textOnlyModel)
							env := &smartRunAdviceEnvelope{
								Reason: "Recovered after startup failure via Qwen/Kimi smart-run chain",
								VLLM:   advice,
							}
							return env, nil
						},
					}
				}
				return startCommandAndPersistAdvice(cmd, buildVLLMCmd, "vllm", modelID, modelRef, vllmAdviceToPersist, recovery)
			}

			modelPath, autoSelected, err := resolveGGUFFile(modelDir, ggufFiles, selectedFile)
			if err != nil {
				return err
			}
			if autoSelected && len(ggufFiles) > 1 {
				cmd.Printf("Auto-selected GGUF file: %s\n", filepath.Base(modelPath))
			}

			defaults := defaultLlamaRunParams(baseInfo)
			defaults = autoTuneRunParams(defaults, baseInfo, modelPath)
			if threads > 0 {
				defaults.threads = threads
			}
			if ctxSize > 0 {
				defaults.ctxSize = ctxSize
			}
			if gpuLayers >= 0 {
				defaults.gpuLayers = gpuLayers
			}
			if tensorSplit, _ := cmd.Flags().GetString("tensor-split"); tensorSplit != "" {
				defaults.tensorSplit = tensorSplit
			}
			resolvedBatch := batchSize
			resolvedUBatch := ubatchSize
			if autoBatch || resolvedBatch == 0 || resolvedUBatch == 0 {
				autoResolved := autoTuneBatchParams(baseInfo, modelPath, defaults.ctxSize, defaults.gpuLayers)
				if resolvedBatch == 0 {
					resolvedBatch = autoResolved.BatchSize
				}
				if resolvedUBatch == 0 {
					resolvedUBatch = autoResolved.UBatchSize
				}
			}
			if resolvedUBatch > 0 && resolvedBatch > 0 && resolvedUBatch > resolvedBatch {
				resolvedUBatch = resolvedBatch
			}
			sampling := llamaSamplingParams{
				Temperature:     temperature,
				TopP:            topP,
				TopK:            topK,
				MinP:            minP,
				PresencePenalty: presencePenalty,
				RepeatPenalty:   repeatPenalty,
			}
			llamaSmartSource := ""
			llamaSmartReason := ""
			llamaSmartErr := error(nil)
			var llamaAdviceToPersist *smartRunAdviceEnvelope
			if smartRun {
				selector := filepath.Base(modelPath)
				if !smartRunRefresh {
					if advice, err := loadSmartRunAdvice("llama.cpp", modelID, selector); err == nil {
						applyLlamaAdvice(&defaults, &resolvedBatch, &resolvedUBatch, &sampling, &chatTemplateKwargs, advice.Llama, map[string]bool{
							"threads":              threadsChanged,
							"ctx_size":             ctxSizeChanged,
							"n_gpu_layers":         gpuLayersChanged,
							"tensor_split":         tensorSplitChanged,
							"batch_size":           batchSizeChanged,
							"ubatch_size":          ubatchSizeChanged,
							"temperature":          temperatureChanged,
							"top_p":                topPChanged,
							"top_k":                topKChanged,
							"min_p":                minPChanged,
							"presence_penalty":     presencePenaltyChanged,
							"repeat_penalty":       repeatPenaltyChanged,
							"chat_template_kwargs": chatTemplateKwargsChanged,
						})
						llamaSmartSource = "local"
						llamaSmartReason = "Reused last saved smart-run parameters"
					} else {
						loadErr := err
						if smartRun && cfg != nil {
							advice, err := suggestLlamaAdvice(cmd.Context(), cfg.LLM, modelID, modelPath, baseInfo, defaults, llamaBatchParams{
								BatchSize:  resolvedBatch,
								UBatchSize: resolvedUBatch,
							}, sampling, chatTemplateKwargs)
							if err == nil {
								applyLlamaAdvice(&defaults, &resolvedBatch, &resolvedUBatch, &sampling, &chatTemplateKwargs, advice, map[string]bool{
									"threads":              threadsChanged,
									"ctx_size":             ctxSizeChanged,
									"n_gpu_layers":         gpuLayersChanged,
									"tensor_split":         tensorSplitChanged,
									"batch_size":           batchSizeChanged,
									"ubatch_size":          ubatchSizeChanged,
									"temperature":          temperatureChanged,
									"top_p":                topPChanged,
									"top_k":                topKChanged,
									"min_p":                minPChanged,
									"presence_penalty":     presencePenaltyChanged,
									"repeat_penalty":       repeatPenaltyChanged,
									"chat_template_kwargs": chatTemplateKwargsChanged,
								})
								llamaSmartSource = "llm"
								llamaSmartReason = "LLM advice applied"
								llamaAdviceToPersist = &smartRunAdviceEnvelope{Llama: advice}
								if loadErr != nil && !errors.Is(loadErr, os.ErrNotExist) {
									llamaSmartReason = fmt.Sprintf("Saved params unavailable (%v); LLM advice applied", loadErr)
								}
							} else {
								llamaSmartErr = err
								if loadErr != nil && !errors.Is(loadErr, os.ErrNotExist) {
									llamaSmartErr = fmt.Errorf("load saved params: %v; llm advice failed: %w", loadErr, err)
								}
							}
						} else if cfgLoadErr != nil {
							llamaSmartErr = fmt.Errorf("load smart-run config: %w", cfgLoadErr)
							if loadErr != nil && !errors.Is(loadErr, os.ErrNotExist) {
								llamaSmartErr = fmt.Errorf("load saved params: %v; load smart-run config: %w", loadErr, cfgLoadErr)
							}
						} else {
							llamaSmartErr = loadErr
						}
					}
				} else if cfg != nil {
					advice, err := suggestLlamaAdvice(cmd.Context(), cfg.LLM, modelID, modelPath, baseInfo, defaults, llamaBatchParams{
						BatchSize:  resolvedBatch,
						UBatchSize: resolvedUBatch,
					}, sampling, chatTemplateKwargs)
					if err == nil {
						applyLlamaAdvice(&defaults, &resolvedBatch, &resolvedUBatch, &sampling, &chatTemplateKwargs, advice, map[string]bool{
							"threads":              threadsChanged,
							"ctx_size":             ctxSizeChanged,
							"n_gpu_layers":         gpuLayersChanged,
							"tensor_split":         tensorSplitChanged,
							"batch_size":           batchSizeChanged,
							"ubatch_size":          ubatchSizeChanged,
							"temperature":          temperatureChanged,
							"top_p":                topPChanged,
							"top_k":                topKChanged,
							"min_p":                minPChanged,
							"presence_penalty":     presencePenaltyChanged,
							"repeat_penalty":       repeatPenaltyChanged,
							"chat_template_kwargs": chatTemplateKwargsChanged,
						})
						llamaSmartSource = "llm"
						llamaSmartReason = "LLM advice applied (refresh requested)"
						llamaAdviceToPersist = &smartRunAdviceEnvelope{Llama: advice}
					} else {
						llamaSmartErr = err
					}
				} else if cfgLoadErr != nil {
					llamaSmartErr = fmt.Errorf("load smart-run config: %w", cfgLoadErr)
				} else {
					llamaSmartErr = fmt.Errorf("smart-run refresh requires LLM configuration")
				}
			}
			llamaSmartSource, llamaSmartReason, llamaSmartFatal := evaluateSmartRunOutcomeWithSource(smartRun, llamaSmartSource, llamaSmartReason, llamaSmartErr, smartRunStrict)
			if smartRunDebug {
				printSmartRunDebug(cmd, "llama.cpp", llamaSmartSource, llamaSmartReason)
			}
			if llamaSmartFatal != nil {
				return llamaSmartFatal
			}
			if temperature < 0 {
				return fmt.Errorf("temperature must be >= 0")
			}
			if topP < 0 || topP > 1 {
				return fmt.Errorf("top-p must be in [0, 1]")
			}
			if topK < 0 {
				return fmt.Errorf("top-k must be >= 0")
			}
			if minP < 0 || minP > 1 {
				return fmt.Errorf("min-p must be in [0, 1]")
			}
			if repeatPenalty < 0 {
				return fmt.Errorf("repeat-penalty must be >= 0")
			}
			if batchSize < 0 {
				return fmt.Errorf("batch-size must be >= 0")
			}
			if ubatchSize < 0 {
				return fmt.Errorf("ubatch-size must be >= 0")
			}

			llamaPath, err := exec.LookPath("llama-server")
			if err != nil {
				return fmt.Errorf("llama-server not found in PATH (install the llama.cpp module first)")
			}

			argsList := buildLlamaServerArgs(
				modelPath,
				defaults,
				host,
				port,
				llamaSamplingParams{
					Temperature:     sampling.Temperature,
					TopP:            sampling.TopP,
					TopK:            sampling.TopK,
					MinP:            sampling.MinP,
					PresencePenalty: sampling.PresencePenalty,
					RepeatPenalty:   sampling.RepeatPenalty,
				},
				llamaBatchParams{BatchSize: resolvedBatch, UBatchSize: resolvedUBatch},
				chatTemplateKwargs,
			)

			if autoBatch {
				cmd.Printf("Auto batch tuned: --batch-size %d --ubatch-size %d\n", resolvedBatch, resolvedUBatch)
			}
			cmd.Printf("Starting llama.cpp server for %s\n", filepath.Base(modelPath))
			if dryRun {
				printDryRunCommand(cmd, llamaPath, argsList, nil)
				return nil
			}
			buildLlamaCmd := func(stdout, stderr io.Writer) (*exec.Cmd, error) {
				argsList := buildLlamaServerArgs(
					modelPath,
					defaults,
					host,
					port,
					llamaSamplingParams{
						Temperature:     sampling.Temperature,
						TopP:            sampling.TopP,
						TopK:            sampling.TopK,
						MinP:            sampling.MinP,
						PresencePenalty: sampling.PresencePenalty,
						RepeatPenalty:   sampling.RepeatPenalty,
					},
					llamaBatchParams{BatchSize: resolvedBatch, UBatchSize: resolvedUBatch},
					chatTemplateKwargs,
				)
				runCmd := exec.CommandContext(cmd.Context(), llamaPath, argsList...)
				if err := addLlamaCppLibraryPath(runCmd); err != nil {
					return nil, err
				}
				runCmd.Stdout = stdout
				runCmd.Stderr = stderr
				runCmd.Stdin = cmd.InOrStdin()
				return runCmd, nil
			}
			var recovery *smartRunRecoveryPlan
			if smartRun && cfg != nil {
				recovery = &smartRunRecoveryPlan{
					Enabled: true,
					Recover: func(ctx context.Context, startupLog string) (*smartRunAdviceEnvelope, error) {
						cmd.Printf("Smart-run recovery (llama.cpp): startup_log_bytes=%d provider=%s base_url=%s extract_model=%s retry_model=%s timeout=%ds\n",
							len(startupLog),
							sanitizeLogValue(strings.TrimSpace(cfg.LLM.Provider)),
							sanitizeLogValue(strings.TrimSpace(cfg.LLM.BaseURL)),
							smartRunErrorExtractModel,
							smartRunRetryPlannerModel,
							withSmartRunRecoveryTimeout(cfg.LLM).TimeoutSeconds,
						)
						if smartRunDebug {
							cmd.Printf("Smart-run recovery (llama.cpp): startup_log_excerpt=%s\n", summarizeForLog(startupLog, 600))
						}
						cmd.Printf("Smart-run recovery (llama.cpp): extracting key errors with %s\n", smartRunErrorExtractModel)
						summary, err := extractSmartRunFailureSummary(ctx, cfg.LLM, "llama.cpp", startupLog)
						if err != nil {
							cmd.Printf("Smart-run recovery (llama.cpp): extraction failed: %v\n", err)
							return nil, err
						}
						cmd.Printf("Smart-run recovery (llama.cpp): extracted_summary=%s\n", summarizeForLog(summary, 400))
						cmd.Printf("Smart-run recovery (llama.cpp): requesting revised parameters with %s\n", smartRunRetryPlannerModel)
						advice, err := suggestLlamaAdviceFromFailure(ctx, cfg.LLM, modelID, modelPath, baseInfo, defaults, llamaBatchParams{
							BatchSize:  resolvedBatch,
							UBatchSize: resolvedUBatch,
						}, sampling, chatTemplateKwargs, summary)
						if err != nil {
							cmd.Printf("Smart-run recovery (llama.cpp): parameter replanning failed: %v\n", err)
							return nil, err
						}
						applyLlamaAdvice(&defaults, &resolvedBatch, &resolvedUBatch, &sampling, &chatTemplateKwargs, advice, map[string]bool{
							"threads":              threadsChanged,
							"ctx_size":             ctxSizeChanged,
							"n_gpu_layers":         gpuLayersChanged,
							"tensor_split":         tensorSplitChanged,
							"batch_size":           batchSizeChanged,
							"ubatch_size":          ubatchSizeChanged,
							"temperature":          temperatureChanged,
							"top_p":                topPChanged,
							"top_k":                topKChanged,
							"min_p":                minPChanged,
							"presence_penalty":     presencePenaltyChanged,
							"repeat_penalty":       repeatPenaltyChanged,
							"chat_template_kwargs": chatTemplateKwargsChanged,
						})
						env := &smartRunAdviceEnvelope{
							Reason: "Recovered after startup failure via Qwen/Kimi smart-run chain",
							Llama:  advice,
						}
						return env, nil
					},
				}
			}
			return startCommandAndPersistAdvice(cmd, buildLlamaCmd, "llama.cpp", modelID, filepath.Base(modelPath), llamaAdviceToPersist, recovery)
		},
	}
	runCmd.Flags().StringP("source", "s", "", "Source of the model (ollama, huggingface, modelscope)")
	runCmd.Flags().StringP("file", "f", "", "Specific GGUF filename to run")
	runCmd.Flags().Int("threads", 0, "CPU threads for llama.cpp (0 = auto)")
	runCmd.Flags().Int("ctx-size", 0, "Context size for llama.cpp (0 = auto)")
	runCmd.Flags().Int("n-gpu-layers", -1, "GPU layers for llama.cpp (-1 = auto)")
	runCmd.Flags().String("tensor-split", "", "Tensor split for multi-GPU (comma-separated percentages)")
	runCmd.Flags().Int("batch-size", 0, "Batch size for llama.cpp (0 = auto)")
	runCmd.Flags().Int("ubatch-size", 0, "Micro batch size for llama.cpp (0 = auto)")
	runCmd.Flags().Bool("auto-batch", false, "Auto-tune llama.cpp --batch-size/--ubatch-size from hardware and model")
	runCmd.Flags().Bool("smart-run", false, "Use LLM to refine runtime parameters from hardware and model context")
	runCmd.Flags().Bool("smart-run-debug", false, "Print smart-run planner source and fallback reason")
	runCmd.Flags().Bool("smart-run-refresh", false, "Force smart-run to ignore saved parameters and ask the LLM again")
	runCmd.Flags().Bool("smart-run-strict", false, "Fail model run if smart-run cannot obtain valid LLM advice")
	runCmd.Flags().Bool("text-only", false, "Force multimodal vLLM models to serve text-only requests")
	runCmd.Flags().Bool("dry-run", false, "Print the final runtime command without launching the process")
	runCmd.Flags().String("host", "0.0.0.0", "Host to bind llama.cpp server")
	runCmd.Flags().Int("port", 8080, "Port to bind llama.cpp server")
	runCmd.Flags().Float64("temperature", 0.7, "Sampling temperature for llama.cpp")
	runCmd.Flags().Float64("top-p", 0.8, "Top-p nucleus sampling for llama.cpp (0-1)")
	runCmd.Flags().Int("top-k", 20, "Top-k sampling for llama.cpp (0 disables)")
	runCmd.Flags().Float64("min-p", 0.0, "Min-p sampling for llama.cpp (0-1)")
	runCmd.Flags().Float64("presence-penalty", 1.5, "Presence penalty for llama.cpp")
	runCmd.Flags().Float64("repeat-penalty", 1.0, "Repeat penalty for llama.cpp")
	runCmd.Flags().String("chat-template-kwargs", "", "JSON object passed to llama.cpp --chat-template-kwargs (e.g. '{\"enable_thinking\":false}')")
	runCmd.Flags().Int("vllm-max-model-len", 0, "vLLM max model length (safetensors only)")
	runCmd.Flags().Float64("vllm-gpu-memory-utilization", 0, "vLLM GPU memory utilization (0-1, safetensors only)")
	runCmd.Flags().Bool("vllm-trust-remote-code", false, "Allow vLLM to execute model custom code from repo (safetensors only)")

	rmCmd := &cobra.Command{
		Use:   "rm [model-id]",
		Short: "Remove a downloaded model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modelID := args[0]
			force, _ := cmd.Flags().GetBool("force")
			source, _ := cmd.Flags().GetString("source")

			if !force {
				cmd.Printf("Are you sure you want to remove model %s? Use --force to confirm.\n", modelID)
				return nil
			}

			mgr := createModelManager()

			var src modelmanager.ModelSource
			if source != "" {
				switch strings.ToLower(source) {
				case "ollama":
					src = modelmanager.SourceOllama
				case "huggingface", "hf":
					src = modelmanager.SourceHuggingFace
				case "modelscope":
					src = modelmanager.SourceModelScope
				default:
					return fmt.Errorf("unknown source: %s", source)
				}
			} else {
				var err error
				src, modelID, err = modelmanager.ParseModelID(modelID)
				if err != nil {
					return err
				}
			}

			if err := mgr.RemoveModel(src, modelID); err != nil {
				return err
			}

			cmd.Printf("Model %s removed successfully.\n", modelID)
			return nil
		},
	}
	rmCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
	rmCmd.Flags().StringP("source", "s", "", "Source of the model (ollama, huggingface, modelscope)")

	repairCmd := &cobra.Command{
		Use:     "repair [model-id]",
		Short:   "Download missing tokenizer/config files",
		Aliases: []string{"fix"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modelID := args[0]
			source, _ := cmd.Flags().GetString("source")

			mgr := createModelManager()

			var src modelmanager.ModelSource
			explicitSource := source != "" || hasExplicitSource(modelID)
			if source != "" {
				switch strings.ToLower(source) {
				case "ollama":
					src = modelmanager.SourceOllama
				case "huggingface", "hf":
					src = modelmanager.SourceHuggingFace
				case "modelscope":
					src = modelmanager.SourceModelScope
				default:
					return fmt.Errorf("unknown source: %s", source)
				}
			} else {
				var err error
				src, modelID, err = modelmanager.ParseModelID(modelID)
				if err != nil {
					return err
				}
			}

			modelDir, err := mgr.ResolveLocalModelDir(src, modelID)
			if err != nil {
				return fmt.Errorf("local model not found: %w", err)
			}

			if !explicitSource {
				if meta, err := readModelMetadata(modelDir); err == nil && meta.ID != "" {
					if meta.Source != "" {
						src = modelmanager.ModelSource(meta.Source)
					}
					modelID = meta.ID
				}
			}

			switch src {
			case modelmanager.SourceOllama:
				cmd.Println("Ollama models do not require tokenizer/config repair.")
				return nil
			case modelmanager.SourceHuggingFace:
				provider, err := mgr.GetProvider(src)
				if err != nil {
					return err
				}
				hf, ok := provider.(*modelmanager.HuggingFaceProvider)
				if !ok {
					return fmt.Errorf("huggingface provider not available")
				}
				return hf.DownloadSupportFiles(cmd.Context(), modelID, mgr.GetModelDir())
			case modelmanager.SourceModelScope:
				provider, err := mgr.GetProvider(src)
				if err != nil {
					return err
				}
				ms, ok := provider.(*modelmanager.ModelScopeProvider)
				if !ok {
					return fmt.Errorf("modelscope provider not available")
				}
				return ms.DownloadSupportFiles(cmd.Context(), modelID, mgr.GetModelDir())
			default:
				return fmt.Errorf("unsupported model source: %s", src)
			}
		},
	}
	repairCmd.Flags().StringP("source", "s", "", "Source of the model (ollama, huggingface, modelscope)")

	smartRunCacheCmd := &cobra.Command{
		Use:   "smart-run-cache",
		Short: "Manage persisted smart-run parameters",
	}

	smartRunCacheListCmd := &cobra.Command{
		Use:   "list [model-id]",
		Short: "List persisted smart-run parameters",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modelFilter := ""
			if len(args) == 1 {
				modelFilter = strings.TrimSpace(args[0])
			}
			runtimeFilter, _ := cmd.Flags().GetString("runtime")
			entries, err := listSmartRunAdviceEntries(runtimeFilter, modelFilter)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				cmd.Println("No smart-run cache entries found.")
				return nil
			}

			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(writer, "RUNTIME\tMODEL\tSELECTOR\tSAVED")
			for _, entry := range entries {
				savedAt := ""
				if !entry.SavedAt.IsZero() {
					savedAt = entry.SavedAt.Local().Format("2006-01-02 15:04:05")
				}
				fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
					entry.Runtime,
					entry.ModelID,
					fallbackString(entry.Selector, "-"),
					fallbackString(savedAt, "-"),
				)
			}
			return writer.Flush()
		},
	}
	smartRunCacheListCmd.Flags().String("runtime", "", "Filter cache entries by runtime (llama.cpp or vllm)")

	smartRunCacheRmCmd := &cobra.Command{
		Use:   "rm [model-id]",
		Short: "Remove persisted smart-run parameters for a model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeFilter, _ := cmd.Flags().GetString("runtime")
			removed, err := removeSmartRunAdviceEntries(runtimeFilter, strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			if removed == 0 {
				cmd.Printf("No smart-run cache entries found for model %s.\n", args[0])
				return nil
			}
			cmd.Printf("Removed %d smart-run cache entr", removed)
			if removed == 1 {
				cmd.Println("y.")
			} else {
				cmd.Println("ies.")
			}
			return nil
		},
	}
	smartRunCacheRmCmd.Flags().String("runtime", "", "Filter cache entries by runtime (llama.cpp or vllm)")
	smartRunCacheCmd.AddCommand(smartRunCacheListCmd)
	smartRunCacheCmd.AddCommand(smartRunCacheRmCmd)

	modelCmd.AddCommand(searchCmd)
	modelCmd.AddCommand(downloadCmd)
	modelCmd.AddCommand(listCmd)
	modelCmd.AddCommand(runCmd)
	modelCmd.AddCommand(rmCmd)
	modelCmd.AddCommand(repairCmd)
	modelCmd.AddCommand(smartRunCacheCmd)
	rootCmd.AddCommand(modelCmd)
}

func createModelManager() *modelmanager.Manager {
	home, _ := os.UserHomeDir()
	modelDir := filepath.Join(home, ".localaistack", "models")
	mgr := modelmanager.NewManager(modelDir)

	mgr.RegisterProvider(modelmanager.NewOllamaProvider())
	mgr.RegisterProvider(modelmanager.NewHuggingFaceProvider(""))
	mgr.RegisterProvider(modelmanager.NewModelScopeProvider(""))

	return mgr
}

func displaySearchResults(cmd *cobra.Command, source modelmanager.ModelSource, models []modelmanager.ModelInfo) {
	if len(models) == 0 {
		return
	}

	cmd.Printf("\n=== %s ===\n", strings.ToUpper(string(source)))
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tFORMAT\tTAGS\tDESCRIPTION")

	for _, model := range models {
		desc := model.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		tags := ""
		switch source {
		case modelmanager.SourceHuggingFace:
			tags = ""
		case modelmanager.SourceOllama:
			if model.Metadata != nil {
				tags = model.Metadata["sizes"]
				if tags == "" {
					tags = model.Metadata["tags"]
				}
			}
			if tags == "" && len(model.Tags) > 0 {
				tags = strings.Join(model.Tags, ", ")
			}
		default:
			if model.Metadata != nil {
				tags = model.Metadata["tags"]
			}
			if tags == "" && len(model.Tags) > 0 {
				tags = strings.Join(model.Tags, ", ")
			}
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", model.ID, model.Format, tags, desc)
	}

	writer.Flush()
}

func printConfigPlanText(cmd *cobra.Command, plan configplanner.Plan) {
	cmd.Printf("Config Plan\n")
	cmd.Printf("  Module: %s\n", plan.Module)
	if strings.TrimSpace(plan.Model) != "" {
		cmd.Printf("  Model: %s\n", plan.Model)
	}
	cmd.Printf("  Source: %s\n", plan.Source)
	cmd.Printf("  Reason: %s\n", plan.Reason)
	cmd.Printf("  Planner: %s@%s (%s)\n", plan.Planner.Name, plan.Planner.Version, plan.Planner.Mode)
	cmd.Printf("  Hardware: cpu=%d, memory_kb=%d, gpu=%s, gpu_count=%d\n",
		plan.Context.CPUCores, plan.Context.MemoryKB, fallbackString(plan.Context.GPUName, "n/a"), plan.Context.GPUCount)
	cmd.Printf("  GeneratedAt: %s\n", plan.GeneratedAt)
	cmd.Printf("  Changes: %d\n", len(plan.Changes))

	if len(plan.Changes) == 0 {
		return
	}
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "SCOPE\tKEY\tVALUE\tREASON")
	for _, change := range plan.Changes {
		fmt.Fprintf(writer, "%s\t%s\t%v\t%s\n",
			change.Scope, change.Key, change.Value, change.Reason)
	}
	_ = writer.Flush()

	if len(plan.Warnings) > 0 {
		cmd.Println("Warnings:")
		for _, warning := range plan.Warnings {
			cmd.Printf("- %s\n", warning)
		}
	}
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

type llamaRunDefaults struct {
	threads     int
	ctxSize     int
	gpuLayers   int
	tensorSplit string
}

type llamaBatchParams struct {
	BatchSize  int
	UBatchSize int
}

type llamaSamplingParams struct {
	Temperature     float64
	TopP            float64
	TopK            int
	MinP            float64
	PresencePenalty float64
	RepeatPenalty   float64
}

type llamaPlannerAdvice struct {
	Threads            *int     `json:"threads,omitempty"`
	CtxSize            *int     `json:"ctx_size,omitempty"`
	NGPULayers         *int     `json:"n_gpu_layers,omitempty"`
	TensorSplit        *string  `json:"tensor_split,omitempty"`
	BatchSize          *int     `json:"batch_size,omitempty"`
	UBatchSize         *int     `json:"ubatch_size,omitempty"`
	Temperature        *float64 `json:"temperature,omitempty"`
	TopP               *float64 `json:"top_p,omitempty"`
	TopK               *int     `json:"top_k,omitempty"`
	MinP               *float64 `json:"min_p,omitempty"`
	PresencePenalty    *float64 `json:"presence_penalty,omitempty"`
	RepeatPenalty      *float64 `json:"repeat_penalty,omitempty"`
	ChatTemplateKwargs *string  `json:"chat_template_kwargs,omitempty"`
}

type vllmPlannerAdvice struct {
	MaxModelLen            *int     `json:"max_model_len,omitempty"`
	GPUMemoryUtilization   *float64 `json:"gpu_memory_utilization,omitempty"`
	DType                  *string  `json:"dtype,omitempty"`
	TensorParallelSize     *int     `json:"tensor_parallel_size,omitempty"`
	EnforceEager           *bool    `json:"enforce_eager,omitempty"`
	OptimizationLevel      *int     `json:"optimization_level,omitempty"`
	MaxNumSeqs             *int     `json:"max_num_seqs,omitempty"`
	DisableCustomAllReduce *bool    `json:"disable_custom_all_reduce,omitempty"`
	TrustRemoteCode        *bool    `json:"trust_remote_code,omitempty"`
}

type smartRunAdviceEnvelope struct {
	Reason string             `json:"reason,omitempty"`
	Llama  llamaPlannerAdvice `json:"llama,omitempty"`
	VLLM   vllmPlannerAdvice  `json:"vllm,omitempty"`
}

type vllmRunDefaults struct {
	maxModelLen            int
	gpuMemUtil             float64
	dtype                  string
	tensorParallelSize     int
	enforceEager           bool
	optimizationLevel      int
	maxNumSeqs             int
	maxNumBatchedTokens    int
	attentionBackend       string
	compilationConfig      string
	skipMMProfiling        bool
	limitMMPerPrompt       string
	disableCustomAllReduce bool
	env                    []string
}

func resolveBaseInfoPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "base_info.json")
	}
	primary := filepath.Join(home, ".localaistack", "base_info.json")
	if _, err := os.Stat(primary); err == nil {
		return primary
	}
	alternate := filepath.Join(home, ".localiastack", "base_info.json")
	if _, err := os.Stat(alternate); err == nil {
		return alternate
	}
	return primary
}

func loadLlamaRunRecommendations() (string, error) {
	return loadRunRecommendations(llamaRunRecommendationsRelativePath)
}

func loadVLLMRunRecommendations() (string, error) {
	return loadRunRecommendations(vllmRunRecommendationsRelativePath)
}

func loadRunRecommendations(relativePath string) (string, error) {
	modulesRoot, err := module.FindModulesRoot()
	if err != nil {
		return "", err
	}
	docPath := filepath.Join(modulesRoot, relativePath)
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(string(raw))
	if content == "" {
		return "", fmt.Errorf("empty recommendations document: %s", docPath)
	}
	if len(content) > llamaRunRecommendationsMaxBytes {
		content = strings.TrimSpace(content[:llamaRunRecommendationsMaxBytes]) + "\n\n[truncated]"
	}
	return content, nil
}

func loadBaseInfoPrompt() (string, error) {
	path := resolveBaseInfoPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(string(raw))
	if content == "" {
		return "", fmt.Errorf("empty base info file: %s", path)
	}
	if len(content) > baseInfoPromptMaxBytes {
		content = strings.TrimSpace(content[:baseInfoPromptMaxBytes]) + "\n\n[truncated]"
	}
	return content, nil
}

func defaultLlamaRunParams(info system.BaseInfoSummary) llamaRunDefaults {
	threads := info.CPUCores
	if threads <= 0 {
		threads = runtime.NumCPU()
		if threads <= 0 {
			threads = 4
		}
	}

	ctxSize := 2048
	switch {
	case info.MemoryKB >= 64*1024*1024:
		ctxSize = 8192
	case info.MemoryKB >= 32*1024*1024:
		ctxSize = 4096
	case info.MemoryKB >= 16*1024*1024:
		ctxSize = 2048
	default:
		ctxSize = 1024
	}

	gpuLayers := 0
	vram := parseVRAMFromGPUName(info.GPUName)
	switch {
	case vram >= 80:
		gpuLayers = 80
	case vram >= 48:
		gpuLayers = 60
	case vram >= 24:
		gpuLayers = 40
	case vram >= 16:
		gpuLayers = 20
	case vram >= 12:
		gpuLayers = 12
	case vram > 0:
		gpuLayers = 8
	}

	return llamaRunDefaults{
		threads:     threads,
		ctxSize:     ctxSize,
		gpuLayers:   gpuLayers,
		tensorSplit: "",
	}
}

func defaultVLLMRunParams(info system.BaseInfoSummary) vllmRunDefaults {
	vram := parseVRAMFromGPUName(info.GPUName)
	gpuCount := info.GPUCount
	if gpuCount <= 0 && vram > 0 {
		gpuCount = 1
	}
	perGPUVRAM := vram
	legacyGPU := isLegacyCUDAInferenceGPU(info.GPUName)

	maxModelLen := 2048
	switch {
	case vram >= 80:
		maxModelLen = 32768
	case vram >= 48:
		maxModelLen = 24576
	case vram >= 24:
		maxModelLen = 16384
	case vram >= 16:
		maxModelLen = 8192
	case vram >= 12:
		maxModelLen = 6144
	case vram > 0:
		maxModelLen = 4096
	default:
		switch {
		case info.MemoryKB >= 128*1024*1024:
			maxModelLen = 8192
		case info.MemoryKB >= 64*1024*1024:
			maxModelLen = 4096
		default:
			maxModelLen = 2048
		}
	}

	gpuMemUtil := 0.0
	if gpuCount > 0 && vram > 0 {
		switch {
		case vram >= 80:
			gpuMemUtil = 0.92
		case vram >= 48:
			gpuMemUtil = 0.90
		case vram >= 24:
			gpuMemUtil = 0.88
		case vram >= 16:
			gpuMemUtil = 0.86
		default:
			gpuMemUtil = 0.82
		}
	}

	// Keep memory pressure conservative on 16GB-class cards to avoid
	// startup failures while probing KV cache memory.
	if perGPUVRAM > 0 && perGPUVRAM <= 16 {
		if maxModelLen > 2048 {
			maxModelLen = 2048
		}
		if gpuMemUtil == 0 || gpuMemUtil != 0.88 {
			gpuMemUtil = 0.88
		}
	}

	tpSize := 1
	if gpuCount >= 2 && perGPUVRAM > 0 && perGPUVRAM <= 24 {
		tpSize = 2
	} else if gpuCount > 0 {
		tpSize = gpuCount
	}
	if tpSize < 1 {
		tpSize = 1
	}

	dtype := ""
	if gpuCount > 0 {
		if legacyGPU {
			dtype = "float16"
		}
	}

	enforceEager := false
	optimizationLevel := 2
	if legacyGPU || (perGPUVRAM > 0 && perGPUVRAM <= 16) {
		enforceEager = true
		optimizationLevel = 0
	}

	maxNumSeqs := 16
	switch {
	case perGPUVRAM > 0 && perGPUVRAM <= 16:
		if tpSize > 1 {
			maxNumSeqs = 2
		} else {
			maxNumSeqs = 4
		}
	case perGPUVRAM > 0 && perGPUVRAM <= 24:
		maxNumSeqs = 8
	case perGPUVRAM > 0 && perGPUVRAM <= 48:
		maxNumSeqs = 12
	}

	disableCustomAllReduce := tpSize > 1 && (legacyGPU || perGPUVRAM <= 16)
	env := make([]string, 0, 1)

	return vllmRunDefaults{
		maxModelLen:            maxModelLen,
		gpuMemUtil:             gpuMemUtil,
		dtype:                  dtype,
		tensorParallelSize:     tpSize,
		enforceEager:           enforceEager,
		optimizationLevel:      optimizationLevel,
		maxNumSeqs:             maxNumSeqs,
		disableCustomAllReduce: disableCustomAllReduce,
		env:                    env,
	}
}

func finalizeVLLMRunParams(info system.BaseInfoSummary, defaults vllmRunDefaults, _ bool) vllmRunDefaults {
	vram := parseVRAMFromGPUName(info.GPUName)
	gpuCount := info.GPUCount
	if gpuCount <= 0 && vram > 0 {
		gpuCount = 1
	}
	perGPUVRAM := vram
	legacyGPU := isLegacyCUDAInferenceGPU(info.GPUName)

	if gpuCount <= 0 {
		defaults.tensorParallelSize = 1
	} else {
		defaults.tensorParallelSize = clampInt(defaults.tensorParallelSize, 1, gpuCount)
	}

	if legacyGPU || (perGPUVRAM > 0 && perGPUVRAM <= 16) {
		if defaults.maxModelLen > 512 {
			defaults.maxModelLen = 512
		}
		if defaults.gpuMemUtil <= 0 || defaults.gpuMemUtil > 0.80 {
			defaults.gpuMemUtil = 0.80
		}
		if defaults.dtype == "" {
			defaults.dtype = "float16"
		}
		defaults.enforceEager = true
		defaults.optimizationLevel = 0
		if defaults.maxNumSeqs <= 0 || defaults.maxNumSeqs > 1 {
			defaults.maxNumSeqs = 1
		}
		if defaults.maxNumBatchedTokens <= 0 || defaults.maxNumBatchedTokens > 128 {
			defaults.maxNumBatchedTokens = 128
		}
		if defaults.attentionBackend == "" {
			defaults.attentionBackend = "TRITON_ATTN"
		}
		if defaults.compilationConfig == "" {
			defaults.compilationConfig = `{"cudagraph_mode":"full_and_piecewise","cudagraph_capture_sizes":[1]}`
		}
		defaults.skipMMProfiling = true
		if defaults.limitMMPerPrompt == "" {
			defaults.limitMMPerPrompt = `{"image":0,"video":0}`
		}
		defaults.disableCustomAllReduce = defaults.tensorParallelSize > 1
	}

	defaults.maxModelLen = clampInt(defaults.maxModelLen, 256, 131072)
	defaults.optimizationLevel = clampInt(defaults.optimizationLevel, 0, 3)
	defaults.maxNumSeqs = clampInt(defaults.maxNumSeqs, 1, 256)
	if defaults.maxNumBatchedTokens > 0 {
		defaults.maxNumBatchedTokens = clampInt(defaults.maxNumBatchedTokens, 32, 65536)
	}
	defaults.attentionBackend = strings.TrimSpace(defaults.attentionBackend)
	defaults.compilationConfig = strings.TrimSpace(defaults.compilationConfig)
	defaults.limitMMPerPrompt = strings.TrimSpace(defaults.limitMMPerPrompt)
	defaults.gpuMemUtil = clampFloat(defaults.gpuMemUtil, 0, 0.98)

	defaults.env = defaults.env[:0]
	if defaults.tensorParallelSize > 1 {
		defaults.env = append(defaults.env, "CUDA_VISIBLE_DEVICES="+buildCUDAVisibleDevices(defaults.tensorParallelSize))
	}
	if legacyGPU {
		defaults.env = append(defaults.env, "VLLM_DISABLE_PYNCCL=1")
	}

	return defaults
}

func buildCUDAVisibleDevices(count int) string {
	if count <= 0 {
		return ""
	}
	devices := make([]string, 0, count)
	for i := 0; i < count; i++ {
		devices = append(devices, strconv.Itoa(i))
	}
	return strings.Join(devices, ",")
}

func parseVRAMFromGPUName(name string) int {
	if name == "" {
		return 0
	}
	re := regexp.MustCompile(`(?i)(\d+)\s*gb`)
	match := re.FindStringSubmatch(name)
	if len(match) < 2 {
		return 0
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return value
}

func isLegacyCUDAInferenceGPU(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return false
	}
	legacyMarkers := []string{
		"v100",
		"p100",
		"p40",
		"p4",
		"k80",
		"k40",
		"m60",
	}
	for _, marker := range legacyMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func inferModelInfo(filename string) (float64, string) {
	base := strings.ToLower(filepath.Base(filename))
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)b`)
	matches := re.FindAllStringSubmatch(base, -1)
	var max float64
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			continue
		}
		if value > max {
			max = value
		}
	}

	quant := ""
	quantPatterns := []string{
		"q2_k",
		"q3_k",
		"q4_k_m",
		"q4_k_s",
		"q4",
		"q5_k_m",
		"q5_k_s",
		"q5",
		"q6_k",
		"q6",
		"q8_0",
		"q8",
	}
	for _, pattern := range quantPatterns {
		if strings.Contains(base, pattern) {
			quant = pattern
			break
		}
	}

	return max, quant
}

func autoTuneRunParams(defaults llamaRunDefaults, info system.BaseInfoSummary, modelPath string) llamaRunDefaults {
	result := defaults
	sizeB, quant := inferModelInfo(modelPath)
	vram := parseVRAMFromGPUName(info.GPUName)
	gpuCount := info.GPUCount
	if gpuCount <= 0 && vram > 0 {
		gpuCount = 1
	}

	if info.MemoryKB >= 64*1024*1024 {
		result.ctxSize = maxInt(result.ctxSize, 8192)
	} else if info.MemoryKB >= 32*1024*1024 {
		result.ctxSize = maxInt(result.ctxSize, 4096)
	}

	if vram >= 16 && gpuCount >= 1 && sizeB > 0 && sizeB <= 30 {
		if strings.HasPrefix(quant, "q4") || strings.HasPrefix(quant, "q5") || strings.HasPrefix(quant, "q6") || strings.HasPrefix(quant, "q8") || quant == "" {
			result.gpuLayers = 999
		}
	}

	if gpuCount > 1 && result.gpuLayers != 0 {
		result.tensorSplit = makeTensorSplit(gpuCount)
	}

	return result
}

func autoTuneBatchParams(info system.BaseInfoSummary, modelPath string, ctxSize int, gpuLayers int) llamaBatchParams {
	sizeB, quant := inferModelInfo(modelPath)
	vram := parseVRAMFromGPUName(info.GPUName)
	gpuCount := info.GPUCount
	if gpuCount <= 0 && vram > 0 {
		gpuCount = 1
	}

	ubatch := 32
	switch {
	case vram >= 80:
		ubatch = 512
	case vram >= 48:
		ubatch = 256
	case vram >= 24:
		ubatch = 128
	case vram >= 16:
		ubatch = 96
	case vram >= 8:
		ubatch = 64
	default:
		if info.MemoryKB >= 128*1024*1024 {
			ubatch = 128
		} else if info.MemoryKB >= 64*1024*1024 {
			ubatch = 64
		}
	}

	if strings.HasPrefix(quant, "q8") {
		ubatch /= 2
	} else if strings.HasPrefix(quant, "q2") || strings.HasPrefix(quant, "q3") {
		ubatch *= 2
	}

	if sizeB >= 70 {
		ubatch /= 2
	} else if sizeB >= 30 {
		ubatch = ubatch * 3 / 4
	}

	switch {
	case ctxSize > 16384:
		ubatch /= 4
	case ctxSize > 8192:
		ubatch /= 2
	}

	if gpuLayers == 0 {
		ubatch /= 2
	}

	if gpuCount > 1 {
		ubatch *= minInt(gpuCount, 2)
	}

	ubatch = clampInt(closestPowerOfTwo(ubatch), 16, 1024)
	batch := ubatch * 2
	if vram >= 48 || (gpuCount > 1 && vram >= 24) {
		batch = ubatch * 4
	}
	batch = clampInt(closestPowerOfTwo(batch), ubatch, 2048)

	return llamaBatchParams{
		BatchSize:  batch,
		UBatchSize: ubatch,
	}
}

func buildVLLMServeArgs(modelRef, servedModelName, host string, port int, defaults vllmRunDefaults, enableTrustRemoteCode bool) []string {
	args := []string{"serve", "--model", modelRef, "--host", host, "--port", strconv.Itoa(port)}
	if servedModelName != "" {
		args = append(args, "--served-model-name", servedModelName)
	}
	if defaults.dtype != "" {
		args = append(args, "--dtype", defaults.dtype)
	}
	if defaults.maxModelLen > 0 {
		args = append(args, "--max-model-len", strconv.Itoa(defaults.maxModelLen))
	}
	if defaults.gpuMemUtil > 0 {
		args = append(args, "--gpu-memory-utilization", fmt.Sprintf("%.2f", defaults.gpuMemUtil))
	}
	if defaults.tensorParallelSize > 1 {
		args = append(args, "--tensor-parallel-size", strconv.Itoa(defaults.tensorParallelSize))
	}
	if defaults.attentionBackend != "" {
		args = append(args, "--attention-backend", defaults.attentionBackend)
	}
	if defaults.enforceEager {
		args = append(args, "--enforce-eager")
	}
	if defaults.optimizationLevel >= 0 {
		args = append(args, "--optimization-level", strconv.Itoa(defaults.optimizationLevel))
	}
	if defaults.disableCustomAllReduce {
		args = append(args, "--disable-custom-all-reduce")
	}
	if defaults.maxNumSeqs > 0 {
		args = append(args, "--max-num-seqs", strconv.Itoa(defaults.maxNumSeqs))
	}
	if defaults.maxNumBatchedTokens > 0 {
		args = append(args, "--max-num-batched-tokens", strconv.Itoa(defaults.maxNumBatchedTokens))
	}
	if defaults.skipMMProfiling {
		args = append(args, "--skip-mm-profiling")
	}
	if defaults.limitMMPerPrompt != "" {
		args = append(args, "--limit-mm-per-prompt", defaults.limitMMPerPrompt)
	}
	if defaults.compilationConfig != "" {
		args = append(args, "--compilation-config", defaults.compilationConfig)
	}
	if enableTrustRemoteCode {
		args = append(args, "--trust-remote-code")
	}
	return args
}

func suggestVLLMServedModelName(modelID string) string {
	base := strings.ToLower(strings.TrimSpace(filepath.Base(modelID)))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}

	replacer := strings.NewReplacer(".", "", "_", "-", " ", "-", "/", "-")
	base = replacer.Replace(base)

	parts := strings.Split(base, "-")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		switch part {
		case "awq", "gguf", "gptq", "fp16", "bf16", "int4", "int8":
			continue
		}
		filtered = append(filtered, part)
	}

	name := strings.Join(filtered, "-")
	name = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`-+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return name
}

func buildLlamaServerArgs(modelPath string, defaults llamaRunDefaults, host string, port int, sampling llamaSamplingParams, batch llamaBatchParams, chatTemplateKwargs string) []string {
	args := []string{
		"--model", modelPath,
		"--threads", strconv.Itoa(defaults.threads),
		"--ctx-size", strconv.Itoa(defaults.ctxSize),
		"--n-gpu-layers", strconv.Itoa(defaults.gpuLayers),
		"--host", host,
		"--port", strconv.Itoa(port),
		"--temp", fmt.Sprintf("%.4g", sampling.Temperature),
		"--top-p", fmt.Sprintf("%.4g", sampling.TopP),
		"--top-k", strconv.Itoa(sampling.TopK),
		"--min-p", fmt.Sprintf("%.4g", sampling.MinP),
		"--presence-penalty", fmt.Sprintf("%.4g", sampling.PresencePenalty),
		"--repeat-penalty", fmt.Sprintf("%.4g", sampling.RepeatPenalty),
	}
	if defaults.tensorSplit != "" {
		args = append(args, "--tensor-split", defaults.tensorSplit)
	}
	if batch.BatchSize > 0 {
		args = append(args, "--batch-size", strconv.Itoa(batch.BatchSize))
	}
	if batch.UBatchSize > 0 {
		args = append(args, "--ubatch-size", strconv.Itoa(batch.UBatchSize))
	}
	if strings.TrimSpace(chatTemplateKwargs) != "" {
		args = append(args, "--chat-template-kwargs", chatTemplateKwargs)
	}
	return args
}

func printDryRunCommand(cmd *cobra.Command, binary string, args []string, extraEnv []string) {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, strconv.Quote(binary))
	for _, arg := range args {
		parts = append(parts, strconv.Quote(arg))
	}
	cmd.Printf("Dry run command: %s\n", strings.Join(parts, " "))
	if len(extraEnv) > 0 {
		cmd.Printf("Dry run env: %s\n", strings.Join(extraEnv, " "))
	}
}

type smartRunRecoveryPlan struct {
	Enabled bool
	Recover func(ctx context.Context, startupLog string) (*smartRunAdviceEnvelope, error)
}

func startCommandAndPersistAdvice(cmd *cobra.Command, buildCmd func(stdout, stderr io.Writer) (*exec.Cmd, error), runtimeName, modelID, selector string, advice *smartRunAdviceEnvelope, recovery *smartRunRecoveryPlan) error {
	startupLog, err := executeManagedRunCommand(cmd, buildCmd, runtimeName, modelID, selector, advice)
	if err == nil {
		return nil
	}
	if recovery == nil || !recovery.Enabled || recovery.Recover == nil {
		return err
	}
	if !promptSmartRunFailureSubmission(cmd, runtimeName) {
		return err
	}
	refinedAdvice, recoverErr := recovery.Recover(cmd.Context(), startupLog)
	if recoverErr != nil {
		cmd.Printf("Warning: smart-run recovery failed: %v\n", recoverErr)
		return err
	}
	cmd.Printf("Retrying %s with refined smart-run parameters.\n", runtimeName)
	_, retryErr := executeManagedRunCommand(cmd, buildCmd, runtimeName, modelID, selector, refinedAdvice)
	if retryErr != nil {
		return fmt.Errorf("initial run failed: %v; retry with refined smart-run parameters failed: %w", err, retryErr)
	}
	return nil
}

func executeManagedRunCommand(cmd *cobra.Command, buildCmd func(stdout, stderr io.Writer) (*exec.Cmd, error), runtimeName, modelID, selector string, advice *smartRunAdviceEnvelope) (string, error) {
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	runCmd, err := buildCmd(io.MultiWriter(cmd.OutOrStdout(), &stdoutBuf), io.MultiWriter(cmd.ErrOrStderr(), &stderrBuf))
	if err != nil {
		return buildManagedRunLog(&stdoutBuf, &stderrBuf, err), err
	}
	if err := runCmd.Start(); err != nil {
		return buildManagedRunLog(&stdoutBuf, &stderrBuf, err), err
	}
	if advice != nil {
		if err := saveSmartRunAdvice(runtimeName, modelID, selector, *advice); err != nil {
			cmd.Printf("Warning: failed to save smart-run parameters: %v\n", err)
		}
	}
	waitErr := runCmd.Wait()
	return buildManagedRunLog(&stdoutBuf, &stderrBuf, waitErr), waitErr
}

func buildManagedRunLog(stdoutBuf, stderrBuf *bytes.Buffer, runErr error) string {
	parts := make([]string, 0, 3)
	if stdout := strings.TrimSpace(stdoutBuf.String()); stdout != "" {
		parts = append(parts, "stdout:\n"+stdout)
	}
	if stderr := strings.TrimSpace(stderrBuf.String()); stderr != "" {
		parts = append(parts, "stderr:\n"+stderr)
	}
	if runErr != nil {
		parts = append(parts, "process_error:\n"+runErr.Error())
	}
	combined := strings.TrimSpace(strings.Join(parts, "\n\n"))
	if len(combined) > smartRunFailureLogMaxBytes {
		combined = strings.TrimSpace(combined[:smartRunFailureLogMaxBytes]) + "\n\n[truncated]"
	}
	return combined
}

func promptSmartRunFailureSubmission(cmd *cobra.Command, runtimeName string) bool {
	cmd.Printf("%s startup failed. Submit the error log to LLMs for better %s parameters? [Y/N]: ", runtimeName, runtimeName)
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		cmd.Printf("\n")
		return false
	}
	choice := strings.ToUpper(strings.TrimSpace(line))
	return choice == "Y" || choice == "YES"
}

func evaluateSmartRunOutcome(enabled bool, plannerErr error, strict bool) (source, reason string, fatal error) {
	return evaluateSmartRunOutcomeWithSource(enabled, "", "", plannerErr, strict)
}

func evaluateSmartRunOutcomeWithSource(enabled bool, source, reason string, plannerErr error, strict bool) (resolvedSource, resolvedReason string, fatal error) {
	if !enabled {
		return "static", "smart-run disabled", nil
	}
	if plannerErr == nil {
		if strings.TrimSpace(source) == "" {
			source = "llm"
		}
		if strings.TrimSpace(reason) == "" {
			reason = "LLM advice applied"
		}
		return source, reason, nil
	}
	if strict {
		return "static", plannerErr.Error(), fmt.Errorf("smart-run strict mode: %w", plannerErr)
	}
	return "static", plannerErr.Error(), nil
}

type persistedSmartRunAdvice struct {
	SchemaVersion int                    `json:"schema_version"`
	Runtime       string                 `json:"runtime"`
	ModelID       string                 `json:"model_id"`
	Selector      string                 `json:"selector,omitempty"`
	SavedAt       time.Time              `json:"saved_at"`
	Advice        smartRunAdviceEnvelope `json:"advice"`
}

type smartRunAdviceEntry struct {
	Path     string
	Runtime  string
	ModelID  string
	Selector string
	SavedAt  time.Time
	Advice   smartRunAdviceEnvelope
}

func smartRunAdviceDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, ".localaistack", "smart-run"), nil
}

func listSmartRunAdviceEntries(runtimeFilter, modelFilter string) ([]smartRunAdviceEntry, error) {
	dir, err := smartRunAdviceDir()
	if err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	normalizedRuntime := strings.TrimSpace(runtimeFilter)
	normalizedModel := strings.TrimSpace(modelFilter)
	entries := make([]smartRunAdviceEntry, 0, len(files))
	for _, path := range files {
		payload, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var saved persistedSmartRunAdvice
		if err := json.Unmarshal(payload, &saved); err != nil {
			return nil, fmt.Errorf("parse saved smart-run advice %s: %w", path, err)
		}
		if saved.SchemaVersion != smartRunAdviceSchemaVersion {
			continue
		}
		if normalizedRuntime != "" && saved.Runtime != normalizedRuntime {
			continue
		}
		if normalizedModel != "" && saved.ModelID != normalizedModel {
			continue
		}
		entries = append(entries, smartRunAdviceEntry{
			Path:     path,
			Runtime:  saved.Runtime,
			ModelID:  saved.ModelID,
			Selector: saved.Selector,
			SavedAt:  saved.SavedAt,
			Advice:   saved.Advice,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ModelID != entries[j].ModelID {
			return entries[i].ModelID < entries[j].ModelID
		}
		if entries[i].Runtime != entries[j].Runtime {
			return entries[i].Runtime < entries[j].Runtime
		}
		if entries[i].Selector != entries[j].Selector {
			return entries[i].Selector < entries[j].Selector
		}
		return entries[i].SavedAt.After(entries[j].SavedAt)
	})
	return entries, nil
}

func removeSmartRunAdviceEntries(runtimeFilter, modelFilter string) (int, error) {
	entries, err := listSmartRunAdviceEntries(runtimeFilter, modelFilter)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if err := os.Remove(entry.Path); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func smartRunAdvicePath(runtimeName, modelID, selector string) (string, error) {
	dir, err := smartRunAdviceDir()
	if err != nil {
		return "", err
	}
	parts := []string{sanitizeSmartRunPathPart(runtimeName), sanitizeSmartRunPathPart(modelID)}
	if trimmed := sanitizeSmartRunPathPart(selector); trimmed != "" {
		parts = append(parts, trimmed)
	}
	return filepath.Join(dir, strings.Join(parts, "__")+".json"), nil
}

func sanitizeSmartRunPathPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	value = strings.ReplaceAll(value, string(filepath.Separator), "_")
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	value = re.ReplaceAllString(value, "_")
	value = strings.Trim(value, "._-")
	if value == "" {
		return "default"
	}
	return value
}

func loadSmartRunAdvice(runtimeName, modelID, selector string) (smartRunAdviceEnvelope, error) {
	path, err := smartRunAdvicePath(runtimeName, modelID, selector)
	if err != nil {
		return smartRunAdviceEnvelope{}, err
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return smartRunAdviceEnvelope{}, err
	}
	var saved persistedSmartRunAdvice
	if err := json.Unmarshal(payload, &saved); err != nil {
		return smartRunAdviceEnvelope{}, fmt.Errorf("parse saved smart-run advice: %w", err)
	}
	if saved.SchemaVersion != smartRunAdviceSchemaVersion {
		return smartRunAdviceEnvelope{}, fmt.Errorf("unsupported saved smart-run advice schema version: %d", saved.SchemaVersion)
	}
	if saved.Runtime != runtimeName || saved.ModelID != modelID || saved.Selector != selector {
		return smartRunAdviceEnvelope{}, fmt.Errorf("saved smart-run advice key mismatch")
	}
	return saved.Advice, nil
}

func saveSmartRunAdvice(runtimeName, modelID, selector string, advice smartRunAdviceEnvelope) error {
	path, err := smartRunAdvicePath(runtimeName, modelID, selector)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(persistedSmartRunAdvice{
		SchemaVersion: smartRunAdviceSchemaVersion,
		Runtime:       runtimeName,
		ModelID:       modelID,
		Selector:      selector,
		SavedAt:       time.Now().UTC(),
		Advice:        advice,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

func printSmartRunDebug(cmd *cobra.Command, runtimeName, source, reason string) {
	if strings.TrimSpace(reason) == "" {
		reason = "n/a"
	}
	cmd.Printf("Smart run planner (%s): source=%s reason=%s\n", runtimeName, source, reason)
}

type smartRunFailureExtraction struct {
	Summary string   `json:"summary"`
	Errors  []string `json:"errors,omitempty"`
}

func generateLLMText(ctx context.Context, cfg config.LLMConfig, modelOverride, prompt string) (string, error) {
	registry, err := llmRegistryFactory(cfg)
	if err != nil {
		return "", err
	}
	provider, err := registry.Provider(cfg.Provider)
	if err != nil {
		return "", err
	}
	resp, err := provider.Generate(ctx, llm.Request{
		Prompt:  prompt,
		Model:   modelOverride,
		Timeout: cfg.TimeoutSeconds,
	})
	if err != nil {
		errorType := "generic"
		if errors.Is(err, context.DeadlineExceeded) {
			errorType = "context_deadline_exceeded"
		} else {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				errorType = "network_timeout"
			}
		}
		return "", fmt.Errorf("llm generate failed: provider=%s model=%s base_url=%s timeout=%ds prompt_bytes=%d error_type=%s: %w",
			strings.TrimSpace(cfg.Provider),
			strings.TrimSpace(modelOverride),
			sanitizeLogValue(strings.TrimSpace(cfg.BaseURL)),
			cfg.TimeoutSeconds,
			len(prompt),
			errorType,
			err,
		)
	}
	return strings.TrimSpace(resp.Text), nil
}

func withSmartRunRecoveryTimeout(cfg config.LLMConfig) config.LLMConfig {
	recovered := cfg
	if recovered.TimeoutSeconds < smartRunRecoveryTimeoutSeconds {
		recovered.TimeoutSeconds = smartRunRecoveryTimeoutSeconds
	}
	return recovered
}

func sanitizeLogValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "n/a"
	}
	return value
}

func summarizeForLog(value string, maxLen int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "n/a"
	}
	trimmed = strings.ReplaceAll(trimmed, "\n", " | ")
	if maxLen > 0 && len(trimmed) > maxLen {
		return trimmed[:maxLen] + "...[truncated]"
	}
	return trimmed
}

func extractSmartRunFailureSummary(ctx context.Context, cfg config.LLMConfig, runtimeName, startupLog string) (string, error) {
	cfg = withSmartRunRecoveryTimeout(cfg)
	prompt := fmt.Sprintf("You are a startup log triage assistant for LocalAIStack.\n"+
		"Analyze the %s startup failure log and extract only the key errors that matter for runtime parameter tuning.\n"+
		"Return JSON only.\n"+
		"Schema:\n"+
		"{\"summary\":string,\"errors\":[string]}\n"+
		"Rules:\n"+
		"- keep the summary concise and factual.\n"+
		"- mention the most likely parameter-related failure causes first.\n"+
		"- keep original error keywords when useful.\n"+
		"- do not suggest new parameters in this step.\n"+
		"Failure log:\n"+
		"```text\n%s\n```", runtimeName, startupLog)
	text, err := generateLLMText(ctx, cfg, smartRunErrorExtractModel, prompt)
	if err != nil {
		return "", err
	}
	payload := extractFirstJSONObject(text)
	if payload == "" {
		return "", fmt.Errorf("smart-run failure extraction did not include JSON")
	}
	var extracted smartRunFailureExtraction
	if err := json.Unmarshal([]byte(payload), &extracted); err != nil {
		return "", err
	}
	parts := make([]string, 0, len(extracted.Errors)+1)
	if summary := strings.TrimSpace(extracted.Summary); summary != "" {
		parts = append(parts, summary)
	}
	for _, item := range extracted.Errors {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	joined := strings.TrimSpace(strings.Join(parts, "\n"))
	if joined == "" {
		return "", fmt.Errorf("smart-run failure extraction was empty")
	}
	return joined, nil
}

func suggestLlamaAdvice(ctx context.Context, cfg config.LLMConfig, modelID, modelPath string, info system.BaseInfoSummary, defaults llamaRunDefaults, batch llamaBatchParams, sampling llamaSamplingParams, chatTemplateKwargs string) (llamaPlannerAdvice, error) {
	input := map[string]any{
		"runtime": "llama.cpp",
		"model": map[string]any{
			"id":   modelID,
			"path": modelPath,
		},
		"hardware": info,
		"baseline": map[string]any{
			"threads":              defaults.threads,
			"ctx_size":             defaults.ctxSize,
			"n_gpu_layers":         defaults.gpuLayers,
			"tensor_split":         defaults.tensorSplit,
			"batch_size":           batch.BatchSize,
			"ubatch_size":          batch.UBatchSize,
			"temperature":          sampling.Temperature,
			"top_p":                sampling.TopP,
			"top_k":                sampling.TopK,
			"min_p":                sampling.MinP,
			"presence_penalty":     sampling.PresencePenalty,
			"repeat_penalty":       sampling.RepeatPenalty,
			"chat_template_kwargs": chatTemplateKwargs,
		},
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return llamaPlannerAdvice{}, err
	}
	recommendations, recommendationsErr := llamaRunRecommendationsLoader()
	prompt := fmt.Sprintf(`You are a runtime tuning assistant for LocalAIStack.
Return JSON only.
Schema:
{"llama":{"threads":int,"ctx_size":int,"n_gpu_layers":int,"tensor_split":string,"batch_size":int,"ubatch_size":int,"temperature":number,"top_p":number,"top_k":int,"min_p":number,"presence_penalty":number,"repeat_penalty":number,"chat_template_kwargs":string},"reason":string}
Rules:
- only suggest safe values for local inference stability.
- do not add new fields.
Input:
%s`, string(payload))
	if recommendationsErr == nil {
		prompt = fmt.Sprintf("%s\nReference tuning guide for llama.cpp (markdown):\n```markdown\n%s\n```", prompt, recommendations)
	}
	if baseInfoContent, baseInfoErr := baseInfoPromptLoader(); baseInfoErr == nil {
		prompt = fmt.Sprintf("%s\nCollected base hardware info (json):\n```json\n%s\n```", prompt, baseInfoContent)
	}
	text, err := generateLLMText(ctx, cfg, cfg.Model, prompt)
	if err != nil {
		return llamaPlannerAdvice{}, err
	}
	var env smartRunAdviceEnvelope
	if err := parseSmartRunAdvice(text, &env); err != nil {
		return llamaPlannerAdvice{}, err
	}
	return env.Llama, nil
}

func suggestVLLMAdvice(ctx context.Context, cfg config.LLMConfig, modelID, modelRef string, info system.BaseInfoSummary, defaults vllmRunDefaults, trustRemoteCode bool) (vllmPlannerAdvice, error) {
	input := map[string]any{
		"runtime": "vllm",
		"model": map[string]any{
			"id":  modelID,
			"ref": modelRef,
		},
		"hardware": info,
		"baseline": map[string]any{
			"max_model_len":             defaults.maxModelLen,
			"gpu_memory_utilization":    defaults.gpuMemUtil,
			"dtype":                     defaults.dtype,
			"tensor_parallel_size":      defaults.tensorParallelSize,
			"enforce_eager":             defaults.enforceEager,
			"optimization_level":        defaults.optimizationLevel,
			"max_num_seqs":              defaults.maxNumSeqs,
			"disable_custom_all_reduce": defaults.disableCustomAllReduce,
			"trust_remote_code":         trustRemoteCode,
		},
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return vllmPlannerAdvice{}, err
	}
	prompt := fmt.Sprintf(`You are a runtime tuning assistant for LocalAIStack.
Return JSON only.
Schema:
{"vllm":{"max_model_len":int,"gpu_memory_utilization":number,"dtype":string,"tensor_parallel_size":int,"enforce_eager":bool,"optimization_level":int,"max_num_seqs":int,"disable_custom_all_reduce":bool,"trust_remote_code":bool},"reason":string}
Rules:
- only suggest safe values for local inference stability.
- do not add new fields.
Input:
%s`, string(payload))
	if baseInfoContent, baseInfoErr := baseInfoPromptLoader(); baseInfoErr == nil {
		prompt = fmt.Sprintf("%s\nCollected base hardware info (json):\n```json\n%s\n```", prompt, baseInfoContent)
	}
	text, err := generateLLMText(ctx, cfg, cfg.Model, prompt)
	if err != nil {
		return vllmPlannerAdvice{}, err
	}
	var env smartRunAdviceEnvelope
	if err := parseSmartRunAdvice(text, &env); err != nil {
		return vllmPlannerAdvice{}, err
	}
	return env.VLLM, nil
}

func suggestLlamaAdviceFromFailure(ctx context.Context, cfg config.LLMConfig, modelID, modelPath string, info system.BaseInfoSummary, defaults llamaRunDefaults, batch llamaBatchParams, sampling llamaSamplingParams, chatTemplateKwargs, failureSummary string) (llamaPlannerAdvice, error) {
	cfg = withSmartRunRecoveryTimeout(cfg)
	input := map[string]any{
		"runtime": "llama.cpp",
		"model": map[string]any{
			"id":   modelID,
			"path": modelPath,
		},
		"hardware": info,
		"current_params": map[string]any{
			"threads":              defaults.threads,
			"ctx_size":             defaults.ctxSize,
			"n_gpu_layers":         defaults.gpuLayers,
			"tensor_split":         defaults.tensorSplit,
			"batch_size":           batch.BatchSize,
			"ubatch_size":          batch.UBatchSize,
			"temperature":          sampling.Temperature,
			"top_p":                sampling.TopP,
			"top_k":                sampling.TopK,
			"min_p":                sampling.MinP,
			"presence_penalty":     sampling.PresencePenalty,
			"repeat_penalty":       sampling.RepeatPenalty,
			"chat_template_kwargs": chatTemplateKwargs,
		},
		"failure_summary": failureSummary,
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return llamaPlannerAdvice{}, err
	}
	recommendations, recommendationsErr := llamaRunRecommendationsLoader()
	prompt := fmt.Sprintf(`You are a runtime tuning assistant for LocalAIStack.
The model failed to start with the current llama.cpp parameters.
Use the failure summary and the tuning guide to produce safer replacement parameters.
Return JSON only.
Schema:
{"llama":{"threads":int,"ctx_size":int,"n_gpu_layers":int,"tensor_split":string,"batch_size":int,"ubatch_size":int,"temperature":number,"top_p":number,"top_k":int,"min_p":number,"presence_penalty":number,"repeat_penalty":number,"chat_template_kwargs":string},"reason":string}
Rules:
- prioritize successful startup and stability over speed.
- only change parameters that help address the failure.
- do not add new fields.
Input:
%s`, string(payload))
	if recommendationsErr == nil {
		prompt = fmt.Sprintf("%s\nReference tuning guide for llama.cpp (markdown):\n```markdown\n%s\n```", prompt, recommendations)
	}
	text, err := generateLLMText(ctx, cfg, smartRunRetryPlannerModel, prompt)
	if err != nil {
		return llamaPlannerAdvice{}, err
	}
	var env smartRunAdviceEnvelope
	if err := parseSmartRunAdvice(text, &env); err != nil {
		return llamaPlannerAdvice{}, err
	}
	return env.Llama, nil
}

func suggestVLLMAdviceFromFailure(ctx context.Context, cfg config.LLMConfig, modelID, modelRef string, info system.BaseInfoSummary, defaults vllmRunDefaults, trustRemoteCode bool, failureSummary string) (vllmPlannerAdvice, error) {
	cfg = withSmartRunRecoveryTimeout(cfg)
	input := map[string]any{
		"runtime": "vllm",
		"model": map[string]any{
			"id":  modelID,
			"ref": modelRef,
		},
		"hardware": info,
		"current_params": map[string]any{
			"max_model_len":             defaults.maxModelLen,
			"gpu_memory_utilization":    defaults.gpuMemUtil,
			"dtype":                     defaults.dtype,
			"tensor_parallel_size":      defaults.tensorParallelSize,
			"enforce_eager":             defaults.enforceEager,
			"optimization_level":        defaults.optimizationLevel,
			"max_num_seqs":              defaults.maxNumSeqs,
			"disable_custom_all_reduce": defaults.disableCustomAllReduce,
			"trust_remote_code":         trustRemoteCode,
		},
		"failure_summary": failureSummary,
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return vllmPlannerAdvice{}, err
	}
	recommendations, recommendationsErr := vllmRunRecommendationsLoader()
	prompt := fmt.Sprintf(`You are a runtime tuning assistant for LocalAIStack.
The model failed to start with the current vLLM parameters.
Use the failure summary and the tuning guide to produce safer replacement parameters.
Return JSON only.
Schema:
{"vllm":{"max_model_len":int,"gpu_memory_utilization":number,"dtype":string,"tensor_parallel_size":int,"enforce_eager":bool,"optimization_level":int,"max_num_seqs":int,"disable_custom_all_reduce":bool,"trust_remote_code":bool},"reason":string}
Rules:
- prioritize successful startup and stability over speed.
- only change parameters that help address the failure.
- do not add new fields.
Input:
%s`, string(payload))
	if recommendationsErr == nil {
		prompt = fmt.Sprintf("%s\nReference tuning guide for vLLM (markdown):\n```markdown\n%s\n```", prompt, recommendations)
	}
	text, err := generateLLMText(ctx, cfg, smartRunRetryPlannerModel, prompt)
	if err != nil {
		return vllmPlannerAdvice{}, err
	}
	var env smartRunAdviceEnvelope
	if err := parseSmartRunAdvice(text, &env); err != nil {
		return vllmPlannerAdvice{}, err
	}
	return env.VLLM, nil
}

func parseSmartRunAdvice(text string, out *smartRunAdviceEnvelope) error {
	payload := extractFirstJSONObject(text)
	if payload == "" {
		return fmt.Errorf("smart run response did not include JSON")
	}
	if err := json.Unmarshal([]byte(payload), out); err != nil {
		return err
	}
	return nil
}

func applyLlamaAdvice(defaults *llamaRunDefaults, resolvedBatch *int, resolvedUBatch *int, sampling *llamaSamplingParams, chatTemplateKwargs *string, advice llamaPlannerAdvice, changed map[string]bool) {
	if !changed["threads"] && advice.Threads != nil {
		defaults.threads = clampInt(*advice.Threads, 1, 256)
	}
	if !changed["ctx_size"] && advice.CtxSize != nil {
		defaults.ctxSize = clampInt(*advice.CtxSize, 512, 262144)
	}
	if !changed["n_gpu_layers"] && advice.NGPULayers != nil {
		defaults.gpuLayers = clampInt(*advice.NGPULayers, 0, 999)
	}
	if !changed["tensor_split"] && advice.TensorSplit != nil {
		defaults.tensorSplit = strings.TrimSpace(*advice.TensorSplit)
	}
	if !changed["batch_size"] && advice.BatchSize != nil {
		*resolvedBatch = clampInt(*advice.BatchSize, 16, 4096)
	}
	if !changed["ubatch_size"] && advice.UBatchSize != nil {
		*resolvedUBatch = clampInt(*advice.UBatchSize, 16, 2048)
	}
	if !changed["temperature"] && advice.Temperature != nil {
		sampling.Temperature = clampFloat(*advice.Temperature, 0.0, 2.0)
	}
	if !changed["top_p"] && advice.TopP != nil {
		sampling.TopP = clampFloat(*advice.TopP, 0.0, 1.0)
	}
	if !changed["top_k"] && advice.TopK != nil {
		sampling.TopK = clampInt(*advice.TopK, 0, 1000)
	}
	if !changed["min_p"] && advice.MinP != nil {
		sampling.MinP = clampFloat(*advice.MinP, 0.0, 1.0)
	}
	if !changed["presence_penalty"] && advice.PresencePenalty != nil {
		sampling.PresencePenalty = clampFloat(*advice.PresencePenalty, 0.0, 2.0)
	}
	if !changed["repeat_penalty"] && advice.RepeatPenalty != nil {
		sampling.RepeatPenalty = clampFloat(*advice.RepeatPenalty, 0.0, 2.0)
	}
	if !changed["chat_template_kwargs"] && advice.ChatTemplateKwargs != nil {
		*chatTemplateKwargs = strings.TrimSpace(*advice.ChatTemplateKwargs)
	}
	if *resolvedUBatch > 0 && *resolvedBatch > 0 && *resolvedUBatch > *resolvedBatch {
		*resolvedUBatch = *resolvedBatch
	}
}

func applyVLLMAdvice(defaults *vllmRunDefaults, trustRemoteCode *bool, advice vllmPlannerAdvice, changed map[string]bool) {
	if !changed["max_model_len"] && advice.MaxModelLen != nil {
		defaults.maxModelLen = clampInt(*advice.MaxModelLen, 256, 131072)
	}
	if !changed["gpu_memory_utilization"] && advice.GPUMemoryUtilization != nil {
		defaults.gpuMemUtil = clampFloat(*advice.GPUMemoryUtilization, 0.30, 0.98)
	}
	if advice.DType != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*advice.DType))
		if trimmed == "float16" || trimmed == "bfloat16" || trimmed == "float32" {
			defaults.dtype = trimmed
		}
	}
	if advice.TensorParallelSize != nil {
		defaults.tensorParallelSize = clampInt(*advice.TensorParallelSize, 1, 16)
	}
	if advice.EnforceEager != nil {
		defaults.enforceEager = *advice.EnforceEager
	}
	if advice.OptimizationLevel != nil {
		defaults.optimizationLevel = clampInt(*advice.OptimizationLevel, 0, 3)
	}
	if advice.MaxNumSeqs != nil {
		defaults.maxNumSeqs = clampInt(*advice.MaxNumSeqs, 1, 256)
	}
	if advice.DisableCustomAllReduce != nil {
		defaults.disableCustomAllReduce = *advice.DisableCustomAllReduce
	}
	if !changed["trust_remote_code"] && advice.TrustRemoteCode != nil {
		*trustRemoteCode = *advice.TrustRemoteCode
	}
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
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

func makeTensorSplit(count int) string {
	if count <= 1 {
		return ""
	}
	parts := make([]string, 0, count)
	base := 100 / count
	remaining := 100 - base*count
	for i := 0; i < count; i++ {
		value := base
		if remaining > 0 {
			value++
			remaining--
		}
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func closestPowerOfTwo(v int) int {
	if v <= 1 {
		return 1
	}
	p := 1
	for p < v {
		p <<= 1
	}
	prev := p >> 1
	if prev < 1 {
		return p
	}
	if (p - v) < (v - prev) {
		return p
	}
	return prev
}

func hasVLLMConfig(modelDir string) bool {
	if _, err := os.Stat(filepath.Join(modelDir, "config.json")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(modelDir, "params.json")); err == nil {
		return true
	}
	return false
}

func shouldAutoEnableVLLMTrustRemoteCode(modelDir string) bool {
	configFiles := []string{
		"config.json",
		"tokenizer_config.json",
		"preprocessor_config.json",
		"processor_config.json",
	}
	for _, name := range configFiles {
		path := filepath.Join(modelDir, name)
		if configHasAutoMap(path) || configSuggestsTrustRemoteCode(path) {
			return true
		}
	}

	pythonFiles, err := filepath.Glob(filepath.Join(modelDir, "*.py"))
	if err == nil && len(pythonFiles) > 0 {
		return true
	}
	return false
}

func configSuggestsTrustRemoteCode(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}

	values := make([]string, 0, 6)
	for _, key := range []string{"model_type", "processor_class"} {
		if value, ok := payload[key].(string); ok {
			values = append(values, value)
		}
	}
	if architectures, ok := payload["architectures"].([]any); ok {
		for _, item := range architectures {
			if value, ok := item.(string); ok {
				values = append(values, value)
			}
		}
	}

	for _, value := range values {
		lowerValue := strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(lowerValue, "qwen3_5") {
			return true
		}
		if strings.Contains(lowerValue, "conditionalgeneration") && (strings.Contains(lowerValue, "qwen") || strings.Contains(lowerValue, "vl")) {
			return true
		}
		if strings.Contains(lowerValue, "vlprocessor") {
			return true
		}
	}

	return false
}

func isLikelyTextOnlyVLLMModel(modelDir string) bool {
	configNames := []string{
		"config.json",
		"preprocessor_config.json",
		"processor_config.json",
	}
	for _, name := range configNames {
		path := filepath.Join(modelDir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			continue
		}
		if configLooksMultimodal(payload) {
			return false
		}
	}
	return true
}

func configLooksMultimodal(payload map[string]any) bool {
	for key, value := range payload {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		if strings.Contains(lowerKey, "vision") || strings.Contains(lowerKey, "image") || strings.Contains(lowerKey, "video") || strings.Contains(lowerKey, "audio") {
			return true
		}
		switch v := value.(type) {
		case string:
			lowerValue := strings.ToLower(strings.TrimSpace(v))
			if strings.Contains(lowerValue, "vision") || strings.Contains(lowerValue, "image") || strings.Contains(lowerValue, "video") || strings.Contains(lowerValue, "audio") || strings.Contains(lowerValue, "vl") {
				return true
			}
		case []any:
			for _, item := range v {
				str, ok := item.(string)
				if !ok {
					continue
				}
				lowerValue := strings.ToLower(strings.TrimSpace(str))
				if strings.Contains(lowerValue, "vision") || strings.Contains(lowerValue, "image") || strings.Contains(lowerValue, "video") || strings.Contains(lowerValue, "audio") || strings.Contains(lowerValue, "vl") {
					return true
				}
			}
		}
	}
	return false
}

func configHasAutoMap(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	value, ok := payload["auto_map"]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case map[string]any:
		return len(v) > 0
	case []any:
		return len(v) > 0
	case string:
		return strings.TrimSpace(v) != ""
	default:
		return true
	}
}

type modelMetadata struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

func readModelMetadata(modelDir string) (modelMetadata, error) {
	path := filepath.Join(modelDir, "metadata.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return modelMetadata{}, fmt.Errorf("missing metadata.json at %s", path)
	}
	var meta modelMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return modelMetadata{}, fmt.Errorf("failed to parse metadata.json: %w", err)
	}
	return meta, nil
}

func hasExplicitSource(input string) bool {
	inputLower := strings.ToLower(strings.TrimSpace(input))
	return strings.HasPrefix(inputLower, "ollama:") ||
		strings.HasPrefix(inputLower, "huggingface:") ||
		strings.HasPrefix(inputLower, "hf:") ||
		strings.HasPrefix(inputLower, "modelscope:")
}

func resolveGGUFFile(modelDir string, ggufFiles []string, selected string) (string, bool, error) {
	if selected != "" {
		selectedTrimmed := strings.TrimSpace(selected)
		modelPath := selectedTrimmed
		if !filepath.IsAbs(modelPath) {
			modelPath = filepath.Join(modelDir, selectedTrimmed)
		}
		if info, err := os.Stat(modelPath); err == nil {
			if info.IsDir() {
				dirGGUF, walkErr := modelmanager.FindGGUFFiles(modelPath)
				if walkErr != nil {
					return "", false, walkErr
				}
				chosen, chooseErr := selectPreferredGGUFFile(dirGGUF)
				if chooseErr != nil {
					return "", false, chooseErr
				}
				return chosen, false, nil
			}
			if !strings.EqualFold(filepath.Ext(modelPath), ".gguf") {
				return "", false, fmt.Errorf("selected file is not a GGUF model: %s", modelPath)
			}
			return modelPath, false, nil
		} else if !os.IsNotExist(err) {
			return "", false, err
		}

		// Support quantization selectors like "Q4_K_M".
		candidates := filterGGUFFilesBySelector(ggufFiles, selectedTrimmed)
		if len(candidates) == 0 {
			return "", false, fmt.Errorf("GGUF file or selector not found: %s", selectedTrimmed)
		}
		chosen, err := selectPreferredGGUFFile(candidates)
		if err != nil {
			return "", false, err
		}
		return chosen, false, nil
	}

	chosen, err := selectPreferredGGUFFile(ggufFiles)
	if err != nil {
		return "", false, err
	}
	return chosen, true, nil
}

func selectPreferredGGUFFile(files []string) (string, error) {
	if chosen, ok := pickFirstShard(files); ok {
		return chosen, nil
	}

	preferred := []string{
		"q4_k_m",
		"q4_k_s",
		"q5_k_m",
		"q5_k_s",
		"q5",
		"q6_k",
		"q6",
		"q8_0",
		"q8",
	}

	for _, pref := range preferred {
		candidates := make([]string, 0, len(files))
		for _, file := range files {
			if strings.Contains(strings.ToLower(filepath.Base(file)), pref) {
				candidates = append(candidates, file)
			}
		}
		if len(candidates) > 0 {
			return pickSmallestFile(candidates)
		}
	}

	return pickSmallestFile(files)
}

func filterGGUFFilesBySelector(files []string, selector string) []string {
	normalized := strings.ToLower(strings.TrimSpace(selector))
	if normalized == "" {
		return nil
	}
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalizedPath := string(os.PathSeparator) + normalized + string(os.PathSeparator)

	var matched []string
	for _, file := range files {
		fileLower := strings.ToLower(file)
		baseLower := strings.ToLower(filepath.Base(file))
		baseLower = strings.ReplaceAll(baseLower, "-", "_")
		fileLowerNorm := strings.ReplaceAll(fileLower, "-", "_")
		if strings.Contains(fileLowerNorm, normalizedPath) || strings.Contains(baseLower, normalized) {
			matched = append(matched, file)
		}
	}
	return matched
}

func pickFirstShard(files []string) (string, bool) {
	var firstShard []string
	for _, file := range files {
		name := strings.ToLower(filepath.Base(file))
		if strings.Contains(name, "-00001-of-") {
			firstShard = append(firstShard, file)
		}
	}
	if len(firstShard) == 0 {
		return "", false
	}
	sort.Strings(firstShard)
	return firstShard[0], true
}

func pickSmallestFile(files []string) (string, error) {
	var (
		bestFile string
		bestSize int64
	)
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		size := info.Size()
		if bestFile == "" || size < bestSize {
			bestFile = file
			bestSize = size
		}
	}
	if bestFile == "" {
		return "", fmt.Errorf("no GGUF files available to run")
	}
	return bestFile, nil
}

func addLlamaCppLibraryPath(cmd *exec.Cmd) error {
	libDirs := candidateLibDirs()
	foundDir := ""
	for _, dir := range libDirs {
		if dir == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, "libmtmd.so.0")); err == nil {
			foundDir = dir
			break
		}
		if _, err := os.Stat(filepath.Join(dir, "libmtmd.so")); err == nil {
			foundDir = dir
			break
		}
	}
	if foundDir == "" {
		return fmt.Errorf("libmtmd.so.0 not found; reinstall the llama.cpp module or set LD_LIBRARY_PATH to the directory containing libmtmd.so.0 (searched: %s)", strings.Join(libDirs, ", "))
	}

	env := os.Environ()
	ldKey := "LD_LIBRARY_PATH="
	updated := false
	for i, kv := range env {
		if strings.HasPrefix(kv, ldKey) {
			current := strings.TrimPrefix(kv, ldKey)
			if current == "" {
				env[i] = ldKey + foundDir
			} else if !strings.Contains(current, foundDir) {
				env[i] = ldKey + foundDir + ":" + current
			}
			updated = true
			break
		}
	}
	if !updated {
		env = append(env, ldKey+foundDir)
	}
	cmd.Env = env
	return nil
}

func candidateLibDirs() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"/usr/local/llama.cpp/build/bin",
		"/usr/local/llama.cpp/build/lib",
		"/usr/local/llama.cpp/bin",
		"/usr/local/llama.cpp/lib",
		"/usr/local/llama.cpp",
		"/usr/local/lib",
		"/usr/lib",
		"/usr/lib/x86_64-linux-gnu",
		filepath.Join(home, "llama.cpp", "build", "bin"),
		filepath.Join(home, "llama.cpp", "build", "lib"),
		filepath.Join(home, "llama-b7618"),
	}
}

func RegisterSystemCommands(rootCmd *cobra.Command) {
	systemCmd := &cobra.Command{
		Use:   "system",
		Short: "System management",
	}

	initCmd := newInitCommand()

	detectCmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect hardware capabilities",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(i18n.T("Detecting hardware..."))
		},
	}

	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Show system information",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(i18n.T("System information:"))
		},
	}

	systemCmd.AddCommand(initCmd)
	systemCmd.AddCommand(detectCmd)
	systemCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(systemCmd)
}

func RegisterProviderCommands(rootCmd *cobra.Command) {
	providerCmd := &cobra.Command{
		Use:   "provider",
		Short: "Manage LLM providers",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available LLM providers",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(i18n.T("Available LLM providers:"))
			for _, provider := range llm.BuiltInProviders() {
				cmd.Printf("%s\n", i18n.T("- %s", provider))
			}
		},
	}

	providerCmd.AddCommand(listCmd)
	rootCmd.AddCommand(providerCmd)
}

func RegisterInitCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(newInitCommand())
}
