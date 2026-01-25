package logging

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

func Setup(cfg config.LoggingConfig) {
	var output io.Writer

	if cfg.Output == "stdout" || cfg.Output == "" {
		output = os.Stdout
	} else if cfg.Output == "stderr" {
		output = os.Stderr
	} else {
		file, err := os.OpenFile(cfg.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Error().Err(err).Msg(i18n.T("Failed to open log file, using stdout"))
			output = os.Stdout
		} else {
			output = file
		}
	}

	if strings.EqualFold(cfg.Format, "console") {
		output = zerolog.ConsoleWriter{Out: output}
	}

	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		log.Warn().Str("level", cfg.Level).Msg(i18n.T("Invalid log level, using info"))
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
}
