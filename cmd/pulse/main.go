// Package main es el punto de entrada del CLI pulse.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"

	"github.com/kriogman/pulse/internal/checker"
	"github.com/kriogman/pulse/internal/config"
	"github.com/kriogman/pulse/internal/domain"
	sqlitestore "github.com/kriogman/pulse/internal/store/sqlite"
	"github.com/kriogman/pulse/migrations"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "pulse",
		Short: "pulse — herramienta de health check para endpoints HTTP",
	}

	// Flag global: ruta a la BD SQLite.
	// Si el servidor está corriendo, CLI y servidor comparten la misma BD (WAL mode).
	var dbPath string
	rootCmd.PersistentFlags().StringVar(&dbPath, "db",
		envStr("PULSE_DB_PATH", "./pulse.db"),
		"ruta a la base de datos SQLite (env: PULSE_DB_PATH)")

	rootCmd.AddCommand(
		newCheckCmd(),
		newImportCmd(&dbPath),
		newListCmd(&dbPath),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── check ─────────────────────────────────────────────────────────────────

func newCheckCmd() *cobra.Command {
	var configPath string
	var timeoutSecs int
	var format string

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Chequea todos los endpoints definidos en el fichero de configuración",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("cargando configuración: %w", err)
			}
			results := checker.RunAll(cfg.Targets, timeoutSecs)
			if !checker.PrintResults(results, format) {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "pulse.yaml", "ruta al fichero YAML")
	cmd.Flags().IntVarP(&timeoutSecs, "timeout", "t", 5, "timeout en segundos por petición")
	cmd.Flags().StringVar(&format, "format", "text", "formato de salida: text | json")
	return cmd
}

// ── import ────────────────────────────────────────────────────────────────

func newImportCmd(dbPath *string) *cobra.Command {
	var fromFile string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Importa monitores desde un fichero YAML a la base de datos (idempotente por nombre)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(fromFile)
			if err != nil {
				return fmt.Errorf("cargando %s: %w", fromFile, err)
			}

			db, err := sqlitestore.Open(*dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			if err := sqlitestore.RunMigrations(db, migrations.FS); err != nil {
				return err
			}

			repo := sqlitestore.NewMonitorRepository(db)
			ctx := context.Background()

			existing, err := repo.List(ctx)
			if err != nil {
				return fmt.Errorf("listando monitores existentes: %w", err)
			}
			byName := make(map[string]*domain.Monitor, len(existing))
			for _, m := range existing {
				byName[m.Name] = m
			}

			for _, t := range cfg.Targets {
				m := yamlTargetToMonitor(t)
				now := time.Now()

				if prev, ok := byName[t.Name]; ok {
					m.ID = prev.ID
					m.CreatedAt = prev.CreatedAt
					m.UpdatedAt = now
					if err := repo.Update(ctx, m); err != nil {
						return fmt.Errorf("actualizando monitor %q: %w", t.Name, err)
					}
					fmt.Printf("actualizado: %s\n", t.Name)
				} else {
					m.ID = ulid.Make().String()
					m.CreatedAt = now
					m.UpdatedAt = now
					if err := repo.Create(ctx, m); err != nil {
						return fmt.Errorf("creando monitor %q: %w", t.Name, err)
					}
					fmt.Printf("creado:      %s\n", t.Name)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&fromFile, "from", "pulse.yaml", "fichero YAML de origen")
	return cmd
}

// ── list ──────────────────────────────────────────────────────────────────

func newListCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista los monitores configurados en la base de datos",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := sqlitestore.Open(*dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			repo := sqlitestore.NewMonitorRepository(db)
			monitors, err := repo.List(context.Background())
			if err != nil {
				return fmt.Errorf("listando monitores: %w", err)
			}

			if len(monitors) == 0 {
				fmt.Println("No hay monitores configurados. Usa 'pulse import --from pulse.yaml'.")
				return nil
			}

			fmt.Printf("%-26s %-20s %-8s %-10s %s\n", "ID", "NOMBRE", "TIPO", "ESTADO", "TARGET")
			fmt.Printf("%-26s %-20s %-8s %-10s %s\n",
				"──────────────────────────", "────────────────────",
				"────────", "──────────", "──────────────────────────────")
			for _, m := range monitors {
				status := "activo"
				if !m.Enabled {
					status = "pausado"
				}
				fmt.Printf("%-26s %-20s %-8s %-10s %s\n",
					m.ID, m.Name, string(m.Type), status, m.Target)
			}
			return nil
		},
	}
}

// ── helpers ───────────────────────────────────────────────────────────────

func yamlTargetToMonitor(t config.Target) *domain.Monitor {
	return &domain.Monitor{
		Name:        t.Name,
		Type:        domain.MonitorTypeHTTP,
		Target:      t.URL,
		IntervalSec: 60,   // default; el YAML no tiene campo interval
		TimeoutMs:   5000, // default; el YAML no tiene campo timeout
		Config: map[string]any{
			"expected_status": t.ExpectedStatus,
			"max_latency_ms":  t.MaxLatencyMs,
		},
		Enabled: true,
	}
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
