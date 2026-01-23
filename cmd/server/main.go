package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/zhuangbiaowei/LocalAIStack/internal/api"
	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/control"
	"github.com/zhuangbiaowei/LocalAIStack/pkg/logging"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logging
	logging.Setup(cfg.Logging)
	log.Info().Msg("Starting LocalAIStack server")
	log.Info().Str("version", config.Version).Msg("Version")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize control layer
	controlLayer, err := control.New(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize control layer")
	}

	// Start control layer
	if err := controlLayer.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to start control layer")
	}

	// Initialize API server
	apiServer := api.NewServer(cfg, controlLayer)
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Error().Err(err).Msg("API server error")
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info().Msg("Shutting down...")

	// Stop API server
	if err := apiServer.Stop(); err != nil {
		log.Error().Err(err).Msg("Error stopping API server")
	}

	// Stop control layer
	if err := controlLayer.Stop(ctx); err != nil {
		log.Error().Err(err).Msg("Error stopping control layer")
	}

	log.Info().Msg("Shutdown complete")
}
