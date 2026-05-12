# Fase de compilación
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Descargar dependencias primero para aprovechar la caché de capas
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0: binario estático puro Go (modernc.org/sqlite no necesita CGO)
# -ldflags="-s -w": elimina debug info y tabla de símbolos (~30% más pequeño)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /pulse-server \
    ./cmd/pulse-server/

# Crear /data con el UID del usuario nonroot (65532) para que SQLite pueda escribir
RUN mkdir /data && chown 65532:65532 /data

# Imagen final mínima: solo el binario estático, sin shell ni herramientas
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder --chown=65532:65532 /pulse-server /pulse-server
COPY --from=builder --chown=65532:65532 /data /data

EXPOSE 8080

ENTRYPOINT ["/pulse-server"]
