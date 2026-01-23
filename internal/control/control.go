package control

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
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
	log.Info().Msg("Initializing control layer")
	return &ControlLayer{cfg: cfg}, nil
}

func (c *ControlLayer) Start(ctx context.Context) error {
	log.Info().Msg("Starting control layer")

	if err := c.initHardwareDetector(ctx); err != nil {
		return fmt.Errorf("failed to initialize hardware detector: %w", err)
	}

	if err := c.initPolicyEngine(ctx); err != nil {
		return fmt.Errorf("failed to initialize policy engine: %w", err)
	}

	if err := c.initStateManager(ctx); err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	if err := c.detectHardware(ctx); err != nil {
		return fmt.Errorf("failed to detect hardware: %w", err)
	}

	if err := c.evaluatePolicies(ctx); err != nil {
		return fmt.Errorf("failed to evaluate policies: %w", err)
	}

	log.Info().Msg("Control layer started successfully")
	return nil
}

func (c *ControlLayer) Stop(ctx context.Context) error {
	log.Info().Msg("Stopping control layer")
	return nil
}

func (c *ControlLayer) initHardwareDetector(ctx context.Context) error {
	log.Info().Msg("Initializing hardware detector")
	c.detector = hardware.NewNativeDetector()
	return nil
}

func (c *ControlLayer) initPolicyEngine(ctx context.Context) error {
	log.Info().Msg("Initializing policy engine")
	engine, err := LoadPolicyEngine(c.cfg.Control.PolicyFile)
	if err != nil {
		return err
	}
	c.policyEngine = engine
	return nil
}

func (c *ControlLayer) initStateManager(ctx context.Context) error {
	log.Info().Msg("Initializing state manager")
	manager, err := NewStateManager(c.cfg.Control.DataDir)
	if err != nil {
		return err
	}
	c.stateManager = manager
	return nil
}

func (c *ControlLayer) detectHardware(ctx context.Context) error {
	if c.detector == nil {
		return fmt.Errorf("hardware detector not initialized")
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
		return fmt.Errorf("policy engine not initialized")
	}
	if c.profile == nil {
		return fmt.Errorf("hardware profile not available")
	}
	capabilities, err := c.policyEngine.Evaluate(c.profile)
	if err != nil {
		return err
	}
	c.capabilities = &capabilities
	return nil
}
