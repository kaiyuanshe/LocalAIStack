package rag

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildIndexAndQueryWithCitations(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(docPath, []byte("# LocalAIStack\n\nLocalAIStack uses recipes to run local inference and RAG."), 0o644); err != nil {
		t.Fatal(err)
	}
	index, err := BuildIndex(dir, 80, 10)
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}
	if len(index.Chunks) == 0 {
		t.Fatal("expected indexed chunks")
	}
	resp, err := Query(context.Background(), index, "What uses recipes?", QueryOptions{TopK: 3})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if !resp.Found || len(resp.Citations) == 0 {
		t.Fatalf("expected found response with citations: %+v", resp)
	}
}

func TestQueryNoContextFallback(t *testing.T) {
	index := Index{Version: indexVersion, Chunks: []Chunk{{ID: "1", Text: "apples oranges", SourceURI: "a.txt"}}}
	resp, err := Query(context.Background(), index, "quantum scheduling", QueryOptions{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if resp.Found {
		t.Fatalf("expected no context fallback: %+v", resp)
	}
	if len(resp.Citations) != 0 {
		t.Fatalf("expected no citations: %+v", resp.Citations)
	}
}

func TestRAGServerQuery(t *testing.T) {
	index := Index{Version: indexVersion, Chunks: []Chunk{{
		ID:        "chunk1",
		Text:      "LocalAIStack has a local RAG API.",
		SourceURI: "doc.md",
		Title:     "doc",
	}}}
	server := NewServer(ServerOptions{Index: index})
	ts := httptest.NewServer(server.Handler)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/v1/rag/query", "application/json", stringsReader(`{"query":"local RAG API"}`))
	if err != nil {
		t.Fatalf("POST returned error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var decoded QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !decoded.Found || len(decoded.Citations) == 0 {
		t.Fatalf("unexpected response: %+v", decoded)
	}
}

func stringsReader(text string) *strings.Reader {
	return strings.NewReader(text)
}
