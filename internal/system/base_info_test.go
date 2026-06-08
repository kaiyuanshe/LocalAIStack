package system

import (
	"encoding/json"
	"testing"

	"github.com/zhuangbiaowei/LocalAIStack/internal/system/info"
)

func TestFormatBaseInfo_CompactJSONIncludesSelectorFacts(t *testing.T) {
	report := info.BaseInfo{
		CPUModel:               "Test CPU",
		CPUCores:               16,
		GPU:                    "Test GPU",
		GPUDetails:             []info.GPUDetail{{Vendor: "nvidia", Name: "Test GPU", VRAMGB: 16}},
		MemoryTotal:            "32768000 kB",
		DiskTotal:              "1.0 TB",
		DiskAvailable:          "800.0 GB",
		OS:                     "linux",
		Arch:                   "amd64",
		DockerAvailable:        true,
		DockerComposeAvailable: true,
	}

	content, err := formatBaseInfo(report, "json")
	if err != nil {
		t.Fatalf("formatBaseInfo returned error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if _, ok := payload["cpu"]; !ok {
		t.Fatalf("expected cpu key in payload")
	}
	if _, ok := payload["gpu"]; !ok {
		t.Fatalf("expected gpu key in payload")
	}
	if _, ok := payload["memory"]; !ok {
		t.Fatalf("expected memory key in payload")
	}
	if _, ok := payload["disk"]; !ok {
		t.Fatalf("expected disk key in payload")
	}
	if _, ok := payload["os"]; !ok {
		t.Fatalf("expected os key in compact payload")
	}
	if _, ok := payload["runtime"]; !ok {
		t.Fatalf("expected runtime key in compact payload")
	}
	if _, ok := payload["gpu_details"]; !ok {
		t.Fatalf("expected gpu_details key in compact payload")
	}
}
