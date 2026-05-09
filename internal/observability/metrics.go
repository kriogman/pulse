package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Metrics agrupa el registro Prometheus y las métricas del servidor.
type Metrics struct {
	Registry *prometheus.Registry

	// Scheduler
	MonitorsActive prometheus.Gauge
	ChecksInFlight prometheus.Gauge
	ChecksTotal    *prometheus.CounterVec
	CheckDuration  *prometheus.HistogramVec

	// HTTP RED (Fase 2)
	// HTTPRequestsTotal   *prometheus.CounterVec
	// HTTPRequestDuration *prometheus.HistogramVec
}

// NewMetrics crea el registro Prometheus con todos los colectores necesarios.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	m := &Metrics{
		Registry: reg,

		MonitorsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pulse_monitors_active",
			Help: "Número de monitores con goroutine activa en el scheduler.",
		}),
		ChecksInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pulse_checks_in_flight",
			Help: "Número de health checks ejecutándose en este momento.",
		}),
		ChecksTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pulse_checks_total",
			Help: "Total de health checks ejecutados.",
		}, []string{"monitor_id", "status"}),
		CheckDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "pulse_check_duration_seconds",
			Help:    "Duración de health checks en segundos.",
			Buckets: prometheus.DefBuckets,
		}, []string{"monitor_id"}),
	}

	reg.MustRegister(
		m.MonitorsActive,
		m.ChecksInFlight,
		m.ChecksTotal,
		m.CheckDuration,
	)

	return m
}
