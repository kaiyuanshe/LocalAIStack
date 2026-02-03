package commands

import (
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
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/llm"
	"github.com/zhuangbiaowei/LocalAIStack/internal/modelmanager"
	"github.com/zhuangbiaowei/LocalAIStack/internal/module"
	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
)

func init() {
	// Initialize commands package
}

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

	moduleCmd.AddCommand(installCmd)
	moduleCmd.AddCommand(uninstallCmd)
	moduleCmd.AddCommand(listCmd)
	moduleCmd.AddCommand(checkCmd)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			modelID := args[0]
			source, _ := cmd.Flags().GetString("source")
			selectedFile, _ := cmd.Flags().GetString("file")
			threads, _ := cmd.Flags().GetInt("threads")
			ctxSize, _ := cmd.Flags().GetInt("ctx-size")
			gpuLayers, _ := cmd.Flags().GetInt("n-gpu-layers")
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")

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

			modelDir, err := mgr.ResolveLocalModelDir(src, modelID)
			if err != nil {
				return fmt.Errorf("local model not found: %w", err)
			}

			ggufFiles, err := modelmanager.FindGGUFFiles(modelDir)
			if err != nil {
				return err
			}
			if len(ggufFiles) == 0 {
				return fmt.Errorf("no GGUF files found for %s", modelID)
			}

			modelPath, autoSelected, err := resolveGGUFFile(modelDir, ggufFiles, selectedFile)
			if err != nil {
				return err
			}
			if autoSelected && len(ggufFiles) > 1 {
				cmd.Printf("Auto-selected GGUF file: %s\n", filepath.Base(modelPath))
			}

			baseInfoPath := resolveBaseInfoPath()
			baseInfo, err := system.LoadBaseInfoSummary(baseInfoPath)
			if err != nil {
				return fmt.Errorf("failed to read base info at %s (try `./build/las system init`): %w", baseInfoPath, err)
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

			llamaPath, err := exec.LookPath("llama-server")
			if err != nil {
				return fmt.Errorf("llama-server not found in PATH (install the llama.cpp module first)")
			}

			argsList := []string{
				"--model", modelPath,
				"--threads", strconv.Itoa(defaults.threads),
				"--ctx-size", strconv.Itoa(defaults.ctxSize),
				"--n-gpu-layers", strconv.Itoa(defaults.gpuLayers),
				"--host", host,
				"--port", strconv.Itoa(port),
			}
			if defaults.tensorSplit != "" {
				argsList = append(argsList, "--tensor-split", defaults.tensorSplit)
			}

			cmd.Printf("Starting llama.cpp server for %s\n", filepath.Base(modelPath))
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
	runCmd.Flags().String("host", "0.0.0.0", "Host to bind llama.cpp server")
	runCmd.Flags().Int("port", 8080, "Port to bind llama.cpp server")

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

	modelCmd.AddCommand(searchCmd)
	modelCmd.AddCommand(downloadCmd)
	modelCmd.AddCommand(listCmd)
	modelCmd.AddCommand(runCmd)
	modelCmd.AddCommand(rmCmd)
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

type llamaRunDefaults struct {
	threads   int
	ctxSize   int
	gpuLayers int
	tensorSplit string
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
		threads:   threads,
		ctxSize:   ctxSize,
		gpuLayers: gpuLayers,
		tensorSplit: "",
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
