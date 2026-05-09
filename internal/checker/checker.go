// Package checker contiene la lógica de ejecución de health checks HTTP.
package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/kriogman/pulse/internal/config"
	"github.com/kriogman/pulse/internal/domain"
)

// CheckResultListener es notificado tras cada ejecución de health check.
// Añade futuros listeners (email, Slack, webhooks) aquí sin tocar el checker.
type CheckResultListener interface {
	OnResult(ctx context.Context, result domain.Check)
}

// Result es el tipo de salida del CLI, mantenido por compatibilidad.
type Result struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
	LatencyMs  int64  `json:"latency_ms"`
	OK         bool   `json:"ok"`
	Reason     string `json:"reason,omitempty"`
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

// RunAll ejecuta checks para todos los targets en paralelo (uso CLI).
func RunAll(targets []config.Target, timeoutSecs int) []Result {
	resultsCh := make(chan Result, len(targets))
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		go func(t config.Target) {
			defer wg.Done()
			resultsCh <- checkOne(t, timeoutSecs)
		}(target)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var results []Result
	for r := range resultsCh {
		results = append(results, r)
	}
	return results
}

// checkOne mantiene compatibilidad con los tests y el CLI existente.
func checkOne(target config.Target, timeoutSecs int) Result {
	m := &domain.Monitor{
		ID:        target.Name,
		Name:      target.Name,
		Type:      domain.MonitorTypeHTTP,
		Target:    target.URL,
		TimeoutMs: timeoutSecs * 1000,
		Config: map[string]any{
			"expected_status": target.ExpectedStatus,
			"max_latency_ms":  target.MaxLatencyMs,
		},
	}
	c := CheckMonitor(context.Background(), m)
	return checkToResult(c, target.Name, target.URL)
}

func checkToResult(c domain.Check, name, url string) Result {
	r := Result{
		Name:      name,
		URL:       url,
		LatencyMs: c.DurationMs,
		OK:        c.Status == domain.CheckStatusUp,
	}
	if c.StatusCode != nil {
		r.StatusCode = *c.StatusCode
	}
	if c.Error != nil {
		r.Reason = *c.Error
	}
	return r
}

// PrintResults muestra los resultados y devuelve true si todos pasaron (uso CLI).
func PrintResults(results []Result, format string) bool {
	allOK := true
	for _, r := range results {
		if !r.OK {
			allOK = false
		}
	}

	switch format {
	case "json":
		printJSON(results)
	default:
		printText(results)
	}

	return allOK
}

func printText(results []Result) {
	for _, r := range results {
		verdict := "OK  "
		if !r.OK {
			verdict = "FAIL"
		}
		if r.StatusCode == 0 {
			fmt.Printf("[%s] %-20s %s — %dms — %s\n",
				verdict, r.Name, r.URL, r.LatencyMs, r.Reason)
		} else {
			line := fmt.Sprintf("[%s] %-20s %s — HTTP %d — %dms",
				verdict, r.Name, r.URL, r.StatusCode, r.LatencyMs)
			if r.Reason != "" {
				line += " — " + r.Reason
			}
			fmt.Println(line)
		}
	}
}

func printJSON(results []Result) {
	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Printf(`[{"error": "%v"}]`+"\n", err)
		return
	}
	fmt.Println(string(output))
}
