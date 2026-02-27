package module

import "testing"

func TestParseLLMInstallPlanFromMarkdownJSON(t *testing.T) {
	text := "```json\n{\"mode\":\"native\",\"steps\":[\"a\",\"a\",\"b\"]}\n```"
	plan, err := parseLLMInstallPlan(text)
	if err != nil {
		t.Fatalf("parseLLMInstallPlan returned error: %v", err)
	}
	if plan.Mode != "native" {
		t.Fatalf("expected mode native, got %q", plan.Mode)
	}
	if len(plan.Steps) != 2 || plan.Steps[0] != "a" || plan.Steps[1] != "b" {
		t.Fatalf("unexpected steps: %#v", plan.Steps)
	}
}

func TestParseLLMInstallPlanWithSelectedStepsAlias(t *testing.T) {
	text := "{\"mode\":\"source\",\"selected_steps\":[\"deps\",\"build\"],\"risk_level\":\"LOW\"}"
	plan, err := parseLLMInstallPlan(text)
	if err != nil {
		t.Fatalf("parseLLMInstallPlan returned error: %v", err)
	}
	if plan.Mode != "source" {
		t.Fatalf("expected mode source, got %q", plan.Mode)
	}
	if len(plan.Steps) != 2 || plan.Steps[0] != "deps" || plan.Steps[1] != "build" {
		t.Fatalf("unexpected steps: %#v", plan.Steps)
	}
	if plan.RiskLevel != "low" {
		t.Fatalf("expected normalized risk level low, got %q", plan.RiskLevel)
	}
}

func TestApplyLLMInstallPlanRejectsUnknownSteps(t *testing.T) {
	spec := moduleInstallSpec{
		Install: map[string][]installStep{
			"native": {
				{ID: "deps", Tool: "shell"},
				{ID: "verify", Tool: "shell"},
			},
		},
	}
	defaultSteps := spec.Install["native"]
	_, _, err := applyLLMInstallPlan(spec, "native", defaultSteps, nil, llmInstallPlan{
		Mode:  "native",
		Steps: []string{"deps", "missing"},
	})
	if err == nil {
		t.Fatalf("expected unknown step validation error")
	}
}

func TestApplyLLMInstallPlanKeepsServiceSteps(t *testing.T) {
	spec := moduleInstallSpec{
		Install: map[string][]installStep{
			"native": {
				{ID: "deps", Tool: "shell"},
				{ID: "service", Tool: "shell", Expected: installExpect{Service: "active"}},
				{ID: "verify", Tool: "shell"},
			},
		},
	}
	defaultSteps := spec.Install["native"]
	_, resolved, err := applyLLMInstallPlan(spec, "native", defaultSteps, nil, llmInstallPlan{
		Mode:  "native",
		Steps: []string{"deps"},
	})
	if err != nil {
		t.Fatalf("applyLLMInstallPlan returned error: %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 steps after ensuring service step, got %d", len(resolved))
	}
	if resolved[0].ID != "deps" || resolved[1].ID != "service" {
		t.Fatalf("unexpected resolved steps order: %s, %s", resolved[0].ID, resolved[1].ID)
	}
}

func TestClassifyInstallStep(t *testing.T) {
	cases := []struct {
		step installStep
		want string
	}{
		{installStep{ID: "deps", Intent: "install dependency", Command: "apt-get install -y git", Tool: "shell"}, "dependency"},
		{installStep{ID: "download", Command: "curl -L -o pkg.tar.gz ...", Tool: "shell"}, "download"},
		{installStep{ID: "binary", Intent: "install binary package", Command: "bash install_binary.sh", Tool: "shell"}, "binary_install"},
		{installStep{ID: "source", Intent: "source build", Command: "cmake .. && make -j", Tool: "shell"}, "source_build"},
		{installStep{ID: "verify", Intent: "verify install", Command: "systemctl is-active svc", Tool: "shell"}, "verify"},
		{installStep{ID: "tmpl", Tool: "template"}, "configure"},
		{installStep{ID: "svc", Tool: "shell", Expected: installExpect{Service: "active"}}, "service"},
	}

	for _, tc := range cases {
		got := classifyInstallStep(tc.step)
		if got != tc.want {
			t.Fatalf("classifyInstallStep(%+v)=%q, want %q", tc.step, got, tc.want)
		}
	}
}

func TestBuildInstallPlannerInputIncludesModeCatalogAndSummary(t *testing.T) {
	spec := moduleInstallSpec{
		InstallModes: []string{"native", "source"},
		Preconditions: []installPrecondition{
			{ID: "check-git", Tool: "shell", Command: "which git"},
		},
		Install: map[string][]installStep{
			"native": {
				{ID: "deps", Intent: "install dependency", Tool: "shell", Command: "apt-get install -y curl"},
				{ID: "download", Intent: "download binary", Tool: "shell", Command: "curl -L -o app.tgz https://example.com/app.tgz"},
				{ID: "verify", Intent: "verify", Tool: "shell", Command: "systemctl is-active app"},
			},
			"source": {
				{ID: "deps", Intent: "install dependency", Tool: "shell", Command: "apt-get install -y build-essential"},
				{ID: "build", Intent: "source build", Tool: "shell", Command: "cmake .. && make -j"},
			},
		},
	}
	input := buildInstallPlannerInput("demo", spec, "native", spec.Install["native"])
	if input.PlannerVersion == "" {
		t.Fatalf("expected planner version to be set")
	}
	if len(input.ModeCatalog) != 2 {
		t.Fatalf("expected mode catalog size 2, got %d", len(input.ModeCatalog))
	}
	if input.CurrentModeSummary.Dependency != 1 || input.CurrentModeSummary.Download != 1 || input.CurrentModeSummary.Verify != 1 {
		t.Fatalf("unexpected current mode summary: %+v", input.CurrentModeSummary)
	}
	if len(input.Preconditions) != 1 || input.Preconditions[0].ID != "check-git" {
		t.Fatalf("unexpected precondition hints: %+v", input.Preconditions)
	}
}
