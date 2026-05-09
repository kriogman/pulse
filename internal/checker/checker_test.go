package checker

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kriogman/pulse/internal/config"
)

// TestCheckOne_OK verifica el camino feliz: servidor responde con el status esperado.
//
// httptest.NewServer levanta un servidor HTTP real en un puerto aleatorio del loopback.
// Es el mecanismo idiomático en Go para testear código que hace peticiones HTTP:
// no hay mocks de interfaz, sino un servidor real que controlamos.
func TestCheckOne_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // 200
	}))
	defer server.Close()

	target := config.Target{
		Name:           "test-ok",
		URL:            server.URL,
		ExpectedStatus: 200,
		MaxLatencyMs:   5000,
	}

	result := checkOne(target, 5)

	if !result.OK {
		t.Errorf("esperaba resultado OK, got FAIL: %s", result.Reason)
	}
	if result.StatusCode != 200 {
		t.Errorf("esperaba StatusCode 200, got %d", result.StatusCode)
	}
	if result.Reason != "" {
		t.Errorf("esperaba Reason vacío, got '%s'", result.Reason)
	}
}

// TestCheckOne_WrongStatus verifica que detectamos un status code inesperado.
func TestCheckOne_WrongStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // 500
	}))
	defer server.Close()

	target := config.Target{
		Name:           "test-status",
		URL:            server.URL,
		ExpectedStatus: 200, // Esperamos 200, recibiremos 500
		MaxLatencyMs:   5000,
	}

	result := checkOne(target, 5)

	if result.OK {
		t.Error("esperaba FAIL por status incorrecto, pero got OK")
	}
	if result.StatusCode != 500 {
		t.Errorf("esperaba StatusCode 500, got %d", result.StatusCode)
	}
	if result.Reason == "" {
		t.Error("esperaba Reason no vacío cuando falla por status")
	}
}

// TestCheckOne_Timeout verifica que detectamos cuando el servidor no responde.
func TestCheckOne_Timeout(t *testing.T) {
	// Este handler bloquea hasta que el cliente se desconecta.
	// r.Context().Done() es un canal que se cierra cuando la request se cancela
	// (por timeout del cliente, cierre de conexión, etc.).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	target := config.Target{
		Name:           "test-timeout",
		URL:            server.URL,
		ExpectedStatus: 200,
		MaxLatencyMs:   5000,
	}

	// Timeout de 1s para no ralentizar el test suite.
	result := checkOne(target, 1)

	if result.OK {
		t.Error("esperaba FAIL por timeout, pero got OK")
	}
	if result.StatusCode != 0 {
		t.Errorf("esperaba StatusCode 0 (sin respuesta), got %d", result.StatusCode)
	}
}

// TestCheckOne_LatencyExceeded verifica que detectamos latencia excesiva.
func TestCheckOne_LatencyExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Añadimos delay artificial para simular un servidor lento.
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	target := config.Target{
		Name:           "test-latencia",
		URL:            server.URL,
		ExpectedStatus: 200,
		MaxLatencyMs:   50, // Límite de 50ms, el servidor tarda ~150ms
	}

	result := checkOne(target, 5)

	if result.OK {
		t.Error("esperaba FAIL por latencia excedida, pero got OK")
	}
	// El status sí llegó bien (200), pero la latencia excede el límite.
	if result.StatusCode != 200 {
		t.Errorf("esperaba StatusCode 200, got %d", result.StatusCode)
	}
}

// TestCheckOne_ZeroMaxLatency verifica que MaxLatencyMs=0 deshabilita el check de latencia.
func TestCheckOne_ZeroMaxLatency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	target := config.Target{
		Name:           "test-sin-limite-latencia",
		URL:            server.URL,
		ExpectedStatus: 200,
		MaxLatencyMs:   0, // 0 = sin límite de latencia
	}

	result := checkOne(target, 5)

	if !result.OK {
		t.Errorf("esperaba OK cuando MaxLatencyMs=0, got FAIL: %s", result.Reason)
	}
}

// TestRunAll verifica que RunAll ejecuta todos los checks y devuelve un resultado por target.
func TestRunAll(t *testing.T) {
	serverOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer serverOK.Close()

	serverFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable) // 503
	}))
	defer serverFail.Close()

	targets := []config.Target{
		{Name: "ok-target", URL: serverOK.URL, ExpectedStatus: 200, MaxLatencyMs: 5000},
		{Name: "fail-target", URL: serverFail.URL, ExpectedStatus: 200, MaxLatencyMs: 5000},
	}

	results := RunAll(targets, 5)

	if len(results) != 2 {
		t.Fatalf("esperaba 2 resultados, got %d", len(results))
	}

	// Los resultados llegan en orden no determinista (son paralelos),
	// así que los indexamos por nombre para buscarlos con seguridad.
	resultMap := make(map[string]Result)
	for _, r := range results {
		resultMap[r.Name] = r
	}

	if _, exists := resultMap["ok-target"]; !exists {
		t.Fatal("falta el resultado de 'ok-target'")
	}
	if _, exists := resultMap["fail-target"]; !exists {
		t.Fatal("falta el resultado de 'fail-target'")
	}

	if !resultMap["ok-target"].OK {
		t.Errorf("'ok-target' debería ser OK, reason: %s", resultMap["ok-target"].Reason)
	}
	if resultMap["fail-target"].OK {
		t.Error("'fail-target' debería ser FAIL")
	}
}
