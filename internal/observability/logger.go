package observability

import (
	"log/slog"
	"os"
)

// LogConfig configura el logger estructurado.
type LogConfig struct {
	Level  string // debug | info | warn | error
	Format string // text | json
}

// NewLogger crea un slog.Logger con formato y nivel desde configuración.
// Producción: format=json para ingestión por Loki/Datadog.
// Desarrollo: format=text para lectura humana.
func NewLogger(cfg LogConfig) *slog.Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
