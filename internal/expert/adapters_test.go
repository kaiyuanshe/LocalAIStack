package expert

import "testing"

func TestVLLMAdapterSkipsLlamaCppQuantizationNames(t *testing.T) {
	adapter := &vLLMAdapter{}
	config := adapter.Generate(
		&LLMFitSystem{HasGPU: true, HasCUDA: true, AvailableVRAMGB: 24},
		&LLMFitModel{BestQuant: "Q8_0"},
		8192,
		1,
	)
	if _, ok := config["quantization"]; ok {
		t.Fatalf("vLLM config should not include llama.cpp quantization: %#v", config)
	}
}

func TestVLLMAdapterUsesHalfOnV100(t *testing.T) {
	adapter := &vLLMAdapter{}
	config := adapter.Generate(
		&LLMFitSystem{HasGPU: true, HasCUDA: true, AvailableVRAMGB: 16, GPUName: "Tesla V100-SXM2-16GB"},
		&LLMFitModel{},
		8192,
		1,
	)
	if got := config["dtype"]; got != "half" {
		t.Fatalf("expected dtype=half for V100, got %#v", got)
	}
}
