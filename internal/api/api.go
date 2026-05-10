// Package api implementa los handlers HTTP de la API REST de Pulse.
// DTOs, mappers y handlers viven aquí; el dominio en internal/domain.
package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/oklog/ulid/v2"

	"github.com/kriogman/pulse/internal/domain"
	"github.com/kriogman/pulse/internal/store"
)

// Reloader notifica al scheduler cuando los monitores cambian.
// Implementado por *scheduler.Scheduler — definido aquí para evitar
// dependencia circular entre api y scheduler.
type Reloader interface {
	Reload(ctx context.Context) error
}

// API agrupa las dependencias de los handlers.
type API struct {
	monitors store.MonitorRepository
	checks   store.CheckRepository
	reloader Reloader
}

// RegisterRoutes registra todos los endpoints de la API en el router huma.
func RegisterRoutes(h huma.API, monitors store.MonitorRepository, checks store.CheckRepository, reloader Reloader) {
	a := &API{monitors: monitors, checks: checks, reloader: reloader}

	huma.Register(h, huma.Operation{
		OperationID: "list-monitors",
		Method:      http.MethodGet,
		Path:        "/api/v1/monitors",
		Summary:     "Lista monitores (paginado)",
		Tags:        []string{"Monitors"},
	}, a.listMonitors)

	huma.Register(h, huma.Operation{
		OperationID:   "create-monitor",
		Method:        http.MethodPost,
		Path:          "/api/v1/monitors",
		Summary:       "Crea un monitor",
		DefaultStatus: http.StatusCreated,
		Tags:          []string{"Monitors"},
	}, a.createMonitor)

	huma.Register(h, huma.Operation{
		OperationID: "get-monitor",
		Method:      http.MethodGet,
		Path:        "/api/v1/monitors/{id}",
		Summary:     "Obtiene un monitor por ID",
		Tags:        []string{"Monitors"},
	}, a.getMonitor)

	huma.Register(h, huma.Operation{
		OperationID: "update-monitor",
		Method:      http.MethodPut,
		Path:        "/api/v1/monitors/{id}",
		Summary:     "Reemplaza la configuración de un monitor",
		Tags:        []string{"Monitors"},
	}, a.updateMonitor)

	huma.Register(h, huma.Operation{
		OperationID:   "delete-monitor",
		Method:        http.MethodDelete,
		Path:          "/api/v1/monitors/{id}",
		Summary:       "Elimina un monitor y su historial",
		DefaultStatus: http.StatusNoContent,
		Tags:          []string{"Monitors"},
	}, a.deleteMonitor)

	huma.Register(h, huma.Operation{
		OperationID: "pause-monitor",
		Method:      http.MethodPost,
		Path:        "/api/v1/monitors/{id}/pause",
		Summary:     "Pausa un monitor (deja de ejecutar checks)",
		Tags:        []string{"Monitors"},
	}, a.pauseMonitor)

	huma.Register(h, huma.Operation{
		OperationID: "resume-monitor",
		Method:      http.MethodPost,
		Path:        "/api/v1/monitors/{id}/resume",
		Summary:     "Reactiva un monitor pausado",
		Tags:        []string{"Monitors"},
	}, a.resumeMonitor)

	huma.Register(h, huma.Operation{
		OperationID: "list-checks",
		Method:      http.MethodGet,
		Path:        "/api/v1/monitors/{id}/checks",
		Summary:     "Historial de checks de un monitor",
		Tags:        []string{"Checks"},
	}, a.listChecks)

	huma.Register(h, huma.Operation{
		OperationID: "monitor-stats",
		Method:      http.MethodGet,
		Path:        "/api/v1/monitors/{id}/stats",
		Summary:     "Estadísticas de uptime y latencia de un monitor",
		Tags:        []string{"Checks"},
	}, a.monitorStats)
}

// ── Handlers ──────────────────────────────────────────────────────────────

func (a *API) listMonitors(ctx context.Context, input *ListMonitorsInput) (*MonitorListResponse, error) {
	all, err := a.monitors.List(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "listando monitores", "error", err)
		return nil, huma.Error500InternalServerError("error listando monitores")
	}

	page := max(input.Page, 1)
	perPage := input.PerPage
	if perPage <= 0 {
		perPage = 50
	}

	total := len(all)
	start := (page - 1) * perPage
	end := start + perPage
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	pages := (total + perPage - 1) / perPage
	if pages == 0 {
		pages = 1
	}

	data := make([]MonitorOutput, 0, end-start)
	for _, m := range all[start:end] {
		data = append(data, monitorToOutput(m))
	}

	resp := &MonitorListResponse{}
	resp.Body.Data = data
	resp.Body.Total = total
	resp.Body.Page = page
	resp.Body.PerPage = perPage
	resp.Body.Pages = pages
	return resp, nil
}

func (a *API) createMonitor(ctx context.Context, input *CreateMonitorInput) (*MonitorResponse, error) {
	now := time.Now().Truncate(time.Second)
	m := bodyToMonitor(ulid.Make().String(), input.Body)
	m.CreatedAt = now
	m.UpdatedAt = now

	if err := a.monitors.Create(ctx, m); err != nil {
		slog.ErrorContext(ctx, "creando monitor", "error", err)
		return nil, huma.Error500InternalServerError("error creando monitor")
	}
	a.triggerReload(ctx)
	return &MonitorResponse{Body: monitorToOutput(m)}, nil
}

func (a *API) getMonitor(ctx context.Context, input *MonitorIDInput) (*MonitorResponse, error) {
	m, err := a.monitors.GetByID(ctx, input.ID)
	if err != nil {
		return nil, apiError(err)
	}
	return &MonitorResponse{Body: monitorToOutput(m)}, nil
}

func (a *API) updateMonitor(ctx context.Context, input *UpdateMonitorInput) (*MonitorResponse, error) {
	existing, err := a.monitors.GetByID(ctx, input.ID)
	if err != nil {
		return nil, apiError(err)
	}

	m := bodyToMonitor(input.ID, input.Body)
	m.CreatedAt = existing.CreatedAt
	m.UpdatedAt = time.Now().Truncate(time.Second)

	if err := a.monitors.Update(ctx, m); err != nil {
		slog.ErrorContext(ctx, "actualizando monitor", "monitor_id", input.ID, "error", err)
		return nil, huma.Error500InternalServerError("error actualizando monitor")
	}
	a.triggerReload(ctx)
	return &MonitorResponse{Body: monitorToOutput(m)}, nil
}

func (a *API) deleteMonitor(ctx context.Context, input *MonitorIDInput) (*struct{}, error) {
	if err := a.monitors.Delete(ctx, input.ID); err != nil {
		return nil, apiError(err)
	}
	a.triggerReload(ctx)
	return nil, nil
}

func (a *API) pauseMonitor(ctx context.Context, input *MonitorIDInput) (*MonitorResponse, error) {
	return a.setEnabled(ctx, input.ID, false)
}

func (a *API) resumeMonitor(ctx context.Context, input *MonitorIDInput) (*MonitorResponse, error) {
	return a.setEnabled(ctx, input.ID, true)
}

func (a *API) setEnabled(ctx context.Context, id string, enabled bool) (*MonitorResponse, error) {
	m, err := a.monitors.GetByID(ctx, id)
	if err != nil {
		return nil, apiError(err)
	}
	m.Enabled = enabled
	m.UpdatedAt = time.Now().Truncate(time.Second)
	if err := a.monitors.Update(ctx, m); err != nil {
		slog.ErrorContext(ctx, "cambiando estado del monitor", "monitor_id", id, "error", err)
		return nil, huma.Error500InternalServerError("error actualizando monitor")
	}
	a.triggerReload(ctx)
	return &MonitorResponse{Body: monitorToOutput(m)}, nil
}

func (a *API) listChecks(ctx context.Context, input *ListChecksInput) (*CheckListResponse, error) {
	from := input.From
	if from.IsZero() {
		from = time.Now().Add(-24 * time.Hour)
	}
	to := input.To
	if to.IsZero() {
		to = time.Now()
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}

	checks, err := a.checks.ListByMonitor(ctx, input.ID, from, to, limit)
	if err != nil {
		slog.ErrorContext(ctx, "listando checks", "monitor_id", input.ID, "error", err)
		return nil, huma.Error500InternalServerError("error listando checks")
	}

	data := make([]CheckOutput, 0, len(checks))
	for _, c := range checks {
		data = append(data, checkToOutput(c))
	}

	resp := &CheckListResponse{}
	resp.Body.Data = data
	resp.Body.Total = len(data)
	return resp, nil
}

func (a *API) monitorStats(ctx context.Context, input *MonitorStatsInput) (*StatsResponse, error) {
	to := time.Now()
	from := to.Add(-periodDuration(input.Period))

	s, err := a.checks.Stats(ctx, input.ID, from, to)
	if err != nil {
		slog.ErrorContext(ctx, "calculando stats", "monitor_id", input.ID, "error", err)
		return nil, huma.Error500InternalServerError("error calculando estadísticas")
	}
	period := input.Period
	if period == "" {
		period = "24h"
	}
	return &StatsResponse{Body: StatsOutput{
		TotalChecks:   s.TotalChecks,
		UpCount:       s.UpCount,
		DownCount:     s.DownCount,
		DegradedCount: s.DegradedCount,
		UptimePct:     s.UptimePct,
		AvgDurationMs: s.AvgDurationMs,
		MaxDurationMs: s.MaxDurationMs,
		Period:        period,
		From:          from,
		To:            to,
	}}, nil
}

func periodDuration(p string) time.Duration {
	switch p {
	case "1h":
		return time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────

func (a *API) triggerReload(ctx context.Context) {
	if err := a.reloader.Reload(ctx); err != nil {
		// El cambio fue persistido; el scheduler se recuperará en el siguiente reload periódico.
		slog.WarnContext(ctx, "reload del scheduler falló", "error", err)
	}
}

func apiError(err error) error {
	if errors.Is(err, domain.ErrMonitorNotFound) {
		return huma.Error404NotFound("monitor not found")
	}
	slog.Error("error inesperado en API", "error", err)
	return huma.Error500InternalServerError("internal error")
}

// ── Mappers (domain ↔ DTO) ────────────────────────────────────────────────

func monitorToOutput(m *domain.Monitor) MonitorOutput {
	return MonitorOutput{
		ID:          m.ID,
		Name:        m.Name,
		Type:        string(m.Type),
		Target:      m.Target,
		IntervalSec: m.IntervalSec,
		TimeoutMs:   m.TimeoutMs,
		Config:      m.Config,
		Enabled:     m.Enabled,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func bodyToMonitor(id string, body MonitorBody) *domain.Monitor {
	cfg := body.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	return &domain.Monitor{
		ID:          id,
		Name:        body.Name,
		Type:        domain.MonitorType(body.Type),
		Target:      body.Target,
		IntervalSec: body.IntervalSec,
		TimeoutMs:   body.TimeoutMs,
		Config:      cfg,
		Enabled:     boolVal(body.Enabled, true),
	}
}

func checkToOutput(c *domain.Check) CheckOutput {
	return CheckOutput{
		ID:         c.ID,
		MonitorID:  c.MonitorID,
		StartedAt:  c.StartedAt,
		DurationMs: c.DurationMs,
		Status:     string(c.Status),
		StatusCode: c.StatusCode,
		Error:      c.Error,
	}
}

func boolVal(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}
