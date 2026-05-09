package store

import (
	"context"
	"time"

	"github.com/kriogman/pulse/internal/domain"
)

// MonitorRepository define las operaciones de persistencia para monitores.
// futuro: añadir ListByUserID(ctx, userID) cuando se implemente multi-tenant.
type MonitorRepository interface {
	Create(ctx context.Context, m *domain.Monitor) error
	GetByID(ctx context.Context, id string) (*domain.Monitor, error)
	List(ctx context.Context) ([]*domain.Monitor, error)
	Update(ctx context.Context, m *domain.Monitor) error
	Delete(ctx context.Context, id string) error
}

// CheckRepository define las operaciones de persistencia para checks.
type CheckRepository interface {
	Save(ctx context.Context, c *domain.Check) error
	ListByMonitor(ctx context.Context, monitorID string, from, to time.Time, limit int) ([]*domain.Check, error)
	// DeleteOlderThan elimina checks anteriores a cutoff; usado por la goroutine de retención.
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}
