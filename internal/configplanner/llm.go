package configplanner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/llm"
	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
)

var llmRegistryFactory = llm.NewRegistryFromConfig

type llmPlanEnvelope struct {
	Reason  string   `json:"reason,omitempty"`
	Changes []Change `json:"changes"`
}

func BuildLLMPlan(ctx context.Context, base Plan, info system.BaseInfoSummary, llmCfg config.LLMConfig) (Plan, error) {
	if strings.TrimSpace(llmCfg.Provider) == "" {
		return Plan{}, fmt.Errorf("llm provider is required")
	}
	registry, err := llmRegistryFactory(llmCfg)
	if err != nil {
		return Plan{}, err
	}
	provider, err := registry.Provider(llmCfg.Provider)
	if err != nil {
		return Plan{}, err
	}

	input := map[string]any{
		"module":   base.Module,
		"model":    base.Model,
		"hardware": info,
		"baseline": base,
		"allowed":  allowedKeysForModule(base.Module),
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return Plan{}, err
	}

	prompt := fmt.Sprintf(`You are a configuration planner for LocalAIStack.
Return valid JSON only.
Schema:
{"reason":"<short reason>","changes":[{"scope":"<scope>","key":"<key>","value":<value>,"reason":"<short reason>"}]}
Rules:
- only return keys listed in allowed.
- keep values conservative and stable.
- avoid adding unknown scopes.
Input:
%s`, string(payload))

	resp, err := provider.Generate(ctx, llm.Request{
		Model:   llmCfg.Model,
		Prompt:  prompt,
		Timeout: llmCfg.TimeoutSeconds,
	})
	if err != nil {
		return Plan{}, err
	}

	var env llmPlanEnvelope
	if err := parseLLMPlanEnvelope(resp.Text, &env); err != nil {
		return Plan{}, err
	}
	merged, err := mergeLLMChanges(base, env.Changes)
	if err != nil {
		return Plan{}, err
	}
	merged.Source = "llm"
	merged.Planner.Mode = "llm+static"
	merged.Planner.Version = "p3-b"
	if strings.TrimSpace(env.Reason) != "" {
		merged.Reason = strings.TrimSpace(env.Reason)
	}
	return merged, nil
}

func allowedKeysForModule(moduleName string) []string {
	switch strings.ToLower(strings.TrimSpace(moduleName)) {
	case "llama.cpp":
		return []string{"threads", "ctx_size", "n_gpu_layers"}
	case "vllm":
		return []string{"max_model_len", "gpu_memory_utilization"}
	case "ollama":
		return []string{"num_parallel", "keep_alive"}
	default:
		return nil
	}
}

func parseLLMPlanEnvelope(text string, out *llmPlanEnvelope) error {
	payload := extractFirstJSONObject(text)
	if payload == "" {
		return fmt.Errorf("llm plan response did not include json")
	}
	return json.Unmarshal([]byte(payload), out)
}

func mergeLLMChanges(base Plan, changes []Change) (Plan, error) {
	if len(changes) == 0 {
		return base, nil
	}
	allowed := make(map[string]bool)
	for _, key := range allowedKeysForModule(base.Module) {
		allowed[key] = true
	}
	index := make(map[string]int, len(base.Changes))
	for i, change := range base.Changes {
		index[change.Key] = i
	}

	merged := base
	for _, change := range changes {
		key := strings.TrimSpace(change.Key)
		if key == "" || !allowed[key] {
			return Plan{}, fmt.Errorf("llm returned unsupported key %q", key)
		}
		pos, ok := index[key]
		if !ok {
			return Plan{}, fmt.Errorf("llm returned key %q not found in baseline plan", key)
		}
		value, err := normalizeChangeValue(merged.Changes[pos].Value, change.Value)
		if err != nil {
			return Plan{}, fmt.Errorf("normalize value for %s: %w", key, err)
		}
		merged.Changes[pos].Value = value
		if strings.TrimSpace(change.Reason) != "" {
			merged.Changes[pos].Reason = strings.TrimSpace(change.Reason)
		}
	}
	return merged, nil
}

func normalizeChangeValue(baseline any, raw any) (any, error) {
	switch baseline.(type) {
	case int:
		n, err := toInt(raw)
		if err != nil {
			return nil, err
		}
		return n, nil
	case float64:
		n, err := toFloat(raw)
		if err != nil {
			return nil, err
		}
		return n, nil
	case string:
		return fmt.Sprintf("%v", raw), nil
	default:
		return raw, nil
	}
}

func toInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case float64:
		return int(n), nil
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, err
		}
		return int(i), nil
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(n), "%d", &parsed); err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported int type %T", v)
	}
}

func toFloat(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case json.Number:
		return n.Float64()
	case string:
		var parsed float64
		if _, err := fmt.Sscanf(strings.TrimSpace(n), "%f", &parsed); err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported float type %T", v)
	}
}

func extractFirstJSONObject(text string) string {
	start := -1
	depth := 0
	inString := false
	escaped := false
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				return text[start : i+1]
			}
		}
	}
	return ""
}
