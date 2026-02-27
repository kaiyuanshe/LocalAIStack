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
