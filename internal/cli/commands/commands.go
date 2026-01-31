package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/llm"
	"github.com/zhuangbiaowei/LocalAIStack/internal/modelmanager"
	"github.com/zhuangbiaowei/LocalAIStack/internal/module"
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
		Use:   "download [model-id]",
		Short: "Download a model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modelID := args[0]
			source, _ := cmd.Flags().GetString("source")

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

			if err := mgr.DownloadModel(src, modelID, progress); err != nil {
				return fmt.Errorf("failed to download model: %w", err)
			}

			cmd.Println("\nModel downloaded successfully!")
			return nil
		},
	}
	downloadCmd.Flags().StringP("source", "s", "", "Source to download from (ollama, huggingface, modelscope)")

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
	fmt.Fprintln(writer, "NAME\tFORMAT\tSIZES\tDESCRIPTION")

	for _, model := range models {
		desc := model.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		sizes := ""
		if model.Metadata != nil {
			sizes = model.Metadata["sizes"]
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", model.ID, model.Format, sizes, desc)
	}

	writer.Flush()
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
