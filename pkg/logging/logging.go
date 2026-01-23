package logging

import (
	"io"
	"os"

	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Setup(cfg config.LoggingConfig) {
	var output io.Writer

	if cfg.Output == "stdout" {
		output = os.Stdout
	} else {
		file, err := os.OpenFile(cfg.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Error().Err(err).Msg("Failed to open log file, using stdout")
			output = os.Stdout
		} else {
			output = file
		}
	}

	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		log.Warn().Str("level", cfg.Level).Msg("Invalid log level, using info")
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
}
