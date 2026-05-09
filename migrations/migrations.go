// Package migrations contiene las migraciones SQL embebidas.
// RunMigrations las aplica en orden al arrancar el servidor.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
