// Package checker contiene la lógica principal: hacer peticiones HTTP en paralelo
// y evaluar si cada endpoint cumple las condiciones esperadas.
package checker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/your-username/pulse/internal/config"
)

// Result guarda el resultado de chequear un único target.
//
// Los tags `json:"..."` controlan cómo se serializa a JSON.
// "omitempty" significa "omitir este campo si está vacío" —
// así Reason no aparece en el JSON cuando el check es OK.
type Result struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
	LatencyMs  int64  `json:"latency_ms"`
	OK         bool   `json:"ok"`
	Reason     string `json:"reason,omitempty"`
}

// checkOne hace un único GET HTTP y devuelve el resultado.
//
// Es una función "privada" (empieza en minúscula): solo es visible dentro
// del paquete checker. Esto es intencional: los llamantes externos solo
// necesitan RunAll, no la mecánica de un check individual.
func checkOne(target config.Target, timeoutSecs int) Result {
	// Creamos un cliente HTTP con timeout explícito.
	// El cliente por defecto (http.DefaultClient) NO tiene timeout —
	// una petición podría bloquearse para siempre sin esto.
	client := &http.Client{
		Timeout: time.Duration(timeoutSecs) * time.Second,
	}

	// Capturamos el tiempo de inicio para medir latencia.
	start := time.Now()

	resp, err := client.Get(target.URL)

	// time.Since(start) equivale a time.Now().Sub(start).
	// Lo calculamos aquí para incluir el tiempo de leer headers,
	// pero antes de leer el body (que no nos interesa).
	latencyMs := time.Since(start).Milliseconds()

	// Comprobamos el error ANTES de usar resp.
	// Si err != nil, resp puede ser nil y acceder a resp.StatusCode provocaría un panic.
	// Este patrón "comprueba el error antes de usar el valor" es fundamental en Go.
	if err != nil {
		return Result{
			Name:      target.Name,
			URL:       target.URL,
			LatencyMs: latencyMs,
			OK:        false,
			// %v formatea el error como string. Para errores usamos %v, no %s.
			Reason: fmt.Sprintf("error de conexión: %v", err),
		}
	}

	// defer ejecuta Body.Close() cuando checkOne retorne.
	// El body de la respuesta HTTP es un stream abierto: SIEMPRE hay que cerrarlo
	// para liberar la conexión al pool de conexiones reutilizables.
	// Sin esto, agotaríamos los file descriptors del sistema operativo.
	defer resp.Body.Close()

	// Construimos el resultado base asumiendo OK.
	// Luego lo modificamos si detectamos problemas.
	result := Result{
		Name:       target.Name,
		URL:        target.URL,
		StatusCode: resp.StatusCode,
		LatencyMs:  latencyMs,
		OK:         true,
	}

	// Verificación 1: status code.
	if resp.StatusCode != target.ExpectedStatus {
		result.OK = false
		result.Reason = fmt.Sprintf("status esperado %d, recibido %d",
			target.ExpectedStatus, resp.StatusCode)
	}

	// Verificación 2: latencia máxima.
	// Solo chequeamos si el target tiene MaxLatencyMs > 0.
	// Si el target ya falló por status, concatenamos la razón de latencia.
	if target.MaxLatencyMs > 0 && latencyMs > target.MaxLatencyMs {
		result.OK = false
		latencyReason := fmt.Sprintf("latencia %dms excede el límite de %dms",
			latencyMs, target.MaxLatencyMs)
		if result.Reason != "" {
			result.Reason += "; " + latencyReason
		} else {
			result.Reason = latencyReason
		}
	}

	return result
}

// RunAll ejecuta checkOne para cada target en paralelo y devuelve todos los resultados.
//
// ─── Por qué WaitGroup + channel y no solo WaitGroup con mutex ───────────────
//
// Opción A — WaitGroup + mutex + slice:
//   var mu sync.Mutex
//   var results []Result
//   go func() { mu.Lock(); results = append(results, r); mu.Unlock() }()
//
// Opción B — WaitGroup + channel (la que usamos):
//   resultsCh := make(chan Result, N)
//   go func() { resultsCh <- r }()
//
// Elegimos B porque:
//   1. El channel es el mecanismo idiomático de Go para comunicar datos entre
//      goroutines ("Do not communicate by sharing memory; share memory by communicating").
//   2. No necesitamos pensar en locks: el channel serializa las escrituras internamente.
//   3. El código es más legible: el channel expresa claramente la intención
//      ("esta goroutine produce resultados, la goroutine principal los consume").
//
// Usamos WaitGroup *además* del channel porque necesitamos saber cuándo
// cerrar el channel (close(resultsCh)). Sin WaitGroup, no sabríamos cuándo
// todas las goroutines han terminado de escribir.
func RunAll(targets []config.Target, timeoutSecs int) []Result {
	// Canal con buffer de tamaño len(targets).
	// Un canal con buffer permite que el emisor escriba hasta N veces
	// sin bloquearse esperando al receptor. Aquí N = número de targets,
	// así cada goroutine puede escribir su resultado sin esperar.
	//
	// Un canal sin buffer (make(chan Result)) bloquearía a cada goroutine
	// hasta que alguien leyera — en nuestro caso sería un deadlock porque
	// el lector (range resultsCh) empieza después de lanzar todas las goroutines.
	resultsCh := make(chan Result, len(targets))

	// WaitGroup es un contador con tres operaciones:
	//   Add(n): suma n al contador
	//   Done():  resta 1 al contador (equivale a Add(-1))
	//   Wait():  bloquea hasta que el contador llega a 0
	var wg sync.WaitGroup

	for _, target := range targets {
		// Incrementamos ANTES de lanzar la goroutine.
		// Si lo hiciésemos dentro de la goroutine, podría haber una race condition:
		// wg.Wait() podría ver el contador a 0 antes de que Add(1) se ejecute.
		wg.Add(1)

		// Lanzamos una goroutine por target.
		// Una goroutine es una función que se ejecuta concurrentemente,
		// gestionada por el runtime de Go (no es un thread del SO).
		// El runtime puede multiplexar miles de goroutines sobre pocos threads.
		//
		// TRAMPA CLÁSICA: si hiciésemos `go func() { checkOne(target, ...) }()`
		// sin pasar target como argumento, todas las goroutines compartirían
		// la misma variable 'target' del loop — que para cuando se ejecuten
		// podría ya valer el último target de la lista.
		// La solución: pasar target como argumento (se copia por valor).
		go func(t config.Target) {
			// defer wg.Done() garantiza que decrementamos aunque checkOne
			// entre en pánico (comportamiento excepcional en Go).
			defer wg.Done()
			resultsCh <- checkOne(t, timeoutSecs)
		}(target) // <-- target se copia aquí al llamar a la función anónima
	}

	// Goroutine auxiliar: espera a que todas terminen y cierra el canal.
	// Cerramos en una goroutine separada para no bloquear este hilo
	// mientras los resultados llegan al canal.
	// Cerrar el canal es la señal para que el `range` de abajo termine.
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Recogemos resultados del canal hasta que se cierre.
	// `range` sobre un canal itera hasta que el canal está cerrado Y vacío.
	var results []Result
	for r := range resultsCh {
		results = append(results, r)
	}

	return results
}

// PrintResults muestra los resultados según el formato elegido.
// Devuelve true si todos los checks pasaron, false si alguno falló.
func PrintResults(results []Result, format string) bool {
	allOK := true
	for _, r := range results {
		if !r.OK {
			allOK = false
			// No hacemos break: queremos evaluar todos para el output completo.
		}
	}

	// switch en Go no necesita break: solo ejecuta el case que coincide.
	switch format {
	case "json":
		printJSON(results)
	default:
		// Cualquier valor desconocido cae aquí, incluyendo "text".
		printText(results)
	}

	return allOK
}

// printText muestra los resultados en formato legible para humanos.
func printText(results []Result) {
	for _, r := range results {
		verdict := "OK  "
		if !r.OK {
			verdict = "FAIL"
		}

		if r.StatusCode == 0 {
			// Sin status code: hubo error de conexión antes de recibir respuesta.
			fmt.Printf("[%s] %-20s %s — %dms — %s\n",
				verdict, r.Name, r.URL, r.LatencyMs, r.Reason)
		} else {
			line := fmt.Sprintf("[%s] %-20s %s — HTTP %d — %dms",
				verdict, r.Name, r.URL, r.StatusCode, r.LatencyMs)
			if r.Reason != "" {
				line += " — " + r.Reason
			}
			fmt.Println(line)
		}
	}
}

// printJSON serializa los resultados como un array JSON con indentación.
func printJSON(results []Result) {
	// json.MarshalIndent devuelve ([]byte, error).
	// Ignoramos el error aquí porque:
	//   1. Solo puede fallar si los datos contienen tipos no serializables (ej: canales, funciones).
	//   2. Result solo contiene string, int, bool — siempre serializables.
	// En producción documentaríamos esta decisión o devolvería error.
	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Printf(`[{"error": "%v"}]`+"\n", err)
		return
	}
	fmt.Println(string(output))
}
