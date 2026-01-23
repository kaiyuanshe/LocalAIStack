package commands

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
)

func init() {
	// Initialize commands package
}

func RegisterModuleCommands(rootCmd *cobra.Command) {
	moduleCmd := &cobra.Command{
		Use:   "module",
		Short: "Manage software modules",
	}

	installCmd := &cobra.Command{
		Use:   "install [module-name]",
		Short: "Install a module",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Installing module: %s\n", args[0])
		},
	}

	uninstallCmd := &cobra.Command{
		Use:   "uninstall [module-name]",
		Short: "Uninstall a module",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Uninstalling module: %s\n", args[0])
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all available modules",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Available modules:")
		},
	}

	moduleCmd.AddCommand(installCmd)
	moduleCmd.AddCommand(uninstallCmd)
	moduleCmd.AddCommand(listCmd)
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
			cmd.Printf("Starting service: %s\n", args[0])
		},
	}

	stopCmd := &cobra.Command{
		Use:   "stop [service-name]",
		Short: "Stop a service",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Stopping service: %s\n", args[0])
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status [service-name]",
		Short: "Get service status",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Service status: %s\n", args[0])
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

	pullCmd := &cobra.Command{
		Use:   "pull [model-name]",
		Short: "Download a model",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Pulling model: %s\n", args[0])
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List downloaded models",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Downloaded models:")
		},
	}

	searchCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for models",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Searching for: %s\n", args[0])
		},
	}

	modelCmd.AddCommand(pullCmd)
	modelCmd.AddCommand(listCmd)
	modelCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(modelCmd)
}

func RegisterSystemCommands(rootCmd *cobra.Command) {
	systemCmd := &cobra.Command{
		Use:   "system",
		Short: "System management",
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Collect base system info and write to ~/.localaistack/base_info.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}
			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}
			appendMode, err := cmd.Flags().GetBool("append")
			if err != nil {
				return err
			}
			if appendMode && force {
				return errors.New("cannot use --force with --append")
			}
			format, err := cmd.Flags().GetString("format")
			if err != nil {
				return err
			}

			if err := system.WriteBaseInfo(output, format, force, appendMode); err != nil {
				return err
			}

			cmd.Printf("Base system info written to %s\n", output)
			return nil
		},
	}
	initCmd.Flags().String("output", "~/.localaistack/base_info.md", "output path for base info")
	initCmd.Flags().Bool("force", false, "overwrite existing file")
	initCmd.Flags().Bool("append", false, "append to existing file")
	initCmd.Flags().String("format", "md", "output format: md or json")

	detectCmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect hardware capabilities",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Detecting hardware...")
		},
	}

	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Show system information",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("System information:")
		},
	}

	systemCmd.AddCommand(initCmd)
	systemCmd.AddCommand(detectCmd)
	systemCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(systemCmd)
}
