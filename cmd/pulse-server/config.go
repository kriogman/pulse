package main

import (
	"fmt"
	"os"
	"strconv"
)

// Config contiene toda la configuración del servidor leída desde variables de entorno.
// 12-factor: env vars como fuente principal; valores hardcodeados solo para desarrollo local.
type Config struct {
	DBPath        string // PULSE_DB_PATH
	ListenAddr    string // PULSE_LISTEN_ADDR
	LogLevel      string // PULSE_LOG_LEVEL
	LogFormat     string // PULSE_LOG_FORMAT
	RetentionDays int    // PULSE_CHECKS_RETENTION_DAYS
}

func loadConfig() (*Config, error) {
	retentionDays, err := envInt("PULSE_CHECKS_RETENTION_DAYS", 90)
	if err != nil {
		return nil, fmt.Errorf("PULSE_CHECKS_RETENTION_DAYS inválido: %w", err)
	}

	cfg := &Config{
		DBPath:        envStr("PULSE_DB_PATH", "./pulse.db"),
		ListenAddr:    envStr("PULSE_LISTEN_ADDR", ":8080"),
		LogLevel:      envStr("PULSE_LOG_LEVEL", "info"),
		LogFormat:     envStr("PULSE_LOG_FORMAT", "text"),
		RetentionDays: retentionDays,
	}

	if cfg.DBPath == "" {
		return nil, fmt.Errorf("PULSE_DB_PATH no puede estar vacío")
	}

	return cfg, nil
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return n, nil
}
