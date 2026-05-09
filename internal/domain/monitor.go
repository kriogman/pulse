package domain

import (
	"errors"
	"time"
)

var ErrMonitorNotFound = errors.New("monitor not found")

// MonitorType identifica el protocolo usado para el health check.
type MonitorType string

const (
	MonitorTypeHTTP MonitorType = "http"
	// futuro: MonitorTypeTCP, MonitorTypePing
)

// Monitor es la entidad central: un health check configurado.
// futuro: añadir UserID string para multi-tenant (migración aditiva).
type Monitor struct {
	ID          string
	Name        string
	Type        MonitorType
	Target      string
	IntervalSec int
	TimeoutMs   int
	// Config almacena parámetros específicos del tipo (ej: expected_status para HTTP).
	// Evita migrar la tabla cada vez que se añade un tipo de monitor.
	Config    map[string]any
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// HTTPConfig contiene los parámetros específicos de monitores HTTP.
// Se serializa/deserializa desde Monitor.Config.
type HTTPConfig struct {
	ExpectedStatus int   `json:"expected_status"`
	MaxLatencyMs   int64 `json:"max_latency_ms"`
}
