package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	Version               = "0.1.0-dev"
	EnvPrefix             = "LOCALAISTACK"
	EnvConfigFile         = "LOCALAISTACK_CONFIG"
	DefaultConfigFileName = "config"
	DefaultConfigDirName  = ".localaistack"
)

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Logging LoggingConfig `mapstructure:"logging"`
	Control ControlConfig `mapstructure:"control"`
	Storage StorageConfig `mapstructure:"storage"`
	Runtime RuntimeConfig `mapstructure:"runtime"`
}

type ServerConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	EnableTLS    bool   `mapstructure:"enable_tls"`
	TLSCertFile  string `mapstructure:"tls_cert_file"`
	TLSKeyFile   string `mapstructure:"tls_key_file"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type ControlConfig struct {
	DataDir    string `mapstructure:"data_dir"`
	PolicyFile string `mapstructure:"policy_file"`
}

type StorageConfig struct {
	ModelDir    string `mapstructure:"model_dir"`
	CacheDir    string `mapstructure:"cache_dir"`
	DownloadDir string `mapstructure:"download_dir"`
}

type RuntimeConfig struct {
	DockerEnabled bool   `mapstructure:"docker_enabled"`
	NativeEnabled bool   `mapstructure:"native_enabled"`
	DefaultMode   string `mapstructure:"default_mode"`
}

type LoadOptions struct {
	ConfigFile        string
	SearchPaths       []string
	EnvPrefix         string
	RequireConfigFile bool
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			EnableTLS:    false,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Control: ControlConfig{
			DataDir:    "/var/lib/localaistack",
			PolicyFile: "/etc/localaistack/policies.yaml",
		},
		Storage: StorageConfig{
			ModelDir:    "/var/lib/localaistack/models",
			CacheDir:    "/var/lib/localaistack/cache",
			DownloadDir: "/var/lib/localaistack/downloads",
		},
		Runtime: RuntimeConfig{
			DockerEnabled: true,
			NativeEnabled: true,
			DefaultMode:   "container",
		},
	}
}

func DefaultConfigPaths() []string {
	paths := []string{".", filepath.Join(".", "configs"), "/etc/localaistack"}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append([]string{filepath.Join(home, DefaultConfigDirName)}, paths...)
	}
	return paths
}

func LoadConfig() (*Config, error) {
	return LoadConfigWithOptions(LoadOptions{})
}

func LoadConfigWithOptions(opts LoadOptions) (*Config, error) {
	v := viper.New()
	applyDefaults(v, DefaultConfig())

	envPrefix := opts.EnvPrefix
	if envPrefix == "" {
		envPrefix = EnvPrefix
	}
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	configFile := opts.ConfigFile
	if configFile == "" {
		if envConfig := os.Getenv(EnvConfigFile); envConfig != "" {
			configFile = envConfig
		}
	}

	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName(DefaultConfigFileName)
		v.SetConfigType("yaml")
		searchPaths := opts.SearchPaths
		if len(searchPaths) == 0 {
			searchPaths = DefaultConfigPaths()
		}
		for _, path := range searchPaths {
			v.AddConfigPath(path)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if opts.RequireConfigFile || !errors.As(err, &notFound) {
			return nil, err
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func applyDefaults(v *viper.Viper, defaults *Config) {
	v.SetDefault("server.host", defaults.Server.Host)
	v.SetDefault("server.port", defaults.Server.Port)
	v.SetDefault("server.read_timeout", defaults.Server.ReadTimeout)
	v.SetDefault("server.write_timeout", defaults.Server.WriteTimeout)
	v.SetDefault("server.enable_tls", defaults.Server.EnableTLS)
	v.SetDefault("server.tls_cert_file", defaults.Server.TLSCertFile)
	v.SetDefault("server.tls_key_file", defaults.Server.TLSKeyFile)

	v.SetDefault("logging.level", defaults.Logging.Level)
	v.SetDefault("logging.format", defaults.Logging.Format)
	v.SetDefault("logging.output", defaults.Logging.Output)

	v.SetDefault("control.data_dir", defaults.Control.DataDir)
	v.SetDefault("control.policy_file", defaults.Control.PolicyFile)

	v.SetDefault("storage.model_dir", defaults.Storage.ModelDir)
	v.SetDefault("storage.cache_dir", defaults.Storage.CacheDir)
	v.SetDefault("storage.download_dir", defaults.Storage.DownloadDir)

	v.SetDefault("runtime.docker_enabled", defaults.Runtime.DockerEnabled)
	v.SetDefault("runtime.native_enabled", defaults.Runtime.NativeEnabled)
	v.SetDefault("runtime.default_mode", defaults.Runtime.DefaultMode)
}
