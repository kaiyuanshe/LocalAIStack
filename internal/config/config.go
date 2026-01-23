package config

const (
	Version = "0.1.0-dev"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Control  ControlConfig  `mapstructure:"control"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Runtime  RuntimeConfig  `mapstructure:"runtime"`
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
	DataDir string `mapstructure:"data_dir"`
	PolicyFile string `mapstructure:"policy_file"`
}

type StorageConfig struct {
	ModelDir       string `mapstructure:"model_dir"`
	CacheDir       string `mapstructure:"cache_dir"`
	DownloadDir    string `mapstructure:"download_dir"`
}

type RuntimeConfig struct {
	DockerEnabled bool   `mapstructure:"docker_enabled"`
	NativeEnabled bool   `mapstructure:"native_enabled"`
	DefaultMode   string `mapstructure:"default_mode"`
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

func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()
	return cfg, nil
}
