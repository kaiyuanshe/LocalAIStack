package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := LoadConfigWithOptions(LoadOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Logging.Format != "json" {
		t.Fatalf("expected default logging format json, got %s", cfg.Logging.Format)
	}
}

func TestLoadConfigEnvOverride(t *testing.T) {
	t.Setenv("LOCALAISTACK_SERVER_PORT", "9090")
	t.Setenv("LOCALAISTACK_LOGGING_LEVEL", "debug")

	cfg, err := LoadConfigWithOptions(LoadOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port 9090 from env, got %d", cfg.Server.Port)
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("expected logging level debug, got %s", cfg.Logging.Level)
	}
}

func TestLoadConfigFileOverride(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configData := []byte(`server:
  port: 7070
logging:
  format: console
`)
	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfigWithOptions(LoadOptions{ConfigFile: configPath, RequireConfigFile: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Server.Port != 7070 {
		t.Fatalf("expected port 7070 from file, got %d", cfg.Server.Port)
	}
	if cfg.Logging.Format != "console" {
		t.Fatalf("expected logging format console, got %s", cfg.Logging.Format)
	}
}
