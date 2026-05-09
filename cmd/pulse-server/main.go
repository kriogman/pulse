package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kriogman/pulse/internal/api"
	"github.com/kriogman/pulse/internal/checker"
	"github.com/kriogman/pulse/internal/observability"
	"github.com/kriogman/pulse/internal/scheduler"
	"github.com/kriogman/pulse/internal/store"
	sqlitestore "github.com/kriogman/pulse/internal/store/sqlite"
	"github.com/kriogman/pulse/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("error fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	logger := observability.NewLogger(observability.LogConfig{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})
	slog.SetDefault(logger)

	observability.SetupTracer()
	metrics := observability.NewMetrics()

	db, err := sqlitestore.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := sqlitestore.RunMigrations(db, migrations.FS); err != nil {
		return err
	}

	monitorRepo := sqlitestore.NewMonitorRepository(db)
	checkRepo := sqlitestore.NewCheckRepository(db)

	sched := scheduler.New(monitorRepo, checkRepo, checker.CheckMonitor, metrics)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Scheduler en goroutine dedicada.
	go func() {
		if err := sched.Start(ctx); err != nil {
			slog.Error("scheduler error", "error", err)
		}
	}()

	// Goroutine de retención: elimina checks antiguos cada hora.
	go runCleanup(ctx, checkRepo, cfg.RetentionDays)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      buildRouter(db, monitorRepo, checkRepo, sched, metrics),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("servidor iniciado", "addr", cfg.ListenAddr, "db", cfg.DBPath)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("apagando servidor...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("apagado del servidor", "error", err)
	}

	slog.Info("servidor detenido")
	return nil
}

func buildRouter(
	db *sql.DB,
	monitors store.MonitorRepository,
	checks store.CheckRepository,
	reloader api.Reloader,
	metrics *observability.Metrics,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			req.Body = http.MaxBytesReader(w, req.Body, 1<<20) // 1 MB
			next.ServeHTTP(w, req)
		})
	})

	apiCfg := huma.DefaultConfig("Pulse API", "0.1.0")
	humaAPI := humachi.New(r, apiCfg)

	registerHealthRoutes(humaAPI, db)
	api.RegisterRoutes(humaAPI, monitors, checks, reloader)

	r.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))

	return r
}

type healthBody struct {
	Status string `json:"status" example:"ok" doc:"Estado del servicio"`
}

type healthOutput struct {
	Body healthBody
}

func registerHealthRoutes(api huma.API, db *sql.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "healthz",
		Method:      http.MethodGet,
		Path:        "/healthz",
		Summary:     "Liveness probe",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, _ *struct{}) (*healthOutput, error) {
		return &healthOutput{Body: healthBody{Status: "ok"}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "readyz",
		Method:      http.MethodGet,
		Path:        "/readyz",
		Summary:     "Readiness probe",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, _ *struct{}) (*healthOutput, error) {
		if err := db.PingContext(ctx); err != nil {
			return nil, huma.Error503ServiceUnavailable("base de datos no disponible")
		}
		return &healthOutput{Body: healthBody{Status: "ok"}}, nil
	})
}

// runCleanup elimina checks más antiguos que retentionDays cada hora.
func runCleanup(ctx context.Context, checks store.CheckRepository, retentionDays int) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().AddDate(0, 0, -retentionDays)
			n, err := checks.DeleteOlderThan(ctx, cutoff)
			if err != nil {
				slog.Error("cleanup error", "error", err)
			} else if n > 0 {
				slog.Info("checks eliminados por retención", "count", n, "dias", retentionDays)
			}
		case <-ctx.Done():
			return
		}
	}
}
