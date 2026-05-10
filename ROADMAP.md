# Pulse — Roadmap

Monitor de endpoints HTTP con dashboard web, scheduler persistente y métricas Prometheus.

---

## Estado actual

Pulse es un daemon Go con frontend React embebido. Un único binario arranca el servidor HTTP, el scheduler de checks y sirve la SPA sin dependencias externas.

**Stack:**
- **Backend:** Go 1.22+, huma v2 (OpenAPI 3.1), chi, SQLite WAL (pure-Go), Prometheus, OTel
- **Frontend:** React 18, TypeScript strict, Vite 5, Tailwind CSS v4, TanStack Query, Recharts
- **Ops:** Docker (distroless), docker-compose, GitHub Actions CI

---

## Fases completadas

### Fase 0 — Bootstrap ✅
Scaffolding multi-binario, módulo Go, Makefile, CI (lint + test -race + govulncheck + docker build), Dockerfile distroless, docker-compose.

### Fase 1 — Scheduler + Persistencia ✅
- Dominio hexagonal: `domain/`, `store/` interfaces, `store/sqlite/` implementación
- Migrations embebidas con runner propio (sin CGO)
- Scheduler: goroutine por monitor, reload incremental, `CheckFn` inyectable
- CLI ampliado: `import` (upsert desde YAML) y `list`
- Métricas Prometheus: monitores activos, checks in-flight, duración (histograma), total por status

### Fase 2 — API REST ✅
- 9 endpoints bajo `/api/v1/`: CRUD monitores, pause/resume, historial de checks, stats
- OpenAPI 3.1 auto-generado por huma (`api/openapi.json`)
- Errores RFC 7807, validación en entrada, `Reloader` interface para evitar dependencia circular
- Tests de integración con BD real y router completo

### Fase 3 — Dashboard Web ✅
- SPA React+TypeScript embebida en el binario (embed.FS)
- Dashboard: tabla de monitores con CRUD, pause/resume, confirmación de borrado
- Historial de checks por monitor
- Hot-reload en dev vía proxy Vite → API Go

### Fase 4 — Gráficas y Stats ✅
- Endpoint `/api/v1/monitors/{id}/stats?period=1h|24h|7d|30d`
- Selector de período en la página de historial
- Cards de estadísticas: uptime %, total checks, respuesta media/máxima
- Gráfica de tiempo de respuesta (Recharts): puntos coloreados por estado, línea de media
- Columna "Uptime 24h" en el dashboard (carga paralela con `useQueries`)

---

## Fases futuras

### Fase 5 — Alertas y Notificaciones
> *Objetivo: saber cuándo algo falla, sin abrir el dashboard.*

- Modelo de alerta: umbral de fallos consecutivos, uptime % por debajo de X, latencia > threshold
- Máquina de estados: `inactive → pending → firing → resolved`
- Canales de notificación: webhook genérico, Slack, email (SMTP)
- Tabla `alerts` y `notification_channels` en SQLite
- API para gestionar canales y reglas desde el dashboard
- Cooldown configurable para evitar spam
- UI: sección "Alertas activas" en el dashboard

### Fase 6 — Observabilidad Profunda
> *Objetivo: Pulse observable por sí mismo, integrable con el stack SRE.*

- **Trazas distribuidas:** activar el exportador OTLP (ya cableado como noop); cada check execution como span con atributos monitor_id, status, duration
- **Logs estructurados mejorados:** correlation ID en cada request HTTP, trace_id propagado al logger, formato compatible con Loki
- **Métricas ampliadas:** etiquetas `env`, `region`, `monitor_name`; histograma de latencia con buckets ajustados; gauge de error budget
- **SLO tracking:** definir SLO por monitor (ej: 99.9% uptime / 30d), calcular error budget restante y burn rate
- **Dashboard de Grafana:** provisioning automático de dashboard + datasources vía JSON en `/grafana/`

### Fase 7 — Resiliencia del Scheduler
> *Objetivo: el scheduler sobrevive a reinicios, picos y fallos transitorios.*

- **Retry con backoff exponencial:** fallos de red transitorios no cuentan como "down" inmediatamente (configurable: intentos, ventana)
- **Circuit breaker por monitor:** si un monitor falla N veces seguidas, entra en open state y reduce la frecuencia de checks para no saturar targets degradados
- **Jitter en el intervalo:** distribuir checks aleatoriamente ±10% del intervalo para evitar thundering herd al arrancar
- **Graceful reload:** el reload del scheduler aplica cambios de configuración sin interrumpir checks en vuelo
- **WAL checkpoint management:** tarea periódica de `PRAGMA wal_checkpoint(TRUNCATE)` para controlar el tamaño del WAL

### Fase 8 — Tipos de Monitor Adicionales
> *Objetivo: cubrir más capas del stack más allá de HTTP.*

- **TCP:** conectividad a host:port, tiempo de establecimiento de conexión
- **DNS:** resolución de nombre, valor esperado, tiempo de respuesta del resolver
- **TLS/SSL:** días hasta expiración del certificado, alerta configurable (ej: <30 días), cadena de confianza
- **Ping (ICMP):** latencia y pérdida de paquetes (requiere privilegios o raw sockets)
- **Keyword match:** buscar/no buscar string en el body de la respuesta HTTP
- **JSON path:** evaluar expresión JSONPath sobre la respuesta (ej: `.status == "ok"`)

### Fase 9 — Autenticación y Multiusuario
> *Objetivo: Pulse como servicio compartido en un equipo.*

- API key simple para acceso programático (header `X-API-Key`)
- JWT con refresh token para la SPA
- Tabla `users` con roles: admin / viewer
- Monitores con owner; admins ven todo, viewers solo los suyos
- Rate limiting por API key en el router chi

### Fase 10 — Escala y Alta Disponibilidad
> *Objetivo: soporte para despliegues multi-instancia.*

- **PostgreSQL** como backend de almacenamiento alternativo (mismas interfaces `store/`)
- **Leader election** vía tabla de lease en BD: solo una instancia ejecuta el scheduler
- **Modo read-replica:** instancias sin scheduler que solo sirven la API y el dashboard
- **Event bus interno:** check results publicados a un canal; listeners desacoplados (notificaciones, métricas)
- **Sharding de monitores:** distribuir la carga de checks entre instancias via consistent hashing

---

## Principios de diseño a mantener

- **Un binario, cero dependencias de runtime:** el servidor embebe frontend, migraciones y assets. `docker run` funciona sin volúmenes adicionales en dev.
- **Tests de integración sobre mocks:** BD real en TempDir, servidor HTTP real en loopback. Los mocks solo para dependencias externas (notificaciones, SMTP).
- **Interfaces mínimas:** añadir métodos a `MonitorRepository` / `CheckRepository` solo cuando hay al menos dos implementaciones o un test que lo requiera.
- **Observabilidad desde el diseño:** cada operación relevante emite un span, una métrica y un log. No añadir instrumentación como afterthought.
- **Degradación elegante:** si una notificación falla, el check ya fue persistido. Si el scheduler falla en recargar, usa la config anterior hasta el siguiente ciclo.
