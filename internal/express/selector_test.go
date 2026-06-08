package express

import (
	"testing"

	"github.com/zhuangbiaowei/LocalAIStack/internal/recipe"
)

func TestRecommendRecipesPrefersNVIDIAWhenVRAMFits(t *testing.T) {
	registry := recipe.NewRegistry()
	mustAdd(t, registry, recipe.Recipe{
		ID:      "cpu",
		Name:    "cpu",
		Kind:    recipe.KindInference,
		Tier:    recipe.TierExpress,
		Status:  recipe.StatusExperimental,
		Target:  recipe.Target{Hardware: recipe.HardwareTarget{Vendor: "cpu", MinRAMGB: 16}},
		Runtime: recipe.Runtime{API: "openai-compatible"},
	})
	mustAdd(t, registry, recipe.Recipe{
		ID:     "nvidia",
		Name:   "nvidia",
		Kind:   recipe.KindInference,
		Tier:   recipe.TierExpress,
		Status: recipe.StatusExperimental,
		Target: recipe.Target{
			Hardware: recipe.HardwareTarget{Vendor: "nvidia", MinVRAMGB: 24},
			Runtime:  recipe.RuntimeTarget{Docker: true},
		},
		Runtime: recipe.Runtime{API: "openai-compatible"},
	})

	recs := RecommendRecipes(registry, HardwareFacts{
		OS:               "linux",
		RAMGB:            64,
		DockerAvailable:  true,
		ComposeAvailable: true,
		GPUs:             []GPUFact{{Vendor: "nvidia", Name: "RTX 4090", VRAMGB: 24}},
	}, Intent{Workload: "api"})

	if len(recs) < 2 {
		t.Fatalf("expected recommendations, got %+v", recs)
	}
	if recs[0].RecipeID != "nvidia" {
		t.Fatalf("expected nvidia first, got %+v", recs)
	}
	if recs[0].Confidence != "high" {
		t.Fatalf("expected high confidence, got %+v", recs[0])
	}
}

func TestRecommendRecipesMarksNVIDIARiskWhenNoGPU(t *testing.T) {
	registry := recipe.NewRegistry()
	mustAdd(t, registry, recipe.Recipe{
		ID:      "nvidia",
		Kind:    recipe.KindInference,
		Tier:    recipe.TierExpress,
		Status:  recipe.StatusExperimental,
		Target:  recipe.Target{Hardware: recipe.HardwareTarget{Vendor: "nvidia", MinVRAMGB: 24}},
		Runtime: recipe.Runtime{API: "openai-compatible"},
	})

	recs := RecommendRecipes(registry, HardwareFacts{OS: "linux", RAMGB: 32}, Intent{Workload: "api"})
	if len(recs) != 0 {
		t.Fatalf("expected no viable nvidia recommendation, got %+v", recs)
	}
}

func mustAdd(t *testing.T, registry *recipe.Registry, r recipe.Recipe) {
	t.Helper()
	if err := registry.Add(recipe.Record{Recipe: r, SourcePath: r.ID + "/recipe.yaml"}); err != nil {
		t.Fatal(err)
	}
}
