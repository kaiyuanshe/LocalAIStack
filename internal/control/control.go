package control

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/pkg/hardware"
)

type ControlLayer struct {
	cfg          *config.Config
	detector     hardware.Detector
	policyEngine *PolicyEngine
	stateManager *StateManager
	profile      *hardware.HardwareProfile
	capabilities *CapabilitySet
}

func New(ctx context.Context, cfg *config.Config) (*ControlLayer, error) {
	log.Info().Msg(i18n.T("Initializing control layer"))
	return &ControlLayer{cfg: cfg}, nil
}

func (c *ControlLayer) Start(ctx context.Context) error {
	log.Info().Msg(i18n.T("Starting control layer"))

	if err := c.initHardwareDetector(ctx); err != nil {
		return i18n.Errorf("failed to initialize hardware detector: %w", err)
	}

	if err := c.initPolicyEngine(ctx); err != nil {
		return i18n.Errorf("failed to initialize policy engine: %w", err)
	}

	if err := c.initStateManager(ctx); err != nil {
		return i18n.Errorf("failed to initialize state manager: %w", err)
	}

	if err := c.detectHardware(ctx); err != nil {
		return i18n.Errorf("failed to detect hardware: %w", err)
	}

	if err := c.evaluatePolicies(ctx); err != nil {
		return i18n.Errorf("failed to evaluate policies: %w", err)
	}

	log.Info().Msg(i18n.T("Control layer started successfully"))
	return nil
}

func (c *ControlLayer) Stop(ctx context.Context) error {
	log.Info().Msg(i18n.T("Stopping control layer"))
	return nil
}

func (c *ControlLayer) initHardwareDetector(ctx context.Context) error {
	log.Info().Msg(i18n.T("Initializing hardware detector"))
	c.detector = hardware.NewNativeDetector()
	return nil
}

func (c *ControlLayer) initPolicyEngine(ctx context.Context) error {
	log.Info().Msg(i18n.T("Initializing policy engine"))
	paths := policyCandidatePaths(c.cfg.Control.PolicyFile)
	var lastErr error
	for _, path := range paths {
		engine, err := LoadPolicyEngine(path)
		if err == nil {
			c.policyEngine = engine
			log.Info().Str("path", path).Msg(i18n.T("Loaded policy file"))
			return nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return lastErr
	}
	return i18n.Errorf("policy file not found")
}

func policyCandidatePaths(primary string) []string {
	seen := map[string]struct{}{}
	add := func(paths *[]string, value string) {
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		if fileExists(value) {
			*paths = append(*paths, value)
			seen[value] = struct{}{}
		}
	}

	paths := make([]string, 0, 4)
	add(&paths, primary)

	if cwd, err := os.Getwd(); err == nil {
		add(&paths, filepath.Join(cwd, "configs", "policies.yaml"))
		add(&paths, filepath.Join(cwd, "policies.yaml"))
	}

	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		add(&paths, filepath.Join(dir, "configs", "policies.yaml"))
		add(&paths, filepath.Join(dir, "policies.yaml"))
	}

	return paths
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func stateCandidateDirs(primary string) []string {
	seen := map[string]struct{}{}
	add := func(paths *[]string, value string) {
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		*paths = append(*paths, value)
		seen[value] = struct{}{}
	}

	paths := make([]string, 0, 4)
	add(&paths, primary)

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		add(&paths, filepath.Join(home, ".localaistack"))
		add(&paths, filepath.Join(home, ".localaistack", "data"))
	}

	if cwd, err := os.Getwd(); err == nil {
		add(&paths, filepath.Join(cwd, "data"))
		add(&paths, filepath.Join(cwd, ".localaistack"))
	}

	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		add(&paths, filepath.Join(dir, "data"))
		add(&paths, filepath.Join(dir, ".localaistack"))
	}

	return paths
}

func (c *ControlLayer) initStateManager(ctx context.Context) error {
	log.Info().Msg(i18n.T("Initializing state manager"))
	paths := stateCandidateDirs(c.cfg.Control.DataDir)
	var lastErr error
	for _, path := range paths {
		manager, err := NewStateManager(path)
		if err == nil {
			c.stateManager = manager
			log.Info().Str("path", path).Msg(i18n.T("State directory ready"))
			return nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return lastErr
	}
	return i18n.Errorf("state directory not available")
}

func (c *ControlLayer) detectHardware(ctx context.Context) error {
	if c.detector == nil {
		return i18n.Errorf("hardware detector not initialized")
	}
	profile, err := c.detector.Detect()
	if err != nil {
		return err
	}
	c.profile = profile
	return nil
}

func (c *ControlLayer) evaluatePolicies(ctx context.Context) error {
	if c.policyEngine == nil {
		return i18n.Errorf("policy engine not initialized")
	}
	if c.profile == nil {
		return i18n.Errorf("hardware profile not available")
	}
	capabilities, err := c.policyEngine.Evaluate(c.profile)
	if err != nil {
		return err
	}
	c.capabilities = &capabilities
	return nil
}
