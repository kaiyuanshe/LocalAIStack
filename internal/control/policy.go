package control

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/pkg/hardware"
	"gopkg.in/yaml.v3"
)

type PolicyEngine struct {
	set PolicySet
}

type PolicySet struct {
	Policies []PolicyDefinition `yaml:"policies"`
}

type PolicyDefinition struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description,omitempty"`
	Conditions  PolicyConditions `yaml:"conditions"`
	Allow       PolicyAllow      `yaml:"allow"`
	Deny        []string         `yaml:"deny"`
}

type PolicyConditions struct {
	GPUVRAMMin  string `yaml:"gpu_vram_min,omitempty"`
	GPUVRAMMax  string `yaml:"gpu_vram_max,omitempty"`
	RAMMin      string `yaml:"ram_min,omitempty"`
	RAMMax      string `yaml:"ram_max,omitempty"`
	GPUCountMin int    `yaml:"gpu_count_min,omitempty"`
	GPUCountMax int    `yaml:"gpu_count_max,omitempty"`
	NVLink      *bool  `yaml:"nvlink,omitempty"`
	MultiGPU    *bool  `yaml:"multi_gpu,omitempty"`
}

type PolicyAllow struct {
	MaxModelSize string   `yaml:"max_model_size,omitempty"`
	Runtimes     []string `yaml:"runtimes,omitempty"`
	Features     []string `yaml:"features,omitempty"`
}

type CapabilitySet struct {
	MatchedPolicies []string `json:"matched_policies"`
	MaxModelSize    string   `json:"max_model_size"`
	Runtimes        []string `json:"runtimes"`
	Features        []string `json:"features"`
	Denied          []string `json:"denied"`
}

func LoadPolicyEngine(path string) (*PolicyEngine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read policy file: %w", err)
	}

	set := PolicySet{}
	if err := yaml.Unmarshal(data, &set); err != nil {
		return nil, fmt.Errorf("parse policy file: %w", err)
	}

	return &PolicyEngine{set: set}, nil
}

func NewPolicyEngine(set PolicySet) *PolicyEngine {
	return &PolicyEngine{set: set}
}

func (e *PolicyEngine) Evaluate(profile *hardware.HardwareProfile) (CapabilitySet, error) {
	if profile == nil {
		return CapabilitySet{}, fmt.Errorf("hardware profile is nil")
	}
	normalized := hardware.NormalizeProfile(profile)
	return e.EvaluateNormalized(normalized)
}

func (e *PolicyEngine) EvaluateNormalized(profile hardware.NormalizedProfile) (CapabilitySet, error) {
	if len(e.set.Policies) == 0 {
		return CapabilitySet{}, fmt.Errorf("no policies loaded")
	}

	var matched []PolicyDefinition
	for _, policy := range e.set.Policies {
		if policyMatches(profile, policy.Conditions) {
			matched = append(matched, policy)
		}
	}

	if len(matched) == 0 {
		return CapabilitySet{}, fmt.Errorf("no matching policies for hardware profile")
	}

	capabilities := CapabilitySet{
		MaxModelSize: "unlimited",
	}

	maxModel := modelSizeLimit("unlimited")
	runtimes := map[string]struct{}{}
	features := map[string]struct{}{}
	denied := map[string]struct{}{}

	for _, policy := range matched {
		capabilities.MatchedPolicies = append(capabilities.MatchedPolicies, policy.Name)
		if policy.Allow.MaxModelSize != "" {
			current := modelSizeLimit(policy.Allow.MaxModelSize)
			if current < maxModel {
				maxModel = current
				capabilities.MaxModelSize = policy.Allow.MaxModelSize
			}
		}
		for _, runtime := range policy.Allow.Runtimes {
			runtimes[runtime] = struct{}{}
		}
		for _, feature := range policy.Allow.Features {
			features[feature] = struct{}{}
		}
		for _, deny := range policy.Deny {
			denied[deny] = struct{}{}
		}
	}

	for deny := range denied {
		delete(runtimes, deny)
		delete(features, deny)
	}

	capabilities.Runtimes = mapKeys(runtimes)
	capabilities.Features = mapKeys(features)
	capabilities.Denied = mapKeys(denied)

	sort.Strings(capabilities.MatchedPolicies)
	sort.Strings(capabilities.Runtimes)
	sort.Strings(capabilities.Features)
	sort.Strings(capabilities.Denied)

	return capabilities, nil
}

func policyMatches(profile hardware.NormalizedProfile, conditions PolicyConditions) bool {
	if !matchesCount(profile.GPUCount, conditions.GPUCountMin, conditions.GPUCountMax) {
		return false
	}

	if conditions.NVLink != nil && profile.HasNVLink != *conditions.NVLink {
		return false
	}

	if conditions.MultiGPU != nil && profile.MultiGPU != *conditions.MultiGPU {
		return false
	}

	if !matchesBytes(profile.MaxGPUVRAMBytes, conditions.GPUVRAMMin, conditions.GPUVRAMMax) {
		return false
	}

	if !matchesBytes(profile.MemoryTotalBytes, conditions.RAMMin, conditions.RAMMax) {
		return false
	}

	return true
}

func matchesCount(value, min, max int) bool {
	if min > 0 && value < min {
		return false
	}
	if max > 0 && value > max {
		return false
	}
	return true
}

func matchesBytes(value uint64, minRaw, maxRaw string) bool {
	if minRaw != "" {
		minValue, err := parseBytes(minRaw)
		if err != nil || value < minValue {
			return false
		}
	}
	if maxRaw != "" {
		maxValue, err := parseBytes(maxRaw)
		if err != nil || value > maxValue {
			return false
		}
	}
	return true
}

func parseBytes(raw string) (uint64, error) {
	trimmed := strings.TrimSpace(strings.ToUpper(raw))
	if trimmed == "" {
		return 0, fmt.Errorf("empty size")
	}

	for _, suffix := range []string{"TB", "GB", "MB", "KB", "B"} {
		if strings.HasSuffix(trimmed, suffix) {
			number := strings.TrimSpace(strings.TrimSuffix(trimmed, suffix))
			if number == "" {
				return 0, fmt.Errorf("invalid size: %s", raw)
			}
			value, err := strconv.ParseFloat(number, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size: %w", err)
			}
			multiplier := uint64(1)
			switch suffix {
			case "TB":
				multiplier = 1024 * 1024 * 1024 * 1024
			case "GB":
				multiplier = 1024 * 1024 * 1024
			case "MB":
				multiplier = 1024 * 1024
			case "KB":
				multiplier = 1024
			}
			return uint64(value * float64(multiplier)), nil
		}
	}

	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size: %w", err)
	}
	return uint64(value), nil
}

func modelSizeLimit(raw string) float64 {
	trimmed := strings.TrimSpace(strings.ToUpper(raw))
	if trimmed == "" || trimmed == "UNLIMITED" {
		return 1e9
	}
	if strings.HasSuffix(trimmed, "B") {
		value, err := strconv.ParseFloat(strings.TrimSuffix(trimmed, "B"), 64)
		if err == nil {
			return value
		}
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 1e9
	}
	return value
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
