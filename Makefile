BINARY        := pulse
SERVER_BINARY := pulse-server
DIST          := dist
MODULE        := github.com/kriogman/pulse
PULSE_DB_PATH ?= ./pulse.db

# ── CLI ──────────────────────────────────────────────────────────────────────

build:
	go build -o $(BINARY) ./cmd/pulse/

run:
	go run ./cmd/pulse/ check

# ── Server ───────────────────────────────────────────────────────────────────

build-server: web-build
	CGO_ENABLED=0 go build -o $(SERVER_BINARY) ./cmd/pulse-server/

dev-server:
	PULSE_LOG_FORMAT=text PULSE_LOG_LEVEL=debug PULSE_DB_PATH=$(PULSE_DB_PATH) \
	go run ./cmd/pulse-server/

# ── Web (frontend) ────────────────────────────────────────────────────────────

web-install:
	cd web && npm install

web-build:
	cd web && npm run build

web-dev:
	cd web && npm run dev

# ── Tests ────────────────────────────────────────────────────────────────────

test:
	go test -v -race ./...

# ── Code generation ──────────────────────────────────────────────────────────

# Requiere sqlc instalado: https://docs.sqlc.dev/en/latest/overview/install.html
gen-sqlc:
	sqlc generate

# Fase 3: genera tipos TS desde el spec OpenAPI.
# gen-api:
#	npx openapi-typescript api/openapi.yaml -o web/src/api/types.ts

# ── Cross-compilation ─────────────────────────────────────────────────────────

build-all: $(DIST)
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o $(DIST)/$(BINARY)-linux-amd64       ./cmd/pulse/
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o $(DIST)/$(BINARY)-darwin-arm64      ./cmd/pulse/
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(DIST)/$(BINARY)-windows-amd64.exe ./cmd/pulse/
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o $(DIST)/$(SERVER_BINARY)-linux-amd64       ./cmd/pulse-server/
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o $(DIST)/$(SERVER_BINARY)-darwin-arm64      ./cmd/pulse-server/
	@ls -lh $(DIST)/

$(DIST):
	mkdir -p $(DIST)

# ── Lint ─────────────────────────────────────────────────────────────────────

lint:
	golangci-lint run ./...

# ── Cleanup ───────────────────────────────────────────────────────────────────

clean:
	rm -f $(BINARY) $(SERVER_BINARY)
	rm -rf $(DIST) web/dist

.PHONY: build run build-server dev-server web-install web-build web-dev test gen-sqlc build-all lint clean
