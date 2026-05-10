# Pulse

[![CI](https://github.com/kriogman/pulse/actions/workflows/ci.yml/badge.svg)](https://github.com/kriogman/pulse/actions/workflows/ci.yml)
[![Go 1.22+](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/dl/)

Daemon de monitorización de endpoints HTTP con dashboard web, API REST y métricas Prometheus. Se despliega como **un único binario** que embebe el frontend React, las migraciones SQL y todos los assets estáticos — sin dependencias de runtime.

## Características

- **Dashboard web** — lista de monitores, CRUD completo, pause/resume, historial de checks
- **Gráficas de latencia** — tiempos de respuesta con colores por estado (up/down/degraded), selector de período 1h / 24h / 7d / 30d
- **Estadísticas de uptime** — uptime %, respuesta media/máxima, contadores por estado
- **API REST** con OpenAPI 3.1 automático (`/openapi.json`)
- **Métricas Prometheus** en `/metrics` — histograma de latencia, checks totales, monitores activos
- **Scheduler persistente** — goroutine por monitor, reload sin reiniciar, checks inmediatamente al arrancar
- **CLI** — `check` (one-shot desde YAML), `import` (upsert a BD), `list` (tabla de monitores)
- **SQLite en modo WAL** — local-first, sin servidor de base de datos, compartido entre CLI y daemon

---

## Inicio rápido

### Docker (recomendado)

```bash
docker run -d \
  --name pulse \
  -p 8080:8080 \
  -v pulse-data:/data \
  -e PULSE_DB_PATH=/data/pulse.db \
  ghcr.io/kriogman/pulse:latest

# Abre http://localhost:8080
```

### docker-compose

```bash
git clone https://github.com/kriogman/pulse.git
cd pulse
docker-compose up -d
# Dashboard en http://localhost:8080
```

### Desde fuente

Requiere [Go 1.22+](https://go.dev/dl/) y [Node.js 20+](https://nodejs.org/).

```bash
git clone https://github.com/kriogman/pulse.git
cd pulse

# Instalar dependencias del frontend y compilar
make web-install
make web-build

# Compilar el servidor (embebe el frontend)
make build-server

# Arrancar
PULSE_DB_PATH=./pulse.db ./pulse-server
# Dashboard en http://localhost:8080
```

---

## Servidor

### Variables de entorno

| Variable | Defecto | Descripción |
|---|---|---|
| `PULSE_DB_PATH` | `./pulse.db` | Ruta al fichero SQLite |
| `PULSE_LISTEN_ADDR` | `:8080` | Dirección de escucha |
| `PULSE_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `PULSE_LOG_FORMAT` | `json` | `json` / `text` |
| `PULSE_CHECKS_RETENTION_DAYS` | `90` | Días de historial a conservar |

### Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/` | Dashboard React |
| `GET` | `/api/v1/monitors` | Lista monitores (paginado) |
| `POST` | `/api/v1/monitors` | Crea un monitor |
| `GET` | `/api/v1/monitors/{id}` | Obtiene un monitor |
| `PUT` | `/api/v1/monitors/{id}` | Actualiza un monitor |
| `DELETE` | `/api/v1/monitors/{id}` | Elimina monitor e historial |
| `POST` | `/api/v1/monitors/{id}/pause` | Pausa un monitor |
| `POST` | `/api/v1/monitors/{id}/resume` | Reactiva un monitor |
| `GET` | `/api/v1/monitors/{id}/checks` | Historial de checks |
| `GET` | `/api/v1/monitors/{id}/stats` | Estadísticas de uptime y latencia |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe (verifica BD) |
| `GET` | `/metrics` | Métricas Prometheus |
| `GET` | `/openapi.json` | Especificación OpenAPI 3.1 |

---

## CLI

El binario `pulse` permite interactuar con la misma base de datos que usa el servidor (modo WAL de SQLite permite acceso concurrente).

```bash
# Compilar
make build

# Chequeo one-shot desde fichero YAML (útil en CI)
pulse check --config pulse.yaml --format json

# Importar monitores desde YAML a la BD (idempotente por nombre)
pulse import --from pulse.yaml --db ./pulse.db

# Listar monitores en la BD
pulse list --db ./pulse.db
```

### Formato del fichero YAML

```yaml
targets:
  - name: api-produccion
    url: https://api.example.com/health
    expected_status: 200
    max_latency_ms: 500

  - name: web-principal
    url: https://example.com
    expected_status: 200
    max_latency_ms: 1000
```

### Exit codes (modo `check`)

| Código | Significado |
|---|---|
| `0` | Todos los endpoints OK |
| `1` | Al menos un endpoint falló |

```yaml
# .github/workflows/smoke.yml
- name: Health check
  run: ./pulse check -c pulse.yaml
  # El job falla si algún endpoint no responde correctamente
```

---

## Desarrollo

```bash
# Terminal 1 — API Go con hot-reload de código
make dev-server          # escucha en :8080, logs en texto

# Terminal 2 — Frontend con hot-reload
make web-dev             # Vite en :5173, proxy /api → :8080

# Tests
make test                # go test -v -race ./...
make lint                # golangci-lint

# Compilar ambos binarios para todas las plataformas
make build-all
```

### Estructura del proyecto

```
cmd/
  pulse/            → CLI: check / import / list
  pulse-server/     → Daemon: HTTP + scheduler + métricas
internal/
  domain/           → Entidades puras (Monitor, Check, CheckStats)
  store/            → Interfaces de repositorio
  store/sqlite/     → Implementación SQLite (pure-Go, sin CGO)
  api/              → Handlers huma v2, DTOs, OpenAPI
  scheduler/        → Goroutine por monitor, reload incremental
  checker/          → Lógica de check HTTP pura e inyectable
  observability/    → slog, Prometheus, OTel
migrations/         → SQL embebido con go:embed
web/                → Frontend React+TypeScript+Vite (embebido en el binario)
```

---

## Roadmap

Ver [ROADMAP.md](./ROADMAP.md) para el plan de fases futuras: alertas y notificaciones, trazas distribuidas, tipos de monitor adicionales (TCP, DNS, TLS), autenticación y alta disponibilidad.
