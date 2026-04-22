// Package config gestiona la carga y validación del fichero YAML de configuración.
//
// Por qué está en "internal/":
//   En Go, cualquier paquete dentro de un directorio "internal/" solo puede ser
//   importado por código dentro del mismo árbol de directorios. Es encapsulación
//   a nivel de módulo: le decimos al compilador que config y checker son
//   implementación interna, no API pública del módulo.
//
//   Si quisiéramos que otras herramientas pudiesen importar este paquete,
//   lo moveríamos a "pkg/" (convención para código reutilizable externo).
//   Para este CLI, "internal/" es la elección correcta.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Target representa un endpoint a chequear.
//
// Los tags `yaml:"..."` le dicen al parser de YAML qué clave del fichero
// corresponde a cada campo de la struct. Sin el tag, yaml.v3 buscaría
// por nombre en minúsculas ("name", "url"…), así que aquí son redundantes,
// pero es buena práctica ser explícito.
//
// Los nombres de campos DEBEN empezar en mayúscula para ser "exportados"
// (visibles desde otros paquetes). En Go no hay public/private: mayúscula = público.
type Target struct {
	Name           string `yaml:"name"`
	URL            string `yaml:"url"`
	ExpectedStatus int    `yaml:"expected_status"`
	MaxLatencyMs   int64  `yaml:"max_latency_ms"`
}

// Config es la estructura raíz del fichero YAML.
// El campo "targets" del YAML se deserializa en el slice Targets.
// Un slice ([]Target) es como un array de tamaño dinámico.
type Config struct {
	Targets []Target `yaml:"targets"`
}

// Load lee el fichero YAML en la ruta dada y lo deserializa en un *Config.
//
// Firma idiomática en Go: devolvemos (valor, error).
//   - Si hay error, devolvemos (nil, err): el llamante DEBE comprobar el error.
//   - Si todo va bien, devolvemos (*Config, nil).
//
// Go no tiene excepciones. Los errores son valores normales que se propagan
// hacia arriba de la pila de llamadas. Esto hace el flujo de errores explícito
// y evita "sorpresas" en tiempo de ejecución.
func Load(path string) (*Config, error) {
	// os.ReadFile lee el fichero completo en memoria como []byte.
	// Si el fichero no existe o no hay permisos, devuelve error.
	data, err := os.ReadFile(path)
	if err != nil {
		// fmt.Errorf con %w "envuelve" el error original.
		// Ventaja: quien llame a Load puede usar errors.Is() o errors.As()
		// para inspeccionar la causa raíz sin perder el contexto del mensaje.
		return nil, fmt.Errorf("no se pudo leer el fichero '%s': %w", path, err)
	}

	// var cfg Config declara cfg con su zero value.
	// En Go, todas las variables tienen zero value por defecto (0, "", nil, false…).
	// Aquí cfg.Targets será nil (slice vacío).
	var cfg Config

	// yaml.Unmarshal convierte los bytes YAML en la struct Go.
	// Pasamos &cfg (puntero a cfg) para que la función pueda modificar la variable.
	// Si pasásemos cfg por valor, yaml.Unmarshal modificaría una copia local.
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parseando YAML en '%s': %w", path, err)
	}

	if len(cfg.Targets) == 0 {
		return nil, fmt.Errorf("el fichero '%s' no tiene targets definidos", path)
	}

	// Devolvemos puntero (&cfg) para evitar copiar la struct entera.
	// En structs pequeñas no importa mucho, pero es el patrón habitual.
	return &cfg, nil
}
