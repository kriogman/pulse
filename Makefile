# Makefile para pulse
# Ejecutar `make` sin argumentos compila el binario (primer target = default).

BINARY  := pulse
DIST    := dist
MODULE  := github.com/your-username/pulse

# build: compila para la plataforma actual.
# `go build -o <destino> <paquete>` produce un binario estático por defecto en Go.
# No necesitas Dockerfile ni toolchains extra para compilar en tu máquina.
build:
	go build -o $(BINARY) ./cmd/pulse/

# test: ejecuta todos los tests del módulo con output detallado.
# `./...` significa "este directorio y todos sus subdirectorios recursivamente".
# Para ejecutar un solo test: go test -v -run TestNombreDelTest ./internal/checker/
test:
	go test -v -race ./...

# run: ejecuta directamente sin producir binario (útil en desarrollo).
run:
	go run ./cmd/pulse/ check

# build-all: cross-compila para las tres plataformas principales.
#
# Go incluye soporte de cross-compilación de serie — no necesitas toolchains externos.
# GOOS  = sistema operativo destino (linux, darwin, windows)
# GOARCH = arquitectura destino (amd64, arm64, 386…)
# El runtime de Go se compila dentro del binario, por eso el ejecutable
# no necesita librerías externas: es 100% estático.
build-all: $(DIST)
	GOOS=linux   GOARCH=amd64  go build -o $(DIST)/$(BINARY)-linux-amd64       ./cmd/pulse/
	GOOS=darwin  GOARCH=arm64  go build -o $(DIST)/$(BINARY)-darwin-arm64      ./cmd/pulse/
	GOOS=windows GOARCH=amd64  go build -o $(DIST)/$(BINARY)-windows-amd64.exe ./cmd/pulse/
	@echo "Binarios generados en $(DIST)/:"
	@ls -lh $(DIST)/

$(DIST):
	mkdir -p $(DIST)

# clean: borra binarios generados.
clean:
	rm -f $(BINARY)
	rm -rf $(DIST)

# .PHONY indica que estos targets no son ficheros: make siempre los ejecuta
# aunque exista un fichero con ese nombre (ej: si alguien crea un fichero "build").
.PHONY: build test run build-all clean
