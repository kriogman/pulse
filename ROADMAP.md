# Pulse — Roadmap

Pulse empezó como un CLI one-shot de health check y ha evolucionado a un daemon de monitorización completo: servidor HTTP persistente, scheduler con goroutines, API REST con OpenAPI, frontend React embebido en el propio binario Go, gráficas de latencia y estadísticas de uptime.

**Stack actual:** Go 1.22 · huma v2 · SQLite WAL (pure-Go) · Prometheus · OTel · React 18 · TypeScript · Vite · Tailwind CSS v4 · Recharts · Docker distroless

---

## Completado

| Fase | Qué se construyó |
|---|---|
| 0 · Bootstrap | Repo multi-binario, Makefile, CI (lint/test/govulncheck/docker), Dockerfile distroless |
| 1 · Scheduler | Dominio hexagonal, SQLite WAL, migrations embebidas, scheduler goroutine-por-monitor, métricas Prometheus |
| 2 · API REST | 9 endpoints CRUD + pause/resume + stats, OpenAPI 3.1 automático, RFC 7807, tests integración |
| 3 · Dashboard | SPA React+TS embebida en el binario, CRUD monitores, historial de checks, hot-reload en dev |
| 4 · Gráficas | Stats endpoint (uptime %, avg/max latencia), selector de período, Recharts con colores por estado |

---

## Mejoras pendientes en lo ya construido

Antes de abrir nuevas fases hay deuda técnica y UX concreta:

### Dashboard
- [ ] Paginación del listado de monitores (el endpoint ya la soporta, falta en la UI)
- [ ] Búsqueda y filtro por nombre / tipo / estado en la tabla
- [ ] Ordenación de columnas (click en cabecera)
- [ ] Vista de estado en tiempo real: badge "último check" con tiempo relativo ("hace 2 min")
- [ ] Indicador visual cuando un monitor lleva N checks consecutivos en `down`
- [ ] Operaciones en lote: pausar / eliminar monitores seleccionados
- [ ] Responsive / mobile: la tabla actual no escala bien en pantallas pequeñas

### Checks e historial
- [ ] Exportar historial como CSV desde la UI
- [ ] Agregar checks por hora/día en vistas de 7d/30d para reducir puntos en la gráfica
- [ ] Mini-sparkline de uptime en la fila del dashboard (últimas 24h)
- [ ] Mostrar percentil p95 de latencia en las stats cards

### API y backend
- [ ] Endpoint `GET /api/v1/status` con resumen global (N monitores, N up/down/degraded)
- [ ] Soporte de etiquetas (tags) en monitores para agrupar y filtrar
- [ ] `config_json` validado contra esquema por tipo al crear/actualizar un monitor
- [ ] Test del endpoint de stats (`/monitors/{id}/stats`)

---

## Fases futuras

### Fase 5 — Alertas y Notificaciones
> *Objetivo: enterarse de los fallos sin abrir el dashboard.*

**Modelo de datos**
- Tabla `alert_rules`: monitor_id, condición (tipo: `consecutive_failures | uptime_below | latency_above`), umbral, ventana temporal, habilitada
- Tabla `notification_channels`: tipo (`webhook | slack | email`), configuración JSON cifrada en reposo
- Tabla `alert_events`: rule_id, estado (`firing | resolved`), primer_fallo, último_fallo, notificaciones enviadas

**Motor de alertas**
- El scheduler evalúa reglas tras cada check; transiciones de estado con cooldown configurable para evitar alert flapping
- Máquina de estados: `inactive → pending (N fallos) → firing → resolved`
- Ventana de silencio por monitor (mantenimiento programado): `POST /api/v1/monitors/{id}/silence`

**Canales de notificación**
- Webhook genérico (POST JSON con payload estándar)
- Slack via Incoming Webhook
- Email via SMTP (con plantilla HTML)
- Todas las notificaciones son reintenables con backoff si el canal falla

**API**
- CRUD de `alert_rules` y `notification_channels`
- `GET /api/v1/alerts/active` — alertas actualmente en estado `firing`

**UI**
- Sección "Alertas" en el dashboard con badge contador
- Formulario de reglas por monitor en la página de edición
- Historial de eventos de alerta con timeline

---

### Fase 6 — Observabilidad
> *Objetivo: Pulse observable por sí mismo e integrable con el stack SRE existente.*

**Trazas distribuidas (OTel)**
- Activar el exportador OTLP (ya hay un noop tracer cableado); configurar via `OTEL_EXPORTER_OTLP_ENDPOINT`
- Cada ejecución de check como span hijo del scheduler: atributos `monitor.id`, `monitor.type`, `check.status`, `http.status_code`, `check.duration_ms`
- Propagar trace context a las peticiones HTTP salientes (`traceparent` header) para correlacionar con trazas del servicio monitoreado
- Trazar también las operaciones de la API REST (span por request)

**Logging**
- Añadir `request_id` en cada log de request HTTP (ya existe via chi RequestID middleware, falta propagarlo al contexto del logger)
- Propagar `trace_id` y `span_id` de OTel al logger slog para correlacionar logs con trazas en Loki/Grafana
- Formato de log compatible con Grafana Loki (timestamp + level + message + campos JSON)

**Métricas Prometheus ampliadas**
- Label `monitor_name` en `pulse_checks_total` y `pulse_check_duration_seconds`
- Nueva métrica `pulse_alert_notifications_total{channel, status}` (éxito / fallo de notificaciones)
- Nueva métrica `pulse_slo_error_budget_remaining{monitor_id}` (porcentaje de error budget restante)
- Buckets del histograma de latencia ajustados a rangos web reales: 50ms, 100ms, 200ms, 500ms, 1s, 2s, 5s

**SLO tracking**
- Campo `slo_target_pct` en el modelo `Monitor` (ej: 99.9)
- Endpoint `GET /api/v1/monitors/{id}/slo?window=30d` → error budget consumido, burn rate 1h/6h/24h
- UI: barra de error budget en la página de historial del monitor

**Grafana**
- Directorio `grafana/` con dashboard JSON provisionable (importar vía API o `docker-compose`)
- Panel de latencia p50/p95/p99 por monitor
- Panel de uptime heatmap (verde/rojo por hora del día)
- Panel de burn rate con alertas de SLO

---

### Fase 7 — Resiliencia y Reliability
> *Objetivo: el scheduler y los checks se comportan bien bajo fallos parciales y carga.*

**Retry inteligente en el checker**
- Configuración por monitor: `retry_count` (defecto 0) y `retry_backoff_ms` (defecto 500)
- Solo se reintenta en errores de red / timeout; un HTTP 500 cuenta como fallo inmediato
- El check guardado en BD refleja el resultado del último intento, pero `duration_ms` incluye tiempo total

**Circuit breaker por monitor**
- Tras N fallos consecutivos (configurable), el monitor entra en `open` state: checks reducidos a 1/5 del intervalo normal
- Tras un check exitoso en `open` state, vuelve a `closed` y recupera el intervalo original
- Evita saturar servicios ya degradados con health checks frecuentes

**Jitter en el arranque del scheduler**
- Al iniciar, los monitores no empiezan todos a la vez: jitter aleatorio de ±20% del intervalo antes del primer tick
- Evita el "thundering herd" cuando el servidor arranca con 50+ monitores

**Graceful reload**
- El reload del scheduler espera a que los checks en vuelo terminen antes de cancelar una goroutine (timeout de 1 ciclo de intervalo)
- Cambios de `enabled: false` vía API pausan la goroutine; cambios de intervalo la reinician con jitter

**Resiliencia de la base de datos**
- Checkpoint WAL automático: tarea horaria con `PRAGMA wal_checkpoint(TRUNCATE)` para evitar que el WAL crezca indefinidamente
- Comando `pulse backup --dest ./pulse-backup.db` en el CLI (usa `VACUUM INTO`)
- Retry con backoff en escrituras que reciben `SQLITE_BUSY` (ya hay `_busy_timeout=5000`, añadir capa de reintento en Go)

**Watchdog del scheduler**
- Goroutine que verifica cada minuto que los monitores activos tienen su goroutine correspondiente
- Si detecta una goroutine muerta (panic recuperado), la reinicia automáticamente y emite un log + métrica

---

### Fase 8 — Tipos de Monitor Adicionales
> *Objetivo: cubrir más capas del stack más allá de HTTP GET.*

Cada tipo nuevo implementa la interfaz `CheckFn` y tiene su propio `config_json`:

| Tipo | Qué comprueba | Config principal |
|---|---|---|
| `http` (actual) | GET/POST, status code, latencia | `expected_status`, `max_latency_ms` |
| `http_keyword` | Igual que http + string en el body | `keyword`, `invert` (not-found = up) |
| `http_json` | JSONPath sobre la respuesta | `json_path`, `expected_value` |
| `tcp` | Conectividad a host:port | `host`, `port`, `send`, `expect` |
| `dns` | Resolución de nombre | `hostname`, `record_type`, `expected_value` |
| `tls` | Expiración del certificado TLS | `host`, `port`, `warn_days` (defecto 30), `critical_days` (defecto 7) |
| `icmp` | Ping (requiere privilegios CAP_NET_RAW) | `host`, `count`, `loss_threshold_pct` |

**TLS / SSL expiry** es candidato a implementarse en Fase 5 por su utilidad inmediata para equipos SRE.

**Impacto en el schema:** columna `type` ya existe como `TEXT`; añadir constraint `CHECK(type IN (...))` en nueva migración `0002_monitor_types.up.sql`.

---

### Fase 9 — Autenticación y Control de Acceso
> *Objetivo: Pulse seguro para exponer en red corporativa o como servicio compartido.*

**API keys (mínimo viable)**
- Tabla `api_keys`: hash SHA-256 de la key, nombre, permisos (`read | write`), última vez usada, expiración opcional
- Middleware en chi: valida `Authorization: Bearer <key>` o header `X-API-Key`
- CLI: `pulse-server keys create --name ci-readonly --permissions read`
- Excepciones sin auth: `/healthz`, `/readyz`, `/metrics` (protegibles opcionalmente vía flag)

**JWT para la SPA**
- `POST /api/v1/auth/login` con usuario+contraseña → access token (15 min) + refresh token (7 días, httpOnly cookie)
- `POST /api/v1/auth/refresh` — rota el access token
- Middleware que acepta Bearer JWT en la SPA
- Primera cuenta creada via variable de entorno `PULSE_ADMIN_PASSWORD` (o fichero de credenciales)

**Roles**
- `admin`: acceso total (CRUD, gestión de canales, usuarios)
- `viewer`: solo lectura (lista monitores, ve historial y stats)
- `ops`: lectura + pause/resume, sin crear/eliminar

**Rate limiting**
- Middleware `chi` con ventana deslizante por IP / API key
- Configurable via env: `PULSE_RATE_LIMIT_RPM` (defecto 600)

---

### Fase 10 — Escala y Alta Disponibilidad
> *Objetivo: Pulse como plataforma de monitorización para flotas grandes.*

**Backend PostgreSQL**
- Nueva implementación de `store.MonitorRepository` y `store.CheckRepository` sobre `pgx/v5`
- Selección vía `PULSE_DB_DRIVER=sqlite|postgres` y `PULSE_DATABASE_URL=postgres://...`
- Migraciones con `golang-migrate` (compatible con ambos drivers)
- Índices parciales en PostgreSQL para queries de stats con grandes volúmenes

**Leader election para el scheduler**
- Tabla `scheduler_lease`: instance_id, acquired_at, expires_at
- Cada instancia intenta renovar la lease cada 5s; si expira, cualquier instancia la toma
- Solo la instancia con lease activa ejecuta checks; las demás sirven API y dashboard
- Failover automático en < 10 segundos sin coordinación externa

**Particionamiento de checks**
- En PostgreSQL: tabla `checks` particionada por `started_at` (partición mensual)
- Script de mantenimiento para `DETACH PARTITION` + archivado de meses antiguos

**Event streaming**
- Interfaz interna `CheckResultListener` (ya definida en el checker)
- Implementación para publicar a NATS JetStream o Redis Streams
- Consumers externos pueden suscribirse a check results sin polling a la BD

**Multi-región**
- Concepto de `probe`: instancia de Pulse sin scheduler propio que recibe órdenes del líder
- `POST /internal/v1/probe/run` — el líder delega checks a probes remotos
- El resultado viaja de vuelta al líder via HTTP o mensaje en el bus
- Dashboard muestra de qué región viene cada check

---

## Principios que guían las decisiones

- **Un binario, cero dependencias de runtime.** Frontend, migraciones y assets embebidos. `docker run` funciona sin volúmenes en dev.
- **Observabilidad no es opcional.** Cualquier operación nueva emite log estructurado, métrica y span. No se instrumenta después.
- **Degradación elegante sobre crash.** Fallo de notificación → log + reintento, no panic. Scheduler fallido → log + reinicio automático, no pérdida de datos.
- **Tests de integración sobre mocks.** BD real en TempDir, servidor HTTP real en loopback. Mocks solo para dependencias externas (SMTP, Slack).
- **Las interfaces crecen bajo demanda.** No añadir métodos a los repositorios hasta tener dos implementaciones reales o un test que lo requiera.
