// Los tests en Go viven en el mismo paquete (o en package config_test para tests de caja negra).
// Aquí usamos "package config" para poder testear también funciones privadas si hiciera falta.
package config

import (
	"os"
	"testing"
)

// TestLoad_Valid verifica que cargamos correctamente un YAML bien formado.
func TestLoad_Valid(t *testing.T) {
	// os.CreateTemp crea un fichero temporal con nombre único.
	// El patrón "*.yaml" pone la extensión correcta al nombre generado.
	tmpFile, err := os.CreateTemp("", "pulse-test-*.yaml")
	if err != nil {
		// t.Fatalf detiene el test inmediatamente (como Fatal + FailNow).
		// Úsalo cuando no tiene sentido continuar si esto falla.
		t.Fatalf("no se pudo crear fichero temporal: %v", err)
	}
	// defer ejecuta la función cuando TestLoad_Valid retorne.
	// Garantiza que limpiamos el fichero aunque el test falle a mitad.
	defer os.Remove(tmpFile.Name())

	content := `
targets:
  - name: ejemplo
    url: https://example.com
    expected_status: 200
    max_latency_ms: 500
  - name: otro
    url: https://example.org
    expected_status: 301
    max_latency_ms: 1000
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("no se pudo escribir en fichero temporal: %v", err)
	}
	// Cerramos antes de leer: en algunos sistemas el buffer no se vacía hasta Close.
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("error inesperado cargando config válida: %v", err)
	}

	if len(cfg.Targets) != 2 {
		t.Errorf("esperaba 2 targets, got %d", len(cfg.Targets))
	}
	if cfg.Targets[0].Name != "ejemplo" {
		t.Errorf("esperaba name 'ejemplo', got '%s'", cfg.Targets[0].Name)
	}
	if cfg.Targets[0].ExpectedStatus != 200 {
		t.Errorf("esperaba expected_status 200, got %d", cfg.Targets[0].ExpectedStatus)
	}
	if cfg.Targets[0].MaxLatencyMs != 500 {
		t.Errorf("esperaba max_latency_ms 500, got %d", cfg.Targets[0].MaxLatencyMs)
	}
}

// TestLoad_FileNotFound verifica que devolvemos error si el fichero no existe.
func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("fichero-que-no-existe-jamas.yaml")
	// En Go, comprobamos el "camino triste" explícitamente.
	// Si err es nil cuando esperamos un error, el test falla.
	if err == nil {
		t.Error("esperaba error por fichero inexistente, pero Load devolvió nil")
	}
}

// TestLoad_EmptyTargets verifica que detectamos un YAML sin targets.
func TestLoad_EmptyTargets(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "pulse-test-*.yaml")
	if err != nil {
		t.Fatalf("no se pudo crear fichero temporal: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("targets: []\n")
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Error("esperaba error por lista de targets vacía, pero Load devolvió nil")
	}
}

// TestLoad_InvalidYAML verifica que detectamos YAML malformado.
func TestLoad_InvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "pulse-test-*.yaml")
	if err != nil {
		t.Fatalf("no se pudo crear fichero temporal: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("esto: no: es: yaml: válido: ::::\n")
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Error("esperaba error por YAML inválido, pero Load devolvió nil")
	}
}
