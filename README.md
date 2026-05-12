# Pulse

[![CI](https://github.com/kriogman/pulse/actions/workflows/ci.yml/badge.svg)](https://github.com/kriogman/pulse/actions/workflows/ci.yml)
[![Go 1.25+](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/dl/)

On-premise HTTP endpoint monitoring daemon. Web dashboard, REST API and Prometheus metrics in a **single binary** that embeds the React frontend, SQL migrations and all static assets — no runtime dependencies.

## Features

- **Web dashboard** — monitor list, full CRUD, pause/resume, check history
- **Latency charts** — response times colored by status (up/down/degraded), period selector 1h / 24h / 7d / 30d
- **Uptime statistics** — uptime %, avg/max response time, counters per status
- **REST API** with automatic OpenAPI 3.1 (`/openapi.json`)
- **Prometheus metrics** at `/metrics` — latency histogram, total checks, active monitors
- **Persistent scheduler** — one goroutine per monitor, reload without restart, checks run immediately on startup
- **SQLite in WAL mode** — no external database server, persisted in a single file

---

## Deployment

### Docker

```bash
docker run -d \
  --name pulse \
  -p 8080:8080 \
  -v pulse-data:/data \
  -e PULSE_DB_PATH=/data/pulse.db \
  ghcr.io/kriogman/pulse:latest
```

Open [http://localhost:8080](http://localhost:8080).

The `pulse-data` volume persists the database across restarts. Without it, data is lost when the container stops.

---

### Docker Compose

```bash
git clone https://github.com/kriogman/pulse.git
cd pulse
docker-compose up -d
```

The included `docker-compose.yml` configures the server, a persistent volume and `restart: unless-stopped`.

```bash
docker-compose logs -f    # stream logs
docker-compose down       # stop, keep data
docker-compose down -v    # stop and delete volume
```

---

### Kubernetes

The manifests below deploy Pulse in a dedicated namespace with a `PersistentVolumeClaim` for the database.

#### 1. Namespace and PersistentVolumeClaim

```yaml
# pulse-namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: pulse
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pulse-data
  namespace: pulse
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
```

#### 2. Deployment

```yaml
# pulse-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pulse
  namespace: pulse
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pulse
  template:
    metadata:
      labels:
        app: pulse
    spec:
      containers:
        - name: pulse
          image: ghcr.io/kriogman/pulse:latest
          ports:
            - containerPort: 8080
          env:
            - name: PULSE_DB_PATH
              value: /data/pulse.db
            - name: PULSE_LISTEN_ADDR
              value: ":8080"
            - name: PULSE_LOG_LEVEL
              value: info
            - name: PULSE_LOG_FORMAT
              value: json
            - name: PULSE_CHECKS_RETENTION_DAYS
              value: "90"
          volumeMounts:
            - name: data
              mountPath: /data
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 15
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 500m
              memory: 256Mi
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: pulse-data
```

> **Note:** SQLite requires `ReadWriteOnce`. Do not scale `replicas` above 1 with SQLite — if you need multiple replicas, see the [Roadmap](./ROADMAP.md) (Phase 10 — PostgreSQL + leader election).

#### 3. Service

```yaml
# pulse-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: pulse
  namespace: pulse
spec:
  selector:
    app: pulse
  ports:
    - port: 80
      targetPort: 8080
```

#### 4. Ingress (optional)

```yaml
# pulse-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: pulse
  namespace: pulse
spec:
  rules:
    - host: pulse.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: pulse
                port:
                  number: 80
```

Replace `pulse.example.com` with your domain. If using cert-manager, add the `cert-manager.io/cluster-issuer` annotation and a `tls` block.

#### Apply

```bash
kubectl apply -f pulse-namespace.yaml
kubectl apply -f pulse-deployment.yaml
kubectl apply -f pulse-service.yaml
kubectl apply -f pulse-ingress.yaml   # optional

kubectl -n pulse rollout status deployment/pulse
```

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `PULSE_DB_PATH` | `./pulse.db` | Path to the SQLite file |
| `PULSE_LISTEN_ADDR` | `:8080` | Listen address |
| `PULSE_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `PULSE_LOG_FORMAT` | `json` | `json` / `text` |
| `PULSE_CHECKS_RETENTION_DAYS` | `90` | Days of check history to retain |

---

## REST API

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | React dashboard |
| `GET` | `/api/v1/monitors` | List monitors (paginated) |
| `POST` | `/api/v1/monitors` | Create a monitor |
| `GET` | `/api/v1/monitors/{id}` | Get a monitor |
| `PUT` | `/api/v1/monitors/{id}` | Update a monitor |
| `DELETE` | `/api/v1/monitors/{id}` | Delete monitor and history |
| `POST` | `/api/v1/monitors/{id}/pause` | Pause a monitor |
| `POST` | `/api/v1/monitors/{id}/resume` | Resume a monitor |
| `GET` | `/api/v1/monitors/{id}/checks` | Check history |
| `GET` | `/api/v1/monitors/{id}/stats` | Uptime and latency statistics |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe (verifies DB) |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/openapi.json` | OpenAPI 3.1 spec |

---

## Development

```bash
# Terminal 1 — Go server with hot-reload
make dev-server          # listens on :8080, text logs

# Terminal 2 — frontend with hot-reload
make web-dev             # Vite on :5173, proxies /api → :8080

# Tests and lint
make test                # go test -v -race ./...
make lint                # golangci-lint

# Build server for all platforms
make build-all
```

### Project structure

```
cmd/
  pulse-server/     → Daemon: HTTP + scheduler + metrics
internal/
  domain/           → Pure entities (Monitor, Check, CheckStats)
  store/            → Repository interfaces
  store/sqlite/     → SQLite implementation (pure-Go, no CGO)
  api/              → huma v2 handlers, DTOs, OpenAPI
  scheduler/        → Per-monitor goroutine, incremental reload
  checker/          → Pure HTTP check logic, injectable
  observability/    → slog, Prometheus, OTel
migrations/         → Embedded SQL with go:embed
web/                → React+TypeScript+Vite frontend (embedded in binary)
```

---

## Roadmap

See [ROADMAP.md](./ROADMAP.md) for the plan of future phases: alerts and notifications, distributed traces, additional monitor types (TCP, DNS, TLS), authentication and high availability.
