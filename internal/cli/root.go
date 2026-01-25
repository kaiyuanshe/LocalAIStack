package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/zhuangbiaowei/LocalAIStack/internal/cli/commands"
	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

var rootCmd = &cobra.Command{
	Use:   "localaistack",
	Short: "LocalAIStack - Local AI workstation management",
	Long:  "LocalAIStack is an open, modular software stack for building and operating local AI workstations. It provides unified control over AI development environments, inference runtimes, models, and applications.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.localaistack/config.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")

	if err := viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config")); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("Error binding config flag: %v", err))
		os.Exit(1)
	}
	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("Error binding verbose flag: %v", err))
		os.Exit(1)
	}

	commands.RegisterModuleCommands(rootCmd)
	commands.RegisterServiceCommands(rootCmd)
	commands.RegisterModelCommands(rootCmd)
	commands.RegisterProviderCommands(rootCmd)
	commands.RegisterSystemCommands(rootCmd)
	commands.RegisterInitCommand(rootCmd)
}

func initConfig() {
	viper.SetEnvPrefix(config.EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if cfgFile := viper.GetString("config"); cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else if envConfig := os.Getenv(config.EnvConfigFile); envConfig != "" {
		viper.SetConfigFile(envConfig)
	} else {
		for _, path := range config.DefaultConfigPaths() {
			viper.AddConfigPath(path)
		}
		viper.SetConfigType("yaml")
		viper.SetConfigName(config.DefaultConfigFileName)
	}

	if err := viper.ReadInConfig(); err != nil {
		if viper.GetString("config") != "" {
			fmt.Fprintf(os.Stderr, "%s\n", i18n.T("Error reading config file: %v", err))
		}
	}

	cfg, err := config.LoadConfig()
	if err == nil {
		_ = i18n.Init(cfg.I18n)
	}

	localizeCommand(rootCmd)
}

func localizeCommand(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	if cmd.Short != "" {
		cmd.Short = i18n.T(cmd.Short)
	}
	if cmd.Long != "" {
		cmd.Long = i18n.T(cmd.Long)
	}
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flag.Usage = i18n.T(flag.Usage)
	})
	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		flag.Usage = i18n.T(flag.Usage)
	})
	for _, sub := range cmd.Commands() {
		localizeCommand(sub)
	}
}
