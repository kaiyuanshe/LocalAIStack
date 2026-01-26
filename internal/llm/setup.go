package llm

import (
	"time"

	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
)

func NewRegistryFromConfig(cfg config.LLMConfig) (*Registry, error) {
	registry := NewRegistry()
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	switch cfg.Provider {
	case "siliconflow":
		if err := registry.Register(NewSiliconFlowProvider(SiliconFlowConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
			Timeout: timeout,
		})); err != nil {
			return nil, err
		}
	default:
		if err := registry.Register(NewEinoProvider(EinoConfig{
			Model:   cfg.Model,
			Timeout: timeout,
		})); err != nil {
			return nil, err
		}
	}
	return registry, nil
}
