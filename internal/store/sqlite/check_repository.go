package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kriogman/pulse/internal/domain"
	"github.com/kriogman/pulse/internal/store"
)

var _ store.CheckRepository = (*checkRepository)(nil)

type checkRepository struct {
	db *sql.DB
}

func NewCheckRepository(db *sql.DB) store.CheckRepository {
	return &checkRepository{db: db}
}

func (r *checkRepository) Save(ctx context.Context, c *domain.Check) error {
	metaJSON, err := json.Marshal(c.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling metadata for check: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO checks (monitor_id, started_at, duration_ms, status, status_code, error, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.MonitorID, c.StartedAt.UnixMilli(), c.DurationMs,
		string(c.Status), c.StatusCode, c.Error, string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf("saving check for monitor %s: %w", c.MonitorID, err)
	}
	return nil
}

func (r *checkRepository) ListByMonitor(ctx context.Context, monitorID string, from, to time.Time, limit int) ([]*domain.Check, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, monitor_id, started_at, duration_ms, status, status_code, error, metadata_json
		FROM checks
		WHERE monitor_id = ? AND started_at BETWEEN ? AND ?
		ORDER BY started_at DESC
		LIMIT ?`,
		monitorID, from.UnixMilli(), to.UnixMilli(), limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listing checks for monitor %s: %w", monitorID, err)
	}
	defer rows.Close()

	var checks []*domain.Check
	for rows.Next() {
		c, err := scanCheck(rows)
		if err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

func (r *checkRepository) Stats(ctx context.Context, monitorID string, from, to time.Time) (*domain.CheckStats, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN status = 'up'       THEN 1 ELSE 0 END), 0) AS up_count,
			COALESCE(SUM(CASE WHEN status = 'down'     THEN 1 ELSE 0 END), 0) AS down_count,
			COALESCE(SUM(CASE WHEN status = 'degraded' THEN 1 ELSE 0 END), 0) AS degraded_count,
			COALESCE(CAST(AVG(duration_ms) AS INTEGER), 0)                    AS avg_duration,
			COALESCE(MAX(duration_ms), 0)                                     AS max_duration
		FROM checks
		WHERE monitor_id = ? AND started_at BETWEEN ? AND ?`,
		monitorID, from.UnixMilli(), to.UnixMilli(),
	)

	var s domain.CheckStats
	var avgDur int64
	if err := row.Scan(&s.TotalChecks, &s.UpCount, &s.DownCount, &s.DegradedCount, &avgDur, &s.MaxDurationMs); err != nil {
		return nil, fmt.Errorf("stats for monitor %s: %w", monitorID, err)
	}
	s.AvgDurationMs = avgDur
	if s.TotalChecks > 0 {
		s.UptimePct = float64(s.UpCount) / float64(s.TotalChecks) * 100
	}
	return &s, nil
}

func (r *checkRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM checks WHERE started_at < ?`, cutoff.UnixMilli())
	if err != nil {
		return 0, fmt.Errorf("deleting old checks: %w", err)
	}
	return result.RowsAffected()
}

func scanCheck(rows *sql.Rows) (*domain.Check, error) {
	var (
		c           domain.Check
		statusStr   string
		statusCode  sql.NullInt64
		errMsg      sql.NullString
		startedAtMs int64
		metaJSON    string
	)
	if err := rows.Scan(&c.ID, &c.MonitorID, &startedAtMs, &c.DurationMs,
		&statusStr, &statusCode, &errMsg, &metaJSON); err != nil {
		return nil, fmt.Errorf("scanning check: %w", err)
	}

	c.StartedAt = time.UnixMilli(startedAtMs)
	c.Status = domain.CheckStatus(statusStr)

	if statusCode.Valid {
		n := int(statusCode.Int64)
		c.StatusCode = &n
	}
	if errMsg.Valid {
		c.Error = &errMsg.String
	}
	if err := json.Unmarshal([]byte(metaJSON), &c.Metadata); err != nil {
		c.Metadata = map[string]any{}
	}
	return &c, nil
}
