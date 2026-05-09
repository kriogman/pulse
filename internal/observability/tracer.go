package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
)

// SetupTracer configura un TracerProvider NOOP para V1.
// futuro: cuando OTEL_EXPORTER_OTLP_ENDPOINT esté definido, activar el exporter OTLP.
func SetupTracer() {
	otel.SetTracerProvider(noop.NewTracerProvider())
}
