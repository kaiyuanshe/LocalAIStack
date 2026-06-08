package modelmanager

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModelScopeSearch_ParsesWrappedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/models/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"Name":"Qwen"`) {
			t.Fatalf("expected name search payload, got: %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"Code": 200,
			"Success": true,
			"Data": {
				"Models": [{
					"Path": "Qwen",
					"Name": "Qwen3-0.6B",
					"ChineseName": "千问3-0.6B",
					"Downloads": 42,
					"Stars": 7,
					"StorageSize": 1234,
					"Libraries": ["transformer", "safetensors"],
					"ModelInfos": {"safetensor": {}}
				}]
			}
		}`))
	}))
	defer srv.Close()

	provider := NewModelScopeProvider("")
	provider.endpoint = srv.URL

	models, err := provider.Search(context.Background(), "Qwen", 5)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("unexpected model count, got %d want 1", len(models))
	}
	if models[0].ID != "Qwen/Qwen3-0.6B" {
		t.Fatalf("unexpected model id: %s", models[0].ID)
	}
	if models[0].Format != FormatSafetensors {
		t.Fatalf("unexpected format: %s", models[0].Format)
	}
	if models[0].Source != SourceModelScope {
		t.Fatalf("unexpected source: %s", models[0].Source)
	}
}

func TestModelScopeSearch_FallsBackFromVersionedQueryToOwnerPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), `"Name":"qwen3"`) || strings.Contains(string(body), `"Path":"qwen3"`) {
			_, _ = w.Write([]byte(`{"Code":200,"Success":true,"Data":{"Models":[]}}`))
			return
		}
		if strings.Contains(string(body), `"Name":"qwen"`) {
			_, _ = w.Write([]byte(`{
				"Code": 200,
				"Success": true,
				"Data": {
					"Models": [
						{"Path": "Qwen", "Name": "Qwen3-0.6B", "Libraries": ["safetensors"], "ModelInfos": {"safetensor": {}}},
						{"Path": "Qwen", "Name": "Qwen2-0.5B", "Libraries": ["safetensors"], "ModelInfos": {"safetensor": {}}}
					]
				}
			}`))
			return
		}
		t.Fatalf("unexpected request body: %s", string(body))
	}))
	defer srv.Close()

	provider := NewModelScopeProvider("")
	provider.endpoint = srv.URL

	models, err := provider.Search(context.Background(), "qwen3", 5)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("unexpected model count, got %d want 1", len(models))
	}
	if models[0].ID != "Qwen/Qwen3-0.6B" {
		t.Fatalf("unexpected model id: %s", models[0].ID)
	}
}

func TestModelScopeSearch_UsesNameSearchForCrossOwnerModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if !strings.Contains(string(body), `"Name":"Qwen3.6"`) {
			t.Fatalf("expected name search payload, got: %s", string(body))
		}
		_, _ = w.Write([]byte(`{
			"Code": 200,
			"Success": true,
			"Data": {
				"Models": [{
					"Path": "tclf90",
					"Name": "Qwen3.6-27B-AWQ",
					"Libraries": ["safetensors"]
				}]
			}
		}`))
	}))
	defer srv.Close()

	provider := NewModelScopeProvider("")
	provider.endpoint = srv.URL

	models, err := provider.Search(context.Background(), "Qwen3.6", 10)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("unexpected model count, got %d want 1", len(models))
	}
	if models[0].ID != "tclf90/Qwen3.6-27B-AWQ" {
		t.Fatalf("unexpected model id: %s", models[0].ID)
	}
}

func TestModelScopeGetModelInfo_ParsesDataWrapper(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models/Qwen/Qwen3-Embedding-0.6B-GGUF" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"Code": 200,
			"Success": true,
			"Data": {
				"Path": "Qwen",
				"Name": "Qwen3-Embedding-0.6B-GGUF",
				"Downloads": 10,
				"Stars": 3,
				"ModelInfos": {"gguf": {}}
			}
		}`))
	}))
	defer srv.Close()

	provider := NewModelScopeProvider("")
	provider.endpoint = srv.URL

	info, err := provider.GetModelInfo(context.Background(), "Qwen/Qwen3-Embedding-0.6B-GGUF")
	if err != nil {
		t.Fatalf("GetModelInfo returned error: %v", err)
	}
	if info.ID != "Qwen/Qwen3-Embedding-0.6B-GGUF" {
		t.Fatalf("unexpected model id: %s", info.ID)
	}
	if info.Format != FormatGGUF {
		t.Fatalf("unexpected format: %s", info.Format)
	}
}
