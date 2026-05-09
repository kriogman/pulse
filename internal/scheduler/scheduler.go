// Package scheduler gestiona un conjunto de goroutines de health check,
// una por monitor, con recarga incremental sin gaps de monitoreo.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kriogman/pulse/internal/domain"
	"github.com/kriogman/pulse/internal/observability"
	"github.com/kriogman/pulse/internal/store"
)

// CheckFn es la función que ejecuta un health check.
// Inyectada para facilitar tests con implementaciones mock.
type CheckFn func(ctx context.Context, m *domain.Monitor) domain.Check

type runningMonitor struct {
	cancel    context.CancelFunc
	updatedAt int64 // unix epoch segundos — detecta cambios de configuración
}

// Scheduler orquesta goroutines de health check.
// Ceiling: ~1000 monitores con el modelo goroutine-por-monitor.
// Por encima: migrar a worker pool con cola. Medir con pulse_monitors_active.
type Scheduler struct {
	monitors store.MonitorRepository
	checks   store.CheckRepository
	checkFn  CheckFn
	metrics  *observability.Metrics

	mu      sync.Mutex
	running map[string]runningMonitor
}

func New(
	monitors store.MonitorRepository,
	checks store.CheckRepository,
	checkFn CheckFn,
	metrics *observability.Metrics,
) *Scheduler {
	return &Scheduler{
		monitors: monitors,
		checks:   checks,
		checkFn:  checkFn,
		metrics:  metrics,
		running:  make(map[string]runningMonitor),
	}
}

// Start carga los monitores y arranca las goroutines. Hace reload periódico cada 30s.
// Bloquea hasta que ctx se cancela y luego detiene todas las goroutines.
func (s *Scheduler) Start(ctx context.Context) error {
	if err := s.Reload(ctx); err != nil {
		return fmt.Errorf("initial scheduler reload: %w", err)
	}

	// Reload periódico hasta que la API dispare recargas directas en Fase 2.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.Reload(ctx); err != nil {
				slog.Error("scheduler reload error", "error", err)
			}
		case <-ctx.Done():
			s.stopAll()
			return nil
		}
	}
}

// Reload detecta monitores añadidos/eliminados/modificados y ajusta goroutines sin reiniciar todo.
// NO para todas las goroutines — evita gaps de monitoreo y picos de carga.
func (s *Scheduler) Reload(ctx context.Context) error {
	all, err := s.monitors.List(ctx)
	if err != nil {
		return fmt.Errorf("listing monitors for reload: %w", err)
	}

	wanted := make(map[string]*domain.Monitor, len(all))
	for _, m := range all {
		if m.Enabled {
			wanted[m.ID] = m
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Detener goroutines de monitores eliminados o desactivados.
	for id, rm := range s.running {
		if _, ok := wanted[id]; !ok {
			rm.cancel()
			delete(s.running, id)
			slog.Info("monitor detenido", "monitor_id", id)
		}
	}

	// Arrancar o reiniciar goroutines para monitores nuevos o modificados.
	for id, m := range wanted {
		existing, running := s.running[id]
		if running && existing.updatedAt == m.UpdatedAt.Unix() {
			continue // sin cambios
		}
		if running {
			existing.cancel() // configuración cambió: reiniciar
			delete(s.running, id)
		}

		mCtx, cancel := context.WithCancel(context.Background())
		s.running[id] = runningMonitor{
			cancel:    cancel,
			updatedAt: m.UpdatedAt.Unix(),
		}

		mCopy := *m // copia para capturar en la goroutine
		go s.runMonitor(mCtx, &mCopy)
		slog.Info("monitor iniciado", "monitor_id", id, "name", m.Name, "interval_sec", m.IntervalSec)
	}

	s.metrics.MonitorsActive.Set(float64(len(s.running)))
	return nil
}

// ActiveCount devuelve el número de goroutines activas. Seguro para uso concurrente.
func (s *Scheduler) ActiveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.running)
}

func (s *Scheduler) runMonitor(ctx context.Context, m *domain.Monitor) {
	// Ejecutar inmediatamente al arrancar; no esperar el primer tick.
	s.executeCheck(ctx, m)

	ticker := time.NewTicker(time.Duration(m.IntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.executeCheck(ctx, m)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) executeCheck(ctx context.Context, m *domain.Monitor) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic en ejecución de check", "monitor_id", m.ID, "panic", r)
		}
	}()

	s.metrics.ChecksInFlight.Inc()
	defer s.metrics.ChecksInFlight.Dec()

	result := s.checkFn(ctx, m)

	s.metrics.ChecksTotal.WithLabelValues(m.ID, string(result.Status)).Inc()
	s.metrics.CheckDuration.WithLabelValues(m.ID).Observe(float64(result.DurationMs) / 1000.0)

	// Timeout independiente para guardar: si el contexto del monitor fue cancelado,
	// intentamos guardar el último resultado igualmente.
	saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.checks.Save(saveCtx, &result); err != nil {
		slog.Error("guardando resultado de check", "monitor_id", m.ID, "error", err)
	}
}

func (s *Scheduler) stopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, rm := range s.running {
		rm.cancel()
		delete(s.running, id)
	}
	s.metrics.MonitorsActive.Set(0)
	slog.Info("scheduler detenido")
}
