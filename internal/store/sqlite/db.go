package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // registra el driver "sqlite" (pure Go, sin CGO)

	"github.com/kriogman/pulse/internal/store"
)

// dsn construye el connection string con parámetros de producción obligatorios.
//
// WAL mode:         lectores concurrentes + un escritor sin bloqueo mutuo.
// busy_timeout=5000: si el CLI está escribiendo, espera 5s en lugar de fallar.
// foreign_keys=on:  SQLite las desactiva por defecto; las necesitamos.
// synchronous=NORMAL: balance durabilidad/rendimiento correcto con WAL.
// txlock=immediate: previene "database is locked" en transacciones write.
func dsn(path string) string {
	return fmt.Sprintf(
		"file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&_synchronous=NORMAL&_txlock=immediate",
		path,
	)
}

// Repos agrupa los repositorios para facilitar el wiring en tests y main.
type Repos struct {
	Monitors store.MonitorRepository
	Checks   store.CheckRepository
}

// Open abre la base de datos SQLite en path con configuración de producción.
//
// Pool: MaxOpenConns=1, MaxIdleConns=1.
// SQLite serializa escrituras internamente; un pool mayor solo añade contención
// sin ganar throughput. Esta decisión está documentada aquí para no "optimizarse" sin medir.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("opening sqlite at %s: %w", path, err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging sqlite at %s: %w", path, err)
	}

	return db, nil
}
