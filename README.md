# pulse

CLI en Go que chequea el estado de endpoints HTTP en paralelo.
Pensado como proyecto de aprendizaje de Go: código comentado, sin abstracciones prematuras.

## Instalación

### Desde fuente

Requiere [Go 1.21+](https://go.dev/dl/).

```bash
git clone https://github.com/your-username/pulse.git
cd pulse
make build          # produce el binario ./pulse
```

### Cross-compilar para otras plataformas

```bash
make build-all
# Genera en dist/:
#   pulse-linux-amd64
#   pulse-darwin-arm64
#   pulse-windows-amd64.exe
```

Go incluye compiladores cruzados de serie: `GOOS` y `GOARCH` controlan el destino.
El binario resultante es **estático** (no necesita librerías del sistema en el host destino).

## Uso

```bash
pulse check [flags]

Flags:
  -c, --config string    ruta al fichero YAML (default: pulse.yaml)
  -t, --timeout int      timeout en segundos por petición (default: 5)
      --format string    formato de salida: text | json (default: text)
```

### Ejemplos

```bash
# Check básico con config por defecto
pulse check

# Config personalizada, timeout 10s
pulse check -c /etc/pulse/prod.yaml --timeout 10

# Salida JSON (útil para pipes o scripts)
pulse check --format json | jq '.[] | select(.ok == false)'
```

### Salida en modo `text`

```
[OK  ] google               https://www.google.com — HTTP 200 — 143ms
[FAIL] api-interno          https://api.example.com/health — HTTP 503 — 201ms — status esperado 200, recibido 503
```

### Salida en modo `json`

```json
[
  {
    "name": "google",
    "url": "https://www.google.com",
    "status_code": 200,
    "latency_ms": 143,
    "ok": true
  },
  {
    "name": "api-interno",
    "url": "https://api.example.com/health",
    "status_code": 503,
    "latency_ms": 201,
    "ok": false,
    "reason": "status esperado 200, recibido 503"
  }
]
```

## Formato del fichero de configuración

```yaml
targets:
  - name: google                        # Identificador legible (requerido)
    url: https://www.google.com         # URL a consultar (GET)
    expected_status: 200                # HTTP status esperado
    max_latency_ms: 500                 # Latencia máxima en ms (0 = sin límite)
```

## Exit codes

| Código | Significado                      |
|--------|----------------------------------|
| `0`    | Todos los targets pasaron        |
| `1`    | Al menos un target falló         |

Útil en pipelines de CI:

```yaml
# .gitlab-ci.yml
health-check:
  script: ./pulse check -c pulse.yaml
  # El job falla automáticamente si pulse devuelve exit code 1
```

## Desarrollo

```bash
make test           # Ejecuta todos los tests con -race detector
make run            # Ejecuta sin compilar (go run)

# Ejecutar un solo test
go test -v -run TestCheckOne_Timeout ./internal/checker/
```

## Estructura del proyecto

```
pulse/
├── cmd/pulse/main.go          # Punto de entrada; wiring de Cobra + flags
├── internal/
│   ├── config/                # Carga y validación del YAML
│   └── checker/               # Lógica HTTP + concurrencia + output
├── pulse.yaml                 # Config de ejemplo
└── Makefile
```

## 🗺️ Roadmap

Este proyecto está en desarrollo activo como proyecto de aprendizaje de Go.
Las versiones siguen [Semantic Versioning](https://semver.org/).

### ✅ v0.1.0 — MVP
- [x] CLI básico con subcomando `check` usando Cobra
- [x] Lectura de targets desde fichero YAML
- [x] Chequeo paralelo de endpoints con goroutines
- [x] Output en formato `text` y `json`
- [x] Exit codes útiles para CI (0 = OK, 1 = algún fallo)
- [x] Tests unitarios con `httptest`
- [x] Makefile con build, test y cross-compile

### 🚧 v0.2.0 — Observabilidad básica
- [ ] Logging estructurado con `log/slog` (stdlib)
- [ ] Niveles de log configurables (`--log-level`)
- [ ] Flag `--verbose` para debugging

### 📋 v0.3.0 — Resiliencia
- [ ] Retries con backoff exponencial (`cenkalti/backoff`)
- [ ] Configuración de reintentos por target
- [ ] Circuit breaker opcional para endpoints flaky

### 📋 v0.4.0 — Modo daemon
- [ ] Subcomando `serve` que ejecuta checks periódicos
- [ ] Endpoint `/metrics` compatible con Prometheus
- [ ] Endpoint `/healthz` para liveness del propio daemon
- [ ] Graceful shutdown con signals (SIGTERM, SIGINT)

### 📋 v0.5.0 — Distribución
- [ ] Dockerfile multi-stage con `FROM scratch`
- [ ] GitHub Actions: CI (test + lint con `golangci-lint`)
- [ ] GitHub Actions: release automático de binarios multiplataforma
- [ ] Publicación en GitHub Container Registry

### 💡 Ideas futuras (sin versión asignada)
- [ ] Notificaciones a Slack / Discord / webhook en fallos
- [ ] Soporte para chequeos TCP y DNS, no solo HTTP
- [ ] Validación de certificados TLS y días hasta expiración
- [ ] Helm chart para despliegue en Kubernetes
- [ ] Dashboard web embebido con estadísticas históricas

---

Las contribuciones e ideas son bienvenidas vía issues.