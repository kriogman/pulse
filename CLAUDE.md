# CLAUDE.md

Guía para Claude Code al trabajar con este repositorio.

## Comandos de desarrollo

```bash
# ── Go ──────────────────────────────────────────────────────────────────────
make build            # Compila servidor → ./pulse-server (incluye web-build)
make test             # go test -v -race ./...
make lint             # golangci-lint run ./...

# Servidor en modo desarrollo (sin necesidad de compilar el frontend)
make dev-server       # API en :8080, logs en texto, BD en ./pulse.db

# ── Frontend ─────────────────────────────────────────────────────────────────
make web-install      # npm install (solo primera vez o tras cambiar deps)
make web-build        # tsc + vite build → web/dist/
make web-dev          # Vite dev server en :5173, proxy de /api → :8080

# ── Ambos en dev ─────────────────────────────────────────────────────────────
# Terminal 1: make dev-server
# Terminal 2: make web-dev

# ── Tests aislados ────────────────────────────────────────────────────────────
go test -v -race -run TestNombreDelTest ./internal/api/
go test -v -race ./internal/store/sqlite/

# ── Dependencias ─────────────────────────────────────────────────────────────
go mod tidy           # Tras añadir/quitar imports
cd web && npm install # Tras cambiar package.json
```

## Arquitectura

Repositorio de un único binario servidor con arquitectura hexagonal:

```
cmd/
  pulse-server/    → Daemon: HTTP API + Scheduler + métricas

internal/
  domain/          → Entidades puras (Monitor, Check, CheckStats). Sin deps externas.
  store/           → Interfaces de repositorio (ports)
  store/sqlite/    → Implementación SQLite (adapter). WAL mode, pure-Go (sin CGO).
  api/             → Handlers huma v2, DTOs, mappers domain↔DTO
  scheduler/       → Goroutine por monitor con time.Ticker, reload incremental
  checker/         → Lógica HTTP pura: CheckMonitor(ctx, *Monitor) → Check
  observability/   → slog, Prometheus metrics, OTel noop tracer

migrations/        → SQL embebido con go:embed, runner propio (sin CGO)
web/               → Frontend React+TypeScript+Vite (embebido en el binario server)
  src/api/         → Cliente fetch tipado contra DTOs Go
  src/components/  → StatusBadge, MonitorForm, ConfirmDialog, UptimeStats, ResponseTimeChart
  src/pages/       → MonitorListPage, CheckHistoryPage
  embed.go         → //go:embed all:dist → var FS embed.FS
```

## Decisiones de diseño clave

| Decisión | Razón |
|---|---|
| SQLite + WAL | Local-first, zero-dependency, sin servidor de base de datos externo |
| `modernc.org/sqlite` | Pure-Go, sin CGO, compilación estática y cross-compile trivial |
| `MaxOpenConns=1` | SQLite serializa escrituras; un pool mayor solo añade contención |
| ULIDs para monitores | Ordenables por tiempo, URL-safe, sin coordinación |
| INTEGER autoincrement para checks | Alto volumen, no expuestos externamente |
| Timestamps como epoch (ms) | SQLite no tiene tipo nativo DATE; UnixMilli es inequívoco |
| `Reloader` interface en `internal/api` | Evita dependencia circular api → scheduler |
| `CheckFn` inyectable en Scheduler | Testabilidad sin mocks de interfaz |
| huma v2 + chi | OpenAPI 3.1 auto-generado, RFC 7807 nativo, zero boilerplate |
| Frontend embebido en binario | Deploy de un único fichero, sin servidor estático externo |

## Flujo de ejecución del servidor

```
main() → run()
  ├── loadConfig() — env vars PULSE_*
  ├── sqlitestore.Open() + RunMigrations()
  ├── scheduler.New(monitorRepo, checkRepo, checker.CheckMonitor, metrics)
  ├── sched.Start(ctx) — goroutine: Reload() cada 30s + goroutine por monitor
  ├── runCleanup(ctx, checkRepo, retentionDays) — goroutine: DELETE cada hora
  └── http.Server → buildRouter()
        ├── /healthz, /readyz
        ├── /metrics (Prometheus)
        ├── /api/v1/* (huma)
        └── /* (SPA React embebida)
```

## Tests

- **Store**: tests de integración contra SQLite real en `t.TempDir()` — no mocks
- **API**: tests de integración con `httptest`, router completo, BD real en TempDir
- **Scheduler**: mock `CheckFn` inyectado + `atomic.Int32` para contar llamadas
- **Checker**: `httptest.NewServer` real en loopback para cada escenario
- `-race` activado en `make test` para detectar data races

## Variables de entorno del servidor

| Variable | Defecto | Descripción |
|---|---|---|
| `PULSE_DB_PATH` | `./pulse.db` | Ruta al fichero SQLite |
| `PULSE_LISTEN_ADDR` | `:8080` | Dirección de escucha |
| `PULSE_LOG_LEVEL` | `info` | debug / info / warn / error |
| `PULSE_LOG_FORMAT` | `json` | json / text |
| `PULSE_CHECKS_RETENTION_DAYS` | `90` | Días de historial a conservar |
