// Package checker contiene la lógica de ejecución de health checks HTTP.
package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kriogman/pulse/internal/domain"
)

// CheckResultListener es notificado tras cada ejecución de health check.
// Añade futuros listeners (email, Slack, webhooks) aquí sin tocar el checker.
type CheckResultListener interface {
	OnResult(ctx context.Context, result domain.Check)
}

// CheckMonitor ejecuta un health check para el monitor dado.
// El timeout se deriva de monitor.TimeoutMs via context.WithTimeout.
func CheckMonitor(ctx context.Context, m *domain.Monitor) domain.Check {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(m.TimeoutMs)*time.Millisecond)
	defer cancel()
	return checkHTTP(ctx, m)
}

func checkHTTP(ctx context.Context, m *domain.Monitor) domain.Check {
	start := time.Now()
	cfg := httpConfigFrom(m)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.Target, nil)
	if err != nil {
		return failedCheck(m.ID, start, 0, fmt.Sprintf("creating request: %v", err))
	}

	resp, err := http.DefaultClient.Do(req)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		return failedCheck(m.ID, start, durationMs, fmt.Sprintf("connection error: %v", err))
	}
	defer resp.Body.Close()

	check := domain.Check{
		MonitorID:  m.ID,
		StartedAt:  start,
		DurationMs: durationMs,
		Status:     domain.CheckStatusUp,
		StatusCode: &resp.StatusCode,
	}

	var reasons []string

	if resp.StatusCode != cfg.ExpectedStatus {
		reasons = append(reasons,
			fmt.Sprintf("expected status %d, got %d", cfg.ExpectedStatus, resp.StatusCode))
	}
	if cfg.MaxLatencyMs > 0 && durationMs > cfg.MaxLatencyMs {
		reasons = append(reasons,
			fmt.Sprintf("latency %dms exceeded limit %dms", durationMs, cfg.MaxLatencyMs))
	}

	if len(reasons) > 0 {
		check.Status = domain.CheckStatusDown
		msg := reasons[0]
		for _, r := range reasons[1:] {
			msg += "; " + r
		}
		check.Error = &msg
	}

	return check
}

func failedCheck(monitorID string, start time.Time, durationMs int64, reason string) domain.Check {
	return domain.Check{
		MonitorID:  monitorID,
		StartedAt:  start,
		DurationMs: durationMs,
		Status:     domain.CheckStatusDown,
		Error:      &reason,
	}
}

func httpConfigFrom(m *domain.Monitor) domain.HTTPConfig {
	data, _ := json.Marshal(m.Config)
	var cfg domain.HTTPConfig
	_ = json.Unmarshal(data, &cfg)
	if cfg.ExpectedStatus == 0 {
		cfg.ExpectedStatus = http.StatusOK
	}
	return cfg
}
