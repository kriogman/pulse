package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kriogman/pulse/internal/domain"
	"github.com/kriogman/pulse/internal/store"
)

var _ store.MonitorRepository = (*monitorRepository)(nil)

type monitorRepository struct {
	db *sql.DB
}

func NewMonitorRepository(db *sql.DB) store.MonitorRepository {
	return &monitorRepository{db: db}
}

func (r *monitorRepository) Create(ctx context.Context, m *domain.Monitor) error {
	configJSON, err := json.Marshal(m.Config)
	if err != nil {
		return fmt.Errorf("marshaling config for monitor %s: %w", m.ID, err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO monitors (id, name, type, target, interval_sec, timeout_ms, config_json, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Name, string(m.Type), m.Target, m.IntervalSec, m.TimeoutMs,
		string(configJSON), boolToInt(m.Enabled), m.CreatedAt.Unix(), m.UpdatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("inserting monitor %s: %w", m.ID, err)
	}
	return nil
}

func (r *monitorRepository) GetByID(ctx context.Context, id string) (*domain.Monitor, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, type, target, interval_sec, timeout_ms, config_json, enabled, created_at, updated_at
		FROM monitors WHERE id = ?`, id)
	m, err := scanMonitor(row)
	if errors.Is(err, domain.ErrMonitorNotFound) {
		return nil, domain.ErrMonitorNotFound
	}
	return m, err
}

func (r *monitorRepository) List(ctx context.Context) ([]*domain.Monitor, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, type, target, interval_sec, timeout_ms, config_json, enabled, created_at, updated_at
		FROM monitors ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing monitors: %w", err)
	}
	defer rows.Close()

	var monitors []*domain.Monitor
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

func (r *monitorRepository) Update(ctx context.Context, m *domain.Monitor) error {
	configJSON, err := json.Marshal(m.Config)
	if err != nil {
		return fmt.Errorf("marshaling config for monitor %s: %w", m.ID, err)
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE monitors
		SET name=?, type=?, target=?, interval_sec=?, timeout_ms=?, config_json=?, enabled=?, updated_at=?
		WHERE id=?`,
		m.Name, string(m.Type), m.Target, m.IntervalSec, m.TimeoutMs,
		string(configJSON), boolToInt(m.Enabled), m.UpdatedAt.Unix(), m.ID,
	)
	if err != nil {
		return fmt.Errorf("updating monitor %s: %w", m.ID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrMonitorNotFound
	}
	return nil
}

func (r *monitorRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM monitors WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting monitor %s: %w", id, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrMonitorNotFound
	}
	return nil
}

// scanner unifica *sql.Row y *sql.Rows para reutilizar scanMonitor.
type scanner interface {
	Scan(dest ...any) error
}

func scanMonitor(s scanner) (*domain.Monitor, error) {
	var (
		m          domain.Monitor
		monType    string
		configJSON string
		enabledInt int
		createdAt  int64
		updatedAt  int64
	)
	err := s.Scan(&m.ID, &m.Name, &monType, &m.Target,
		&m.IntervalSec, &m.TimeoutMs, &configJSON,
		&enabledInt, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrMonitorNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning monitor: %w", err)
	}

	m.Type = domain.MonitorType(monType)
	m.Enabled = enabledInt == 1
	m.CreatedAt = time.Unix(createdAt, 0)
	m.UpdatedAt = time.Unix(updatedAt, 0)

	if err := json.Unmarshal([]byte(configJSON), &m.Config); err != nil {
		return nil, fmt.Errorf("unmarshaling config for monitor %s: %w", m.ID, err)
	}
	return &m, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
