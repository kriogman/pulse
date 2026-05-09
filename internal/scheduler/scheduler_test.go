package scheduler_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kriogman/pulse/internal/domain"
	"github.com/kriogman/pulse/internal/observability"
	"github.com/kriogman/pulse/internal/scheduler"
	"github.com/kriogman/pulse/internal/store/sqlite"
	"github.com/kriogman/pulse/migrations"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("abriendo test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlite.RunMigrations(db, migrations.FS); err != nil {
		t.Fatalf("migraciones: %v", err)
	}
	return db
}

func testMonitor(id, name string) *domain.Monitor {
	now := time.Now()
	return &domain.Monitor{
		ID: id, Name: name,
		Type: domain.MonitorTypeHTTP, Target: "https://example.com",
		IntervalSec: 5, TimeoutMs: 5000,
		Config:    map[string]any{"expected_status": 200},
		Enabled:   true,
		CreatedAt: now, UpdatedAt: now,
	}
}

// TestScheduler_ImmediateCheck verifica que el scheduler ejecuta el primer check
// inmediatamente al arrancar, sin esperar el primer tick del ticker.
func TestScheduler_ImmediateCheck(t *testing.T) {
	db := newTestDB(t)
	monRepo := sqlite.NewMonitorRepository(db)
	chkRepo := sqlite.NewCheckRepository(db)
	ctx := context.Background()

	m := testMonitor("01SCHED000000001", "immediate-test")
	if err := monRepo.Create(ctx, m); err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int32
	mockFn := func(_ context.Context, _ *domain.Monitor) domain.Check {
		calls.Add(1)
		return domain.Check{
			MonitorID:  m.ID,
			StartedAt:  time.Now(),
			DurationMs: 5,
			Status:     domain.CheckStatusUp,
		}
	}

	sched := scheduler.New(monRepo, chkRepo, mockFn, observability.NewMetrics())

	cancelCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		sched.Start(cancelCtx) //nolint:errcheck
	}()

	// Esperar a que se ejecute el check inmediato (máximo 2s).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if calls.Load() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done

	if calls.Load() < 1 {
		t.Errorf("esperaba >= 1 llamada al checkFn, got %d", calls.Load())
	}

	// Verificar que el resultado se guardó en la BD.
	checks, err := chkRepo.ListByMonitor(ctx, m.ID,
		time.Now().Add(-time.Minute), time.Now().Add(time.Minute), 10)
	if err != nil {
		t.Fatalf("ListByMonitor: %v", err)
	}
	if len(checks) == 0 {
		t.Error("esperaba al menos 1 check guardado en BD")
	}
}

// TestScheduler_ReloadAddsMonitor verifica que Reload arranca goroutines para nuevos monitores.
func TestScheduler_ReloadAddsMonitor(t *testing.T) {
	db := newTestDB(t)
	monRepo := sqlite.NewMonitorRepository(db)
	chkRepo := sqlite.NewCheckRepository(db)

	var calls atomic.Int32
	mockFn := func(_ context.Context, _ *domain.Monitor) domain.Check {
		calls.Add(1)
		return domain.Check{Status: domain.CheckStatusUp}
	}

	sched := scheduler.New(monRepo, chkRepo, mockFn, observability.NewMetrics())

	cancelCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		sched.Start(cancelCtx) //nolint:errcheck
	}()

	// Inicialmente 0 monitores: ningún check.
	time.Sleep(50 * time.Millisecond)
	if sched.ActiveCount() != 0 {
		t.Errorf("ActiveCount inicial: got %d, want 0", sched.ActiveCount())
	}

	// Añadir un monitor y forzar reload.
	m := testMonitor("01SCHED000000002", "reload-test")
	if err := monRepo.Create(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	if err := sched.Reload(cancelCtx); err != nil {
		t.Fatal(err)
	}

	// Esperar al check inmediato tras el reload.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if calls.Load() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done

	if calls.Load() < 1 {
		t.Errorf("esperaba >= 1 llamada al checkFn tras reload, got %d", calls.Load())
	}
}

// TestScheduler_StopsOnDisabledMonitor verifica que Reload detiene goroutines
// cuando un monitor se desactiva.
func TestScheduler_StopsOnDisabledMonitor(t *testing.T) {
	db := newTestDB(t)
	monRepo := sqlite.NewMonitorRepository(db)
	chkRepo := sqlite.NewCheckRepository(db)

	mockFn := func(_ context.Context, _ *domain.Monitor) domain.Check {
		return domain.Check{Status: domain.CheckStatusUp}
	}

	sched := scheduler.New(monRepo, chkRepo, mockFn, observability.NewMetrics())

	m := testMonitor("01SCHED000000003", "disable-test")
	if err := monRepo.Create(context.Background(), m); err != nil {
		t.Fatal(err)
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		sched.Start(cancelCtx) //nolint:errcheck
	}()

	time.Sleep(100 * time.Millisecond)
	if sched.ActiveCount() != 1 {
		t.Fatalf("ActiveCount tras inicio: got %d, want 1", sched.ActiveCount())
	}

	// Desactivar y recargar.
	m.Enabled = false
	m.UpdatedAt = time.Now()
	if err := monRepo.Update(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	if err := sched.Reload(cancelCtx); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)

	if sched.ActiveCount() != 0 {
		t.Errorf("ActiveCount tras desactivar: got %d, want 0", sched.ActiveCount())
	}

	cancel()
	<-done
}
