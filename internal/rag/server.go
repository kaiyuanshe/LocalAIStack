package rag

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ServerOptions struct {
	Addr    string
	Index   Index
	BaseURL string
	Model   string
	TopK    int
}

func NewServer(opts ServerOptions) *http.Server {
	if strings.TrimSpace(opts.Addr) == "" {
		opts.Addr = "127.0.0.1:18080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"chunks": len(opts.Index.Chunks),
		})
	})
	mux.HandleFunc("/v1/rag/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Query string `json:"query"`
			TopK  int    `json:"top_k,omitempty"`
			Model string `json:"model,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("decode request: %v", err), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Query) == "" {
			http.Error(w, "query is required", http.StatusBadRequest)
			return
		}
		topK := opts.TopK
		if req.TopK > 0 {
			topK = req.TopK
		}
		model := opts.Model
		if strings.TrimSpace(req.Model) != "" {
			model = req.Model
		}
		resp, err := Query(r.Context(), opts.Index, req.Query, QueryOptions{
			TopK:    topK,
			BaseURL: opts.BaseURL,
			Model:   model,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return &http.Server{
		Addr:         opts.Addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 90 * time.Second,
	}
}
