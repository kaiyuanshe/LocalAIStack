package express

import (
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/recipe"
)

type Intent struct {
	Workload   string `json:"workload,omitempty"`
	ModelSize  string `json:"model_size,omitempty"`
	MemoryMode string `json:"memory_mode,omitempty"`
}

type Recommendation struct {
	RecipeID   string   `json:"recipe_id"`
	Name       string   `json:"name,omitempty"`
	Kind       string   `json:"kind"`
	Confidence string   `json:"confidence"`
	Score      int      `json:"score"`
	Reasons    []string `json:"reasons"`
	Risks      []string `json:"risks,omitempty"`
	Fallbacks  []string `json:"fallbacks,omitempty"`
}

func RecommendRecipes(registry *recipe.Registry, facts HardwareFacts, intent Intent) []Recommendation {
	records := registry.All()
	recs := make([]Recommendation, 0, len(records))
	for _, record := range records {
		if record.Recipe.Tier != recipe.TierExpress || record.Recipe.Kind != recipe.KindInference {
			continue
		}
		rec := scoreRecipe(record.Recipe, facts, intent)
		if rec.Score <= 0 {
			continue
		}
		recs = append(recs, rec)
	}
	sort.Slice(recs, func(i, j int) bool {
		if recs[i].Score == recs[j].Score {
			return recs[i].RecipeID < recs[j].RecipeID
		}
		return recs[i].Score > recs[j].Score
	})
	return recs
}

func scoreRecipe(r recipe.Recipe, facts HardwareFacts, intent Intent) Recommendation {
	rec := Recommendation{
		RecipeID:  r.ID,
		Name:      r.Name,
		Kind:      string(r.Kind),
		Reasons:   []string{},
		Risks:     []string{},
		Fallbacks: flattenFallbacks(r.Fallbacks),
	}
	score := 40

	if len(r.Target.OS) > 0 {
		if containsFold(r.Target.OS, facts.OS) {
			score += 15
			rec.Reasons = append(rec.Reasons, "OS matches recipe target")
		} else {
			rec.Risks = append(rec.Risks, fmt.Sprintf("recipe targets %s but detected %s", strings.Join(r.Target.OS, ","), facts.OS))
			score -= 35
		}
	}

	vendor := strings.ToLower(strings.TrimSpace(r.Target.Hardware.Vendor))
	switch vendor {
	case "", "cpu":
		if r.Target.Hardware.MinRAMGB > 0 && facts.RAMGB > 0 {
			if facts.RAMGB >= r.Target.Hardware.MinRAMGB {
				score += 15
				rec.Reasons = append(rec.Reasons, fmt.Sprintf("RAM %dGB satisfies minimum %dGB", facts.RAMGB, r.Target.Hardware.MinRAMGB))
			} else {
				score -= 30
				rec.Risks = append(rec.Risks, fmt.Sprintf("RAM %dGB is below minimum %dGB", facts.RAMGB, r.Target.Hardware.MinRAMGB))
			}
		}
		if vendor == "cpu" {
			score += 10
			rec.Reasons = append(rec.Reasons, "CPU fallback recipe is generally available")
		}
	case "nvidia":
		gpu, ok := bestGPU(facts, "nvidia")
		if !ok {
			score -= 50
			rec.Risks = append(rec.Risks, "NVIDIA GPU was not detected")
		} else {
			score += 25
			rec.Reasons = append(rec.Reasons, fmt.Sprintf("detected NVIDIA GPU %s with %dGB VRAM", gpu.Name, gpu.VRAMGB))
			if r.Target.Hardware.MinVRAMGB > 0 {
				if gpu.VRAMGB >= r.Target.Hardware.MinVRAMGB {
					score += 20
					rec.Reasons = append(rec.Reasons, fmt.Sprintf("VRAM %dGB satisfies minimum %dGB", gpu.VRAMGB, r.Target.Hardware.MinVRAMGB))
				} else {
					score -= 40
					rec.Risks = append(rec.Risks, fmt.Sprintf("VRAM %dGB is below minimum %dGB", gpu.VRAMGB, r.Target.Hardware.MinVRAMGB))
				}
			}
		}
	case "apple":
		if runtime.GOOS == "darwin" || facts.MetalAvailable {
			score += 25
			rec.Reasons = append(rec.Reasons, "Apple/Metal runtime is available")
		} else {
			score -= 50
			rec.Risks = append(rec.Risks, "Apple/Metal runtime is not available")
		}
	default:
		rec.Risks = append(rec.Risks, "unknown hardware vendor target: "+vendor)
		score -= 10
	}

	if r.Target.Runtime.Docker {
		if facts.DockerAvailable && facts.ComposeAvailable {
			score += 15
			rec.Reasons = append(rec.Reasons, "Docker and Compose are available")
		} else {
			score -= 25
			rec.Risks = append(rec.Risks, "Docker or Compose is not available")
		}
	}
	if r.Target.Runtime.Native {
		score += 5
		rec.Reasons = append(rec.Reasons, "native runtime fallback is declared")
	}

	score += intentScore(r, intent, &rec)
	rec.Score = score
	rec.Confidence = confidence(score, rec.Risks)
	return rec
}

func intentScore(r recipe.Recipe, intent Intent, rec *Recommendation) int {
	score := 0
	workload := strings.ToLower(strings.TrimSpace(intent.Workload))
	if workload == "" {
		return score
	}
	switch workload {
	case "api", "openai-api", "chat":
		if strings.EqualFold(r.Runtime.API, "openai-compatible") {
			score += 10
			rec.Reasons = append(rec.Reasons, "recipe provides OpenAI-compatible API")
		}
	case "rag":
		if strings.EqualFold(r.Runtime.API, "openai-compatible") {
			score += 8
			rec.Reasons = append(rec.Reasons, "OpenAI-compatible generation can back RAG")
		}
	case "agent":
		if containsFold(r.Tags, "sglang") {
			score += 15
			rec.Reasons = append(rec.Reasons, "SGLang tag matches agent/prefix-cache workloads")
		}
	}
	return score
}

func confidence(score int, risks []string) string {
	if score >= 85 && len(risks) == 0 {
		return "high"
	}
	if score >= 55 {
		return "medium"
	}
	return "low"
}

func bestGPU(facts HardwareFacts, vendor string) (GPUFact, bool) {
	var best GPUFact
	ok := false
	for _, gpu := range facts.GPUs {
		if !strings.EqualFold(gpu.Vendor, vendor) {
			continue
		}
		if !ok || gpu.VRAMGB > best.VRAMGB {
			best = gpu
			ok = true
		}
	}
	return best, ok
}

func containsFold(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(needle)) {
			return true
		}
	}
	return false
}

func flattenFallbacks(fallbacks map[string][]string) []string {
	if len(fallbacks) == 0 {
		return nil
	}
	keys := make([]string, 0, len(fallbacks))
	for key := range fallbacks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out []string
	for _, key := range keys {
		for _, item := range fallbacks[key] {
			out = append(out, key+": "+item)
		}
	}
	return out
}
