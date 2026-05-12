package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kriogman/pulse/internal/domain"
)

func monitorFor(t *testing.T, url string, expectedStatus int, maxLatencyMs int64, timeoutMs int) *domain.Monitor {
	t.Helper()
	return &domain.Monitor{
		ID:        "test",
		Name:      "test",
		Type:      domain.MonitorTypeHTTP,
		Target:    url,
		TimeoutMs: timeoutMs,
		Config: map[string]any{
			"expected_status": expectedStatus,
			"max_latency_ms":  maxLatencyMs,
		},
	}
}

func TestCheckMonitor_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := CheckMonitor(context.Background(), monitorFor(t, srv.URL, 200, 5000, 5000))

	if c.Status != domain.CheckStatusUp {
		t.Errorf("esperaba Up, got %s: %v", c.Status, c.Error)
	}
	if c.StatusCode == nil || *c.StatusCode != 200 {
		t.Errorf("esperaba StatusCode 200, got %v", c.StatusCode)
	}
}

func TestCheckMonitor_WrongStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := CheckMonitor(context.Background(), monitorFor(t, srv.URL, 200, 5000, 5000))

	if c.Status != domain.CheckStatusDown {
		t.Error("esperaba Down por status incorrecto")
	}
	if c.StatusCode == nil || *c.StatusCode != 500 {
		t.Errorf("esperaba StatusCode 500, got %v", c.StatusCode)
	}
	if c.Error == nil || *c.Error == "" {
		t.Error("esperaba Error no vacío")
	}
}

func TestCheckMonitor_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := CheckMonitor(context.Background(), monitorFor(t, srv.URL, 200, 5000, 500))

	if c.Status != domain.CheckStatusDown {
		t.Error("esperaba Down por timeout")
	}
	if c.StatusCode != nil {
		t.Errorf("esperaba StatusCode nil (sin respuesta), got %v", c.StatusCode)
	}
}

func TestCheckMonitor_LatencyExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := CheckMonitor(context.Background(), monitorFor(t, srv.URL, 200, 50, 5000))

	if c.Status != domain.CheckStatusDown {
		t.Error("esperaba Down por latencia excedida")
	}
	if c.StatusCode == nil || *c.StatusCode != 200 {
		t.Errorf("esperaba StatusCode 200 (respondió bien, pero tarde), got %v", c.StatusCode)
	}
}

func TestCheckMonitor_ZeroMaxLatency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := CheckMonitor(context.Background(), monitorFor(t, srv.URL, 200, 0, 5000))

	if c.Status != domain.CheckStatusUp {
		t.Errorf("esperaba Up cuando MaxLatencyMs=0, got Down: %v", c.Error)
	}
}
