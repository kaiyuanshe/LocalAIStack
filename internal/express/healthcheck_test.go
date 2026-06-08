package express

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheckOpenAICompatibleServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"id": "test-model"}},
			})
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]string{"content": "OK"}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	report, err := Healthcheck(context.Background(), HealthOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("Healthcheck returned error: %v", err)
	}
	if !report.OpenAICompatible || !report.ModelsOK || !report.ChatOK {
		t.Fatalf("expected compatible health report: %+v", report)
	}
	if report.ModelCount != 1 {
		t.Fatalf("expected model count 1, got %d", report.ModelCount)
	}
}

func TestBenchmarkOpenAICompatibleServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"id": "test-model"}},
			})
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]string{"content": "local ai works"}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	report, err := Benchmark(context.Background(), BenchmarkOptions{BaseURL: server.URL, Requests: 2})
	if err != nil {
		t.Fatalf("Benchmark returned error: %v", err)
	}
	if report.Successful != 2 || report.Failed != 0 {
		t.Fatalf("unexpected benchmark report: %+v", report)
	}
	if report.ApproxTokens == 0 {
		t.Fatalf("expected approximate tokens: %+v", report)
	}
}
