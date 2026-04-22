// Package main es el punto de entrada del programa.
//
// Por qué cmd/pulse/main.go y no solo main.go en la raíz:
//
//   La convención Go para proyectos con múltiples binarios (o que podrían tenerlos)
//   es poner cada ejecutable en cmd/<nombre>/main.go.
//
//   Estructura de carpetas en proyectos Go:
//     cmd/      → Puntos de entrada. Cada subdirectorio = un binario compilable.
//                 Solo contiene main.go + wiring de dependencias. Lógica mínima.
//     internal/ → Paquetes privados del módulo. No importables desde fuera.
//                 Aquí va la lógica de negocio: config, checker, etc.
//     pkg/      → Paquetes públicos reutilizables por otros módulos.
//                 En este proyecto no tenemos (es un CLI, no una librería).
//
//   Este diseño separa "qué hace el programa" (internal/) de
//   "cómo se arranca" (cmd/), facilitando tests y futuras extensiones.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/your-username/pulse/internal/checker"
	"github.com/your-username/pulse/internal/config"
)

func main() {
	// rootCmd es el comando raíz ("pulse").
	// Cobra estructura los CLI como un árbol de comandos.
	rootCmd := &cobra.Command{
		Use:   "pulse",
		Short: "pulse chequea el estado de endpoints HTTP",
		Long: `pulse lee una lista de endpoints desde un fichero YAML
y los chequea en paralelo, reportando el estado de cada uno.

Ejemplo:
  pulse check -c pulse.yaml
  pulse check --format json --timeout 10`,
	}

	// Declaramos las variables donde Cobra almacenará los flags.
	// Viven aquí (en el closure de la función que construye checkCmd)
	// para que RunE pueda acceder a ellas.
	var configPath string
	var timeoutSecs int
	var format string

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Chequea todos los endpoints definidos en el fichero de configuración",
		// RunE es como Run pero permite devolver un error.
		// Cobra imprimirá el error y el "uso" del comando automáticamente.
		// Usamos RunE en lugar de Run siempre que podamos devolver un error.
		RunE: func(cmd *cobra.Command, args []string) error {
			// Paso 1: carga de configuración.
			// Si Load falla, devolvemos el error envuelto con contexto.
			cfg, err := config.Load(configPath)
			if err != nil {
				// fmt.Errorf con %w preserva el error original para que
				// el llamante pueda usar errors.Is() o errors.As().
				return fmt.Errorf("cargando configuración: %w", err)
			}

			// Paso 2: ejecutar checks en paralelo.
			results := checker.RunAll(cfg.Targets, timeoutSecs)

			// Paso 3: mostrar resultados.
			allOK := checker.PrintResults(results, format)

			// Paso 4: exit code.
			// RunE no puede controlar el exit code directamente.
			// Usamos os.Exit(1) para indicar fallo cuando algún check falló.
			// Esto es útil en CI: `pulse check` retornará 1 si hay fallos.
			//
			// NOTA: os.Exit omite defer's — aquí no hay ninguno, así que es seguro.
			if !allOK {
				os.Exit(1)
			}
			return nil
		},
	}

	// Registramos los flags del subcomando.
	// StringVarP = flag de tipo string, con nombre largo y corto (-c / --config).
	// IntVarP    = flag de tipo int, con nombre largo y corto (-t / --timeout).
	// StringVar  = flag de tipo string, solo nombre largo (--format).
	checkCmd.Flags().StringVarP(&configPath, "config", "c", "pulse.yaml",
		"ruta al fichero de configuración YAML")
	checkCmd.Flags().IntVarP(&timeoutSecs, "timeout", "t", 5,
		"timeout global en segundos para cada petición")
	checkCmd.Flags().StringVar(&format, "format", "text",
		"formato de salida: 'text' (default) o 'json'")

	rootCmd.AddCommand(checkCmd)

	// Execute parsea os.Args[1:] y ejecuta el comando correspondiente.
	// Si hay un error de parsing o RunE devuelve error, Cobra lo imprime.
	if err := rootCmd.Execute(); err != nil {
		// Cobra ya imprimió el error; solo necesitamos el exit code.
		os.Exit(1)
	}
}
