package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/kriogman/pulse/internal/domain"
	"github.com/kriogman/pulse/internal/store/sqlite"
	"github.com/kriogman/pulse/migrations"
)

// newTestDB abre una base de datos SQLite temporal con migraciones aplicadas.
// Cada test recibe una BD aislada en t.TempDir().
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("abriendo test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlite.RunMigrations(db, migrations.FS); err != nil {
		t.Fatalf("ejecutando migraciones: %v", err)
	}
	return db
}

func testMonitor(id, name string) *domain.Monitor {
	now := time.Now().Truncate(time.Second)
	return &domain.Monitor{
		ID:          id,
		Name:        name,
		Type:        domain.MonitorTypeHTTP,
		Target:      "https://example.com",
		IntervalSec: 60,
		TimeoutMs:   5000,
		Config:      map[string]any{"expected_status": 200, "max_latency_ms": float64(500)},
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ── MonitorRepository ─────────────────────────────────────────────────────

func TestMonitorRepository_Create_GetByID(t *testing.T) {
	repo := sqlite.NewMonitorRepository(newTestDB(t))
	ctx := context.Background()

	m := testMonitor("01MON0000000001", "test-create")
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.Name != m.Name {
		t.Errorf("Name: got %q, want %q", got.Name, m.Name)
	}
	if got.Target != m.Target {
		t.Errorf("Target: got %q, want %q", got.Target, m.Target)
	}
	if got.IntervalSec != m.IntervalSec {
		t.Errorf("IntervalSec: got %d, want %d", got.IntervalSec, m.IntervalSec)
	}
	if got.TimeoutMs != m.TimeoutMs {
		t.Errorf("TimeoutMs: got %d, want %d", got.TimeoutMs, m.TimeoutMs)
	}
	if !got.Enabled {
		t.Error("Enabled: got false, want true")
	}
}

func TestMonitorRepository_GetByID_NotFound(t *testing.T) {
	repo := sqlite.NewMonitorRepository(newTestDB(t))
	_, err := repo.GetByID(context.Background(), "no-existe")
	if err == nil {
		t.Fatal("esperaba error ErrMonitorNotFound, got nil")
	}
}

func TestMonitorRepository_List(t *testing.T) {
	repo := sqlite.NewMonitorRepository(newTestDB(t))
	ctx := context.Background()

	for i, name := range []string{"alpha", "beta", "gamma"} {
		id := fmt.Sprintf("01MON000000000%d", i+1)
		if err := repo.Create(ctx, testMonitor(id, name)); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	monitors, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(monitors) != 3 {
		t.Errorf("len: got %d, want 3", len(monitors))
	}
}

func TestMonitorRepository_Update(t *testing.T) {
	repo := sqlite.NewMonitorRepository(newTestDB(t))
	ctx := context.Background()

	m := testMonitor("01MON0000000004", "to-update")
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("Create: %v", err)
	}

	m.Name = "updated-name"
	m.IntervalSec = 120
	m.UpdatedAt = time.Now().Truncate(time.Second)
	if err := repo.Update(ctx, m); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(ctx, m.ID)
	if got.Name != "updated-name" {
		t.Errorf("Name post-update: got %q, want %q", got.Name, "updated-name")
	}
	if got.IntervalSec != 120 {
		t.Errorf("IntervalSec post-update: got %d, want 120", got.IntervalSec)
	}
}

func TestMonitorRepository_Delete(t *testing.T) {
	repo := sqlite.NewMonitorRepository(newTestDB(t))
	ctx := context.Background()

	m := testMonitor("01MON0000000005", "to-delete")
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Delete(ctx, m.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := repo.GetByID(ctx, m.ID)
	if err == nil {
		t.Fatal("esperaba error tras Delete, got nil")
	}
}

// ── CheckRepository ───────────────────────────────────────────────────────

func TestCheckRepository_SaveAndList(t *testing.T) {
	db := newTestDB(t)
	monitorRepo := sqlite.NewMonitorRepository(db)
	checkRepo := sqlite.NewCheckRepository(db)
	ctx := context.Background()

	m := testMonitor("01MON0000000006", "check-test")
	if err := monitorRepo.Create(ctx, m); err != nil {
		t.Fatalf("Create monitor: %v", err)
	}

	sc := 200
	check := &domain.Check{
		MonitorID:  m.ID,
		StartedAt:  time.Now(),
		DurationMs: 42,
		Status:     domain.CheckStatusUp,
		StatusCode: &sc,
	}
	if err := checkRepo.Save(ctx, check); err != nil {
		t.Fatalf("Save: %v", err)
	}

	checks, err := checkRepo.ListByMonitor(ctx, m.ID,
		time.Now().Add(-time.Minute), time.Now().Add(time.Minute), 10)
	if err != nil {
		t.Fatalf("ListByMonitor: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("len: got %d, want 1", len(checks))
	}
	if checks[0].DurationMs != 42 {
		t.Errorf("DurationMs: got %d, want 42", checks[0].DurationMs)
	}
}

func TestCheckRepository_DeleteOlderThan(t *testing.T) {
	db := newTestDB(t)
	monitorRepo := sqlite.NewMonitorRepository(db)
	checkRepo := sqlite.NewCheckRepository(db)
	ctx := context.Background()

	m := testMonitor("01MON0000000007", "retention-test")
	if err := monitorRepo.Create(ctx, m); err != nil {
		t.Fatalf("Create monitor: %v", err)
	}

	// Guardar un check antiguo y uno reciente
	old := &domain.Check{
		MonitorID:  m.ID,
		StartedAt:  time.Now().Add(-48 * time.Hour),
		DurationMs: 10,
		Status:     domain.CheckStatusUp,
	}
	recent := &domain.Check{
		MonitorID:  m.ID,
		StartedAt:  time.Now(),
		DurationMs: 10,
		Status:     domain.CheckStatusUp,
	}
	for _, c := range []*domain.Check{old, recent} {
		if err := checkRepo.Save(ctx, c); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	n, err := checkRepo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteOlderThan: %v", err)
	}
	if n != 1 {
		t.Errorf("deleted: got %d, want 1", n)
	}

	remaining, _ := checkRepo.ListByMonitor(ctx, m.ID,
		time.Now().Add(-72*time.Hour), time.Now().Add(time.Hour), 10)
	if len(remaining) != 1 {
		t.Errorf("remaining: got %d, want 1", len(remaining))
	}
}
