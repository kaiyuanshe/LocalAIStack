package runtime

import (
	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/module"
)

type SelectionInput struct {
	ManifestRuntime module.RuntimeConfig
	AllowedModes    []string
	Preference      string
	Config          config.RuntimeConfig
}

func SelectExecutionMode(input SelectionInput) (ExecutionMode, error) {
	available := map[string]struct{}{}
	for _, mode := range input.ManifestRuntime.Modes {
		available[mode] = struct{}{}
	}
	if len(available) == 0 {
		return "", i18n.Errorf("no runtime modes declared")
	}
	if len(input.AllowedModes) > 0 {
		allowed := map[string]struct{}{}
		for _, mode := range input.AllowedModes {
			allowed[mode] = struct{}{}
		}
		for mode := range available {
			if _, ok := allowed[mode]; !ok {
				delete(available, mode)
			}
		}
	}
	if !input.Config.DockerEnabled {
		delete(available, string(ModeContainer))
	}
	if !input.Config.NativeEnabled {
		delete(available, string(ModeNative))
	}
	if len(available) == 0 {
		return "", i18n.Errorf("no runtime modes available after policy and config filters")
	}

	preferred := input.Preference
	if preferred == "" {
		preferred = input.ManifestRuntime.Preferred
	}
	if preferred != "" {
		if _, ok := available[preferred]; ok {
			return ExecutionMode(preferred), nil
		}
	}

	defaultMode := input.Config.DefaultMode
	if defaultMode != "" {
		if _, ok := available[defaultMode]; ok {
			return ExecutionMode(defaultMode), nil
		}
	}

	for _, candidate := range []ExecutionMode{ModeContainer, ModeNative} {
		if _, ok := available[string(candidate)]; ok {
			return candidate, nil
		}
	}

	for mode := range available {
		return ExecutionMode(mode), nil
	}
	return "", i18n.Errorf("unable to select execution mode")
}
