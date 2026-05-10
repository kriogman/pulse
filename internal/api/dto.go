package api

import "time"

// ── Stats ──────────────────────────────────────────────────────────────────

type MonitorStatsInput struct {
	ID     string `path:"id"`
	Period string `query:"period" enum:"1h,24h,7d,30d" doc:"Período de análisis (defecto: 24h)"`
}

type StatsOutput struct {
	TotalChecks   int       `json:"total_checks"`
	UpCount       int       `json:"up_count"`
	DownCount     int       `json:"down_count"`
	DegradedCount int       `json:"degraded_count"`
	UptimePct     float64   `json:"uptime_pct"`
	AvgDurationMs int64     `json:"avg_duration_ms"`
	MaxDurationMs int64     `json:"max_duration_ms"`
	Period        string    `json:"period"`
	From          time.Time `json:"from"`
	To            time.Time `json:"to"`
}

type StatsResponse struct {
	Body StatsOutput
}

// ── Request bodies ────────────────────────────────────────────────────────

// MonitorBody es el cuerpo de peticiones POST y PUT.
type MonitorBody struct {
	Name        string         `json:"name" minLength:"1" maxLength:"100" doc:"Nombre del monitor"`
	Type        string         `json:"type" enum:"http" doc:"Tipo de monitor"`
	Target      string         `json:"target" minLength:"1" doc:"URL del endpoint"`
	IntervalSec int            `json:"interval_sec" minimum:"5" maximum:"86400" doc:"Intervalo en segundos"`
	TimeoutMs   int            `json:"timeout_ms" minimum:"100" maximum:"30000" doc:"Timeout en milisegundos"`
	Config      map[string]any `json:"config,omitempty" doc:"Parámetros específicos del tipo (ej: expected_status para HTTP)"`
	// Enabled usa puntero: nil → defecto true; &false → crear pausado.
	Enabled *bool `json:"enabled,omitempty" doc:"Si el monitor está activo (defecto: true)"`
}

// ── Path / query inputs ───────────────────────────────────────────────────

type ListMonitorsInput struct {
	Page    int `query:"page" minimum:"1" doc:"Número de página (defecto: 1)"`
	PerPage int `query:"per_page" minimum:"1" maximum:"100" doc:"Elementos por página (defecto: 50)"`
}

type MonitorIDInput struct {
	ID string `path:"id" doc:"ULID del monitor"`
}

type CreateMonitorInput struct {
	Body MonitorBody
}

type UpdateMonitorInput struct {
	ID   string `path:"id"`
	Body MonitorBody
}

type ListChecksInput struct {
	ID    string    `path:"id"`
	From  time.Time `query:"from" doc:"Inicio del rango (RFC3339). Defecto: -24h"`
	To    time.Time `query:"to" doc:"Fin del rango (RFC3339). Defecto: ahora"`
	Limit int       `query:"limit" minimum:"1" maximum:"1000" doc:"Máximo de resultados (defecto: 100)"`
}

// ── Response bodies ───────────────────────────────────────────────────────

// MonitorOutput es la representación del monitor en respuestas API.
type MonitorOutput struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Target      string         `json:"target"`
	IntervalSec int            `json:"interval_sec"`
	TimeoutMs   int            `json:"timeout_ms"`
	Config      map[string]any `json:"config"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// CheckOutput es la representación de un check en respuestas API.
type CheckOutput struct {
	ID         int64     `json:"id"`
	MonitorID  string    `json:"monitor_id"`
	StartedAt  time.Time `json:"started_at"`
	DurationMs int64     `json:"duration_ms"`
	Status     string    `json:"status"`
	StatusCode *int      `json:"status_code,omitempty"`
	Error      *string   `json:"error,omitempty"`
}

// ── Response wrappers (huma serializa el campo Body como cuerpo HTTP) ─────

type MonitorResponse struct {
	Body MonitorOutput
}

type MonitorListResponse struct {
	Body struct {
		Data    []MonitorOutput `json:"data"`
		Total   int             `json:"total"`
		Page    int             `json:"page"`
		PerPage int             `json:"per_page"`
		Pages   int             `json:"pages"`
	}
}

type CheckListResponse struct {
	Body struct {
		Data  []CheckOutput `json:"data"`
		Total int           `json:"total"`
	}
}
