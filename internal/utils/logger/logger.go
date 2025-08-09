package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

func logger() zerolog.Logger {
	// Customize ConsoleWriter
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339, // Custom time format
	}
	consoleWriter.FormatLevel = func(i interface{}) string {
		switch i {
		case "info":
			return "\033[32m[INFO]\033[0m" // Green
		case "error":
			return "\033[31m[ERROR]\033[0m" // Red
		case "debug":
			return "\033[36m[DEBUG]\033[0m" // Cyan
		case "warn":
			return "\033[33m[WARN]\033[0m" // Yellow
		case "fatal":
			return "\033[35m[FATAL]\033[0m" // Magenta
		default:
			return fmt.Sprintf("[%s]", i)
		}
	}
	consoleWriter.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	consoleWriter.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("\033[1m%s:\033[0m", i) // Bold field names
	}
	consoleWriter.FormatFieldValue = func(i interface{}) string {
		return fmt.Sprintf("%v", i)
	}

	return zerolog.New(consoleWriter).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Logger()
}

func Info(message string, args ...interface{}) {
	logger := logger()
	if len(args) == 0 {
		logger.Info().Msg(message)
	} else {
		logger.Info().Msgf(message, args...)
	}
}

func Error(message string, args ...interface{}) {
	logger := logger()
	if len(args) == 0 {
		logger.Error().Msg(message)
	} else {
		logger.Error().Msgf(message, args...)
	}
}

func Fatal(message string, args ...interface{}) {
	logger := logger()
	if len(args) == 0 {
		logger.Fatal().Msg(message)
	} else {
		logger.Fatal().Msgf(message, args...)
	}
}

func Debug(message string, args ...interface{}) {
	logger := logger()
	if len(args) == 0 {
		logger.Debug().Msg(message)
	} else {
		logger.Debug().Msgf(message, args...)
	}
}
