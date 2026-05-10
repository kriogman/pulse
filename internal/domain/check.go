package domain

import "time"

// CheckStatus representa el resultado de una ejecución del health check.
type CheckStatus string

const (
	CheckStatusUp       CheckStatus = "up"
	CheckStatusDown     CheckStatus = "down"
	CheckStatusDegraded CheckStatus = "degraded"
)

// CheckStats resume el rendimiento de un monitor en un período de tiempo.
type CheckStats struct {
	TotalChecks   int
	UpCount       int
	DownCount     int
	DegradedCount int
	UptimePct     float64
	AvgDurationMs int64
	MaxDurationMs int64
}

// Check es un registro inmutable de una ejecución de health check.
// IDs como INTEGER autoincrement: alto volumen, no se exponen al exterior.
type Check struct {
	ID         int64
	MonitorID  string
	StartedAt  time.Time
	DurationMs int64
	Status     CheckStatus
	// StatusCode es nil cuando el check falló antes de recibir respuesta HTTP.
	StatusCode *int
	// Error contiene la causa del fallo; nil cuando Status es up.
	Error    *string
	Metadata map[string]any
}
