// Package web expone el frontend compilado como un embed.FS.
// Requiere que web/dist/ haya sido construido con "npm run build" antes de compilar Go.
package web

import "embed"

//go:embed all:dist
var FS embed.FS
