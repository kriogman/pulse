SERVER_BINARY := pulse-server
DIST          := dist
PULSE_DB_PATH ?= ./pulse.db

# ── Server ───────────────────────────────────────────────────────────────────

build: web-build
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

# ── Lint ─────────────────────────────────────────────────────────────────────

lint:
	golangci-lint run ./...

# ── Cross-compilation ─────────────────────────────────────────────────────────

build-all: web-build $(DIST)
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o $(DIST)/$(SERVER_BINARY)-linux-amd64       ./cmd/pulse-server/
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o $(DIST)/$(SERVER_BINARY)-darwin-arm64      ./cmd/pulse-server/
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o $(DIST)/$(SERVER_BINARY)-linux-arm64       ./cmd/pulse-server/
	@ls -lh $(DIST)/

$(DIST):
	mkdir -p $(DIST)

# ── Cleanup ───────────────────────────────────────────────────────────────────

clean:
	rm -f $(SERVER_BINARY)
	rm -rf $(DIST) web/dist

.PHONY: build dev-server web-install web-build web-dev test lint build-all clean
