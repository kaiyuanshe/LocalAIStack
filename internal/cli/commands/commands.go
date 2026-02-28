package commands

import (
	"context"
	"encoding/json"
	"fmt"
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
var baseInfoPromptLoader = loadBaseInfoPrompt

const (
	llamaRunRecommendationsRelativePath = "llama.cpp/RUN_PARAMS_RECOMMENDATIONS.md"
	llamaRunRecommendationsMaxBytes     = 16 * 1024
	baseInfoPromptMaxBytes              = 16 * 1024
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

			if err := mgr.DownloadModel(src, modelID, progress, modelmanager.DownloadOptions{FileHint: fileHint}); err != nil {
				return fmt.Errorf("failed to download model: %w", err)
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
		Use:   "run [model-id]",
		Short: "Run a local model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (retErr error) {
			modelID := args[0]
			source, _ := cmd.Flags().GetString("source")
			selectedFile, _ := cmd.Flags().GetString("file")
			threads, _ := cmd.Flags().GetInt("threads")
			ctxSize, _ := cmd.Flags().GetInt("ctx-size")
			gpuLayers, _ := cmd.Flags().GetInt("n-gpu-layers")
			batchSize, _ := cmd.Flags().GetInt("batch-size")
			ubatchSize, _ := cmd.Flags().GetInt("ubatch-size")
			autoBatch, _ := cmd.Flags().GetBool("auto-batch")
			smartRun, _ := cmd.Flags().GetBool("smart-run")
			smartRunDebug, _ := cmd.Flags().GetBool("smart-run-debug")
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
				vllmSmartErr := error(nil)
				if smartRun && cfg != nil {
					advice, err := suggestVLLMAdvice(cmd.Context(), cfg.LLM, modelID, modelRef, baseInfo, vllmDefaults, enableTrustRemoteCode)
					if err == nil {
						applyVLLMAdvice(&vllmDefaults, &enableTrustRemoteCode, advice, map[string]bool{
							"max_model_len":          vllmMaxModelLenChanged,
							"gpu_memory_utilization": vllmGpuMemUtilChanged,
							"trust_remote_code":      vllmTrustRemoteCodeChanged,
						})
					} else {
						vllmSmartErr = err
					}
				} else if smartRun && cfgLoadErr != nil {
					vllmSmartErr = fmt.Errorf("load smart-run config: %w", cfgLoadErr)
				}
				vllmSmartSource, vllmSmartReason, vllmSmartFatal := evaluateSmartRunOutcome(smartRun, vllmSmartErr, smartRunStrict)
				if smartRunDebug {
					printSmartRunDebug(cmd, "vllm", vllmSmartSource, vllmSmartReason)
				}
				if vllmSmartFatal != nil {
					return vllmSmartFatal
				}
				cmd.Printf("Starting vLLM server for %s\n", modelID)
				args := buildVLLMServeArgs(modelRef, host, port, vllmDefaults, enableTrustRemoteCode)
				if dryRun {
					printDryRunCommand(cmd, vllmPath, args, vllmDefaults.env)
					return nil
				}
				runCmd := exec.CommandContext(cmd.Context(), vllmPath, args...)
				if len(vllmDefaults.env) > 0 {
					runCmd.Env = append(os.Environ(), vllmDefaults.env...)
				}
				runCmd.Stdout = cmd.OutOrStdout()
				runCmd.Stderr = cmd.ErrOrStderr()
				runCmd.Stdin = cmd.InOrStdin()
				return runCmd.Run()
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
			llamaSmartErr := error(nil)
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
				} else {
					llamaSmartErr = err
				}
			} else if smartRun && cfgLoadErr != nil {
				llamaSmartErr = fmt.Errorf("load smart-run config: %w", cfgLoadErr)
			}
			llamaSmartSource, llamaSmartReason, llamaSmartFatal := evaluateSmartRunOutcome(smartRun, llamaSmartErr, smartRunStrict)
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
			runCmd := exec.CommandContext(cmd.Context(), llamaPath, argsList...)
			if err := addLlamaCppLibraryPath(runCmd); err != nil {
				return err
			}
			runCmd.Stdout = cmd.OutOrStdout()
			runCmd.Stderr = cmd.ErrOrStderr()
			runCmd.Stdin = cmd.InOrStdin()
			return runCmd.Run()
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
	runCmd.Flags().Bool("smart-run-strict", false, "Fail model run if smart-run cannot obtain valid LLM advice")
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

	modelCmd.AddCommand(searchCmd)
	modelCmd.AddCommand(downloadCmd)
	modelCmd.AddCommand(listCmd)
	modelCmd.AddCommand(runCmd)
	modelCmd.AddCommand(rmCmd)
	modelCmd.AddCommand(repairCmd)
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
	modulesRoot, err := module.FindModulesRoot()
	if err != nil {
		return "", err
	}
	docPath := filepath.Join(modulesRoot, llamaRunRecommendationsRelativePath)
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
	env := make([]string, 0, 2)
	if disableCustomAllReduce {
		env = append(env, "NCCL_IB_DISABLE=1", "NCCL_P2P_DISABLE=1")
	}

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

func buildVLLMServeArgs(modelRef, host string, port int, defaults vllmRunDefaults, enableTrustRemoteCode bool) []string {
	args := []string{"serve", modelRef, "--host", host, "--port", strconv.Itoa(port)}
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
	if enableTrustRemoteCode {
		args = append(args, "--trust-remote-code")
	}
	return args
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

func evaluateSmartRunOutcome(enabled bool, plannerErr error, strict bool) (source, reason string, fatal error) {
	if !enabled {
		return "static", "smart-run disabled", nil
	}
	if plannerErr == nil {
		return "llm", "LLM advice applied", nil
	}
	if strict {
		return "static", plannerErr.Error(), fmt.Errorf("smart-run strict mode: %w", plannerErr)
	}
	return "static", plannerErr.Error(), nil
}

func printSmartRunDebug(cmd *cobra.Command, runtimeName, source, reason string) {
	if strings.TrimSpace(reason) == "" {
		reason = "n/a"
	}
	cmd.Printf("Smart run planner (%s): source=%s reason=%s\n", runtimeName, source, reason)
}

func suggestLlamaAdvice(ctx context.Context, cfg config.LLMConfig, modelID, modelPath string, info system.BaseInfoSummary, defaults llamaRunDefaults, batch llamaBatchParams, sampling llamaSamplingParams, chatTemplateKwargs string) (llamaPlannerAdvice, error) {
	registry, err := llmRegistryFactory(cfg)
	if err != nil {
		return llamaPlannerAdvice{}, err
	}
	provider, err := registry.Provider(cfg.Provider)
	if err != nil {
		return llamaPlannerAdvice{}, err
	}
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
	resp, err := provider.Generate(ctx, llm.Request{
		Prompt:  prompt,
		Model:   cfg.Model,
		Timeout: cfg.TimeoutSeconds,
	})
	if err != nil {
		return llamaPlannerAdvice{}, err
	}
	var env smartRunAdviceEnvelope
	if err := parseSmartRunAdvice(resp.Text, &env); err != nil {
		return llamaPlannerAdvice{}, err
	}
	return env.Llama, nil
}

func suggestVLLMAdvice(ctx context.Context, cfg config.LLMConfig, modelID, modelRef string, info system.BaseInfoSummary, defaults vllmRunDefaults, trustRemoteCode bool) (vllmPlannerAdvice, error) {
	registry, err := llmRegistryFactory(cfg)
	if err != nil {
		return vllmPlannerAdvice{}, err
	}
	provider, err := registry.Provider(cfg.Provider)
	if err != nil {
		return vllmPlannerAdvice{}, err
	}
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
	resp, err := provider.Generate(ctx, llm.Request{
		Prompt:  prompt,
		Model:   cfg.Model,
		Timeout: cfg.TimeoutSeconds,
	})
	if err != nil {
		return vllmPlannerAdvice{}, err
	}
	var env smartRunAdviceEnvelope
	if err := parseSmartRunAdvice(resp.Text, &env); err != nil {
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
		if configHasAutoMap(filepath.Join(modelDir, name)) {
			return true
		}
	}

	pythonFiles, err := filepath.Glob(filepath.Join(modelDir, "*.py"))
	if err == nil && len(pythonFiles) > 0 {
		return true
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
		modelPath := selected
		if !filepath.IsAbs(modelPath) {
			modelPath = filepath.Join(modelDir, selected)
		}
		if _, err := os.Stat(modelPath); err != nil {
			return "", false, fmt.Errorf("GGUF file not found: %s", modelPath)
		}
		if !strings.EqualFold(filepath.Ext(modelPath), ".gguf") {
			return "", false, fmt.Errorf("selected file is not a GGUF model: %s", modelPath)
		}
		return modelPath, false, nil
	}

	chosen, err := selectPreferredGGUFFile(ggufFiles)
	if err != nil {
		return "", false, err
	}
	return chosen, true, nil
}

func selectPreferredGGUFFile(files []string) (string, error) {
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
