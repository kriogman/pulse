// go.mod define el módulo y sus dependencias directas.
//
// Cómo funciona la gestión de dependencias en Go:
//   - El "módulo" es la unidad de distribución: un repositorio = un módulo.
//   - go.mod declara el nombre del módulo (ruta de importación) y sus deps directas.
//   - go.sum almacena los hashes criptográficos de cada dep para garantizar
//     reproducibilidad: dos desarrolladores con el mismo go.sum obtienen
//     exactamente los mismos bytes de código de terceros.
//   - `go mod tidy` resuelve el árbol de dependencias transitivas,
//     actualiza go.mod con las indirectas necesarias y regenera go.sum.
//   - No hay node_modules ni vendor por defecto: Go descarga deps a
//     ~/.cache/go/pkg/mod/ (caché global compartida entre proyectos).

module github.com/your-username/pulse

// Versión mínima de Go requerida.
// Go garantiza compatibilidad hacia atrás: código Go 1.21 compila con Go 1.22+.
go 1.21

require (
	// cobra: framework estándar para CLIs en Go.
	// Gestiona subcomandos, flags, autocompletado y ayuda automática.
	github.com/spf13/cobra v1.8.0

	// yaml.v3: parser de YAML maduro, soporte completo de YAML 1.2.
	gopkg.in/yaml.v3 v3.0.1
)

// Las dependencias indirectas (deps de nuestras deps) se listan aquí
// después de ejecutar `go mod tidy`. Por ahora están vacías.
require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)
