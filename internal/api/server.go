package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/control"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/llm"
)

type Server struct {
	cfg          *config.Config
	controlLayer *control.ControlLayer
	server       *http.Server
}

func NewServer(cfg *config.Config, controlLayer *control.ControlLayer) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/v1/status", statusHandler)

	server := &Server{
		cfg:          cfg,
		controlLayer: controlLayer,
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler:      mux,
			ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		},
	}

	mux.HandleFunc("/api/v1/providers", server.providersHandler)

	return server
}

func (s *Server) Start() error {
	log.Info().Str("addr", s.server.Addr).Msg(i18n.T("Starting API server"))
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Stop() error {
	log.Info().Msg(i18n.T("Stopping API server"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"status":"running","version":"%s"}`, config.Version)))
}

type providersResponse struct {
	Default   string   `json:"default"`
	Providers []string `json:"providers"`
}

func (s *Server) providersHandler(w http.ResponseWriter, r *http.Request) {
	registry, err := llm.NewRegistryFromConfig(s.cfg.LLM)
	if err != nil {
		http.Error(w, i18n.T("failed to load providers: %v", err), http.StatusInternalServerError)
		return
	}

	response := providersResponse{
		Default:   s.cfg.LLM.Provider,
		Providers: registry.Providers(),
	}

	payload, err := json.Marshal(response)
	if err != nil {
		http.Error(w, i18n.T("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}
