# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Comandos de desarrollo

```bash
make build          # Compila ./pulse para la plataforma actual
make test           # go test -v -race ./... (todos los paquetes)
make run            # go run ./cmd/pulse/ check (sin compilar)
make build-all      # Cross-compila a linux/amd64, darwin/arm64, windows/amd64

# Test de un solo caso
go test -v -run TestNombreDelTest ./internal/checker/
go test -v -run TestNombreDelTest ./internal/config/

# Primera vez o tras cambiar dependencias
go mod tidy
```

Go debe estar instalado ([go.dev/dl](https://go.dev/dl/), v1.21+). Tras clonar, ejecutar `go mod tidy` para descargar dependencias.

## Arquitectura

CLI de un único subcomando (`check`) con tres capas:

```
cmd/pulse/main.go          → Wiring: Cobra, flags, orquesta config + checker
internal/config/config.go  → Lee pulse.yaml → struct Config{Targets []Target}
internal/checker/checker.go → HTTP GET en paralelo, evalúa status+latencia, imprime
```

**Flujo de ejecución:**
1. `main.go` parsea flags y llama `config.Load(path)`
2. Llama `checker.RunAll(targets, timeout)` → lanza una goroutine por target
3. Cada goroutine ejecuta `checkOne()` y envía `Result` a un canal con buffer
4. `RunAll` recoge resultados del canal hasta que se cierra (WaitGroup cierra el canal)
5. `checker.PrintResults(results, format)` imprime y devuelve `allOK bool`
6. Si `!allOK` → `os.Exit(1)` (exit code para CI)

## Concurrencia

`RunAll` usa **WaitGroup + canal con buffer**: el canal comunica resultados entre goroutines (patrón idiomático Go), el WaitGroup controla cuándo cerrar el canal. Ver comentarios extensos en `checker.go` sobre por qué no se usa mutex.

Trampa del closure en loops: cada goroutine recibe `target` como **argumento** (copia por valor), no por captura de variable del loop.

## Módulo Go

Módulo: `github.com/your-username/pulse` (placeholder — cambiar al hacer fork).
Dependencias: `spf13/cobra` (CLI), `gopkg.in/yaml.v3` (YAML).
Compilación estática por defecto: el binario no requiere librerías del SO en destino.
Cross-compile: `GOOS=linux GOARCH=amd64 go build -o dist/pulse-linux-amd64 ./cmd/pulse/`

## Tests

Los tests de `checker` usan `httptest.NewServer` para levantar servidores HTTP reales en loopback (no mocks de interfaz). Los tests de `config` usan ficheros temporales con `os.CreateTemp`.

El `-race` flag en `make test` activa el detector de race conditions de Go.
