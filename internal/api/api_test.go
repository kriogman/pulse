package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/kriogman/pulse/internal/api"
	"github.com/kriogman/pulse/internal/domain"
	sqlitestore "github.com/kriogman/pulse/internal/store/sqlite"
	"github.com/kriogman/pulse/migrations"
)

// ── Test helpers ──────────────────────────────────────────────────────────

type mockReloader struct{ calls int }

func (m *mockReloader) Reload(context.Context) error {
	m.calls++
	return nil
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlitestore.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlitestore.RunMigrations(db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	return db
}

func newTestRouter(t *testing.T) (http.Handler, sqlitestore.Repos, *mockReloader) {
	t.Helper()
	db := newTestDB(t)
	reloader := &mockReloader{}

	r := chi.NewRouter()
	h := humachi.New(r, huma.DefaultConfig("Test API", "0.0.0"))

	repos := sqlitestore.Repos{
		Monitors: sqlitestore.NewMonitorRepository(db),
		Checks:   sqlitestore.NewCheckRepository(db),
	}
	api.RegisterRoutes(h, repos.Monitors, repos.Checks, reloader)
	return r, repos, reloader
}

// do realiza una petición HTTP al router y devuelve el recorder.
func do(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func mustDecode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(w.Body).Decode(&v); err != nil {
		t.Fatalf("decodificando respuesta: %v\nbody: %s", err, w.Body.String())
	}
	return v
}

const monitorJSON = `{
	"name":"api-test",
	"type":"http",
	"target":"https://example.com",
	"interval_sec":60,
	"timeout_ms":5000,
	"config":{"expected_status":200}
}`

// ── Monitor CRUD ──────────────────────────────────────────────────────────

func TestListMonitors_Empty(t *testing.T) {
	router, _, _ := newTestRouter(t)
	w := do(t, router, http.MethodGet, "/api/v1/monitors", "")
	if w.Code != http.StatusOK {
		t.Fatalf("esperaba 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data  []any `json:"data"`
		Total int   `json:"total"`
	}
	mustDecode[struct {
		Data  []any `json:"data"`
		Total int   `json:"total"`
	}](t, w)
	_ = resp
}

func TestCreateMonitor_Success(t *testing.T) {
	router, _, reloader := newTestRouter(t)
	w := do(t, router, http.MethodPost, "/api/v1/monitors", monitorJSON)

	if w.Code != http.StatusCreated {
		t.Fatalf("esperaba 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.MonitorOutput
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.ID == "" {
		t.Error("ID vacío en respuesta")
	}
	if resp.Name != "api-test" {
		t.Errorf("Name: got %q, want %q", resp.Name, "api-test")
	}
	if !resp.Enabled {
		t.Error("Enabled: esperaba true por defecto")
	}
	if reloader.calls != 1 {
		t.Errorf("Reload calls: got %d, want 1", reloader.calls)
	}
}

func TestCreateMonitor_ValidationError(t *testing.T) {
	router, _, _ := newTestRouter(t)

	// interval_sec < 5 → error de validación
	body := `{"name":"x","type":"http","target":"https://example.com","interval_sec":1,"timeout_ms":5000,"config":{}}`
	w := do(t, router, http.MethodPost, "/api/v1/monitors", body)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperaba 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetMonitor_Success(t *testing.T) {
	router, repos, _ := newTestRouter(t)

	// Crear un monitor directamente en la BD para el test.
	now := time.Now()
	m := &domain.Monitor{
		ID: "01TEST0000000001", Name: "direct", Type: domain.MonitorTypeHTTP,
		Target: "https://example.com", IntervalSec: 60, TimeoutMs: 5000,
		Config: map[string]any{}, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Monitors.Create(context.Background(), m); err != nil {
		t.Fatal(err)
	}

	w := do(t, router, http.MethodGet, "/api/v1/monitors/01TEST0000000001", "")
	if w.Code != http.StatusOK {
		t.Fatalf("esperaba 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.MonitorOutput
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Name != "direct" {
		t.Errorf("Name: got %q, want %q", resp.Name, "direct")
	}
}

func TestGetMonitor_NotFound(t *testing.T) {
	router, _, _ := newTestRouter(t)
	w := do(t, router, http.MethodGet, "/api/v1/monitors/no-existe", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("esperaba 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateMonitor_Success(t *testing.T) {
	router, _, reloader := newTestRouter(t)

	// Crear
	w := do(t, router, http.MethodPost, "/api/v1/monitors", monitorJSON)
	var created api.MonitorOutput
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	// Actualizar
	updated := `{"name":"api-test-updated","type":"http","target":"https://example.com","interval_sec":120,"timeout_ms":5000,"config":{}}`
	w = do(t, router, http.MethodPut, "/api/v1/monitors/"+created.ID, updated)
	if w.Code != http.StatusOK {
		t.Fatalf("esperaba 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.MonitorOutput
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Name != "api-test-updated" {
		t.Errorf("Name: got %q, want %q", resp.Name, "api-test-updated")
	}
	if resp.IntervalSec != 120 {
		t.Errorf("IntervalSec: got %d, want 120", resp.IntervalSec)
	}
	if resp.CreatedAt != created.CreatedAt {
		t.Error("CreatedAt no debe cambiar al actualizar")
	}
	if reloader.calls != 2 { // 1 create + 1 update
		t.Errorf("Reload calls: got %d, want 2", reloader.calls)
	}
}

func TestDeleteMonitor_Success(t *testing.T) {
	router, _, reloader := newTestRouter(t)

	w := do(t, router, http.MethodPost, "/api/v1/monitors", monitorJSON)
	var created api.MonitorOutput
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	w = do(t, router, http.MethodDelete, "/api/v1/monitors/"+created.ID, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("esperaba 204, got %d: %s", w.Code, w.Body.String())
	}

	// El monitor ya no existe.
	w = do(t, router, http.MethodGet, "/api/v1/monitors/"+created.ID, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("esperaba 404 tras borrar, got %d", w.Code)
	}

	if reloader.calls != 2 { // create + delete
		t.Errorf("Reload calls: got %d, want 2", reloader.calls)
	}
}

func TestPauseAndResumeMonitor(t *testing.T) {
	router, _, _ := newTestRouter(t)

	w := do(t, router, http.MethodPost, "/api/v1/monitors", monitorJSON)
	var created api.MonitorOutput
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if !created.Enabled {
		t.Fatal("monitor debe estar activo al crearse")
	}

	// Pausar
	w = do(t, router, http.MethodPost, "/api/v1/monitors/"+created.ID+"/pause", "")
	if w.Code != http.StatusOK {
		t.Fatalf("pause: esperaba 200, got %d: %s", w.Code, w.Body.String())
	}
	var paused api.MonitorOutput
	if err := json.NewDecoder(w.Body).Decode(&paused); err != nil {
		t.Fatal(err)
	}
	if paused.Enabled {
		t.Error("pause: Enabled debe ser false")
	}

	// Reactivar
	w = do(t, router, http.MethodPost, "/api/v1/monitors/"+created.ID+"/resume", "")
	if w.Code != http.StatusOK {
		t.Fatalf("resume: esperaba 200, got %d: %s", w.Code, w.Body.String())
	}
	var resumed api.MonitorOutput
	if err := json.NewDecoder(w.Body).Decode(&resumed); err != nil {
		t.Fatal(err)
	}
	if !resumed.Enabled {
		t.Error("resume: Enabled debe ser true")
	}
}

// ── Checks ────────────────────────────────────────────────────────────────

func TestListChecks_Success(t *testing.T) {
	router, repos, _ := newTestRouter(t)
	ctx := context.Background()

	// Crear monitor
	now := time.Now()
	m := &domain.Monitor{
		ID: "01TEST0000000002", Name: "checks-test", Type: domain.MonitorTypeHTTP,
		Target: "https://example.com", IntervalSec: 60, TimeoutMs: 5000,
		Config: map[string]any{}, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Monitors.Create(ctx, m); err != nil {
		t.Fatal(err)
	}

	// Guardar un check
	sc := 200
	if err := repos.Checks.Save(ctx, &domain.Check{
		MonitorID: m.ID, StartedAt: time.Now(), DurationMs: 42,
		Status: domain.CheckStatusUp, StatusCode: &sc,
	}); err != nil {
		t.Fatal(err)
	}

	w := do(t, router, http.MethodGet, "/api/v1/monitors/01TEST0000000002/checks", "")
	if w.Code != http.StatusOK {
		t.Fatalf("esperaba 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data  []api.CheckOutput `json:"data"`
		Total int               `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 {
		t.Errorf("Total: got %d, want 1", resp.Total)
	}
	if resp.Data[0].DurationMs != 42 {
		t.Errorf("DurationMs: got %d, want 42", resp.Data[0].DurationMs)
	}
}
