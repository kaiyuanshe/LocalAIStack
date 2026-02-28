package system

import (
	"encoding/json"
	"testing"

	"github.com/zhuangbiaowei/LocalAIStack/internal/system/info"
)

func TestFormatBaseInfo_CompactJSONOnlyHardwareFields(t *testing.T) {
	report := info.BaseInfo{
		CPUModel:      "Test CPU",
		CPUCores:      16,
		GPU:           "Test GPU",
		MemoryTotal:   "32768000 kB",
		DiskTotal:     "1.0 TB",
		DiskAvailable: "800.0 GB",
		OS:            "linux",
		Arch:          "amd64",
	}

	content, err := formatBaseInfo(report, "json")
	if err != nil {
		t.Fatalf("formatBaseInfo returned error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if len(payload) != 4 {
		t.Fatalf("expected 4 top-level keys, got %d", len(payload))
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
	if _, ok := payload["os"]; ok {
		t.Fatalf("did not expect os key in compact payload")
	}
}
