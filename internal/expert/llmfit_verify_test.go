//go:build ignore

package expert

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestLLMFitSystemV2(t *testing.T) {
	sys, err := callLLMFitSystem()
	if err != nil {
		t.Fatalf("callLLMFitSystem failed: %v", err)
	}
	data, _ := json.MarshalIndent(sys, "", "  ")
	fmt.Println(string(data))

	if !sys.HasGPU {
		t.Error("Expected HasGPU=true, got false - GPU detection failed")
	}
	if !sys.HasCUDA {
		t.Error("Expected HasCUDA=true, got false - CUDA detection failed")
	}
	if sys.GPUVRAMGB == 0 {
		t.Error("Expected GPUVRAMGB > 0, got 0")
	}
	if sys.GPUName == "" {
		t.Error("Expected GPUName to be set")
	}
}
