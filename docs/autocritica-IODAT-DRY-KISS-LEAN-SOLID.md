# Autocrítica ioDat: DRY + KISS + LEAN + SOLID

> Documento generado el 2026-05-12 con base en análisis del código fuente.
> Cada hallazgo está diseñado para ser una tarea independiente ejecutable
> en un worktree Git separado.

---

## Tabla de contenidos

1. [Resumen ejecutivo](#1-resumen-ejecutivo)
2. [DRY — violaciones](#2-dry--violaciones)
3. [KISS — violaciones](#3-kiss--violaciones)
4. [LEAN — violaciones](#4-lean--violaciones)
5. [SOLID — violaciones](#5-solid--violaciones)
6. [Archivos afectados por hallazgo](#6-archivos-afectados-por-hallazgo)
7. [Propuesta de estructura futura](#7-propuesta-de-estructura-futura)
8. [Recomendaciones de prioridad](#8-recomendaciones-de-prioridad)

---

## 1. Resumen ejecutivo

| Métrica | Valor |
|---------|-------|
| Líneas totales (Go) | 1,584 |
| Archivos fuente | 11 |
| Tests | 5 archivos cubriendo solo helpers |
| Plataformas | 3 (linux, darwin, windows) |
| Commits en `develop` | 2 |
| `go vet` | ✅ limpio (cero issues) |

**Problema central:** el código funciona pero está organizado como si hubiera sido escrito
en una sola sesión sin refactor posterior. La repetición entre plataformas es alta,
los tests son marginales, y varias decisiones de diseño (parsing artesanal, falta de
timeouts, error handling inconsistente) no sobrevivirían una auditoría de seguridad real.

---

## 2. DRY — Violaciones

### D1. Helpers de parseo duplicados por plataforma

**Dónde:** `collector_linux.go:346` (`parseInt64`), `collector_windows.go:355` (`parseInt`, `parseFloat`)

**Problema:** Tres implementaciones de `string → int` que hacen exactamente lo mismo.
`parseInt64` en Linux es idéntica en comportamiento a `parseInt` en Windows pero
con distinto tipo de retorno (`int64` vs `int`). `parseFloat` solo existe en Windows.

**Severidad:** alta. 3 copy-pastes, 0 tests compartidos.

**Propuesta:** mover a `utils.go` un conjunto único de helpers tipados:

```go
func ParseInt(s string) int
func ParseInt64(s string) int64
func ParseFloat64(s string) float64
```

---

### D2. parseBlocksToGB (Linux) vs parseSizeToGB (macOS)

**Dónde:** `collector_linux.go:333`, `collector_darwin.go:311`

**Problema:** Misma función con distinto nombre y lógica:
- Linux: multiplica bloques de 512 bytes → GB (base 1000)
- macOS: parsea string "500.24 GB" con regex
- Windows: parsea bytes en crudo (`TotalPhysicalMemory`)

Tres formas distintas de llegar a la misma unidad (GB). Deberían converger en una
sola función con un `type ByteSize int64` y formato consistente.

**Severidad:** media. No es bug (cada SO entrega la info distinto), pero la variación
en precisión (base 1000 vs 1024) es fuente de bugs sutiles.

**Propuesta:**

```go
// En types.go
type ByteSize int64

const MB ByteSize = 1 << 20
const GB ByteSize = 1 << 30

func ParseByteSize(s string) (ByteSize, error) // para strings como "500.24 GB"
func ByteSizeFromBlocks(blocks int64) ByteSize  // para Linux /sys/block/size
func ByteSizeFromBytes(bytes int64) ByteSize    // para Windows TotalPhysicalMemory
```

---

### D3. Filtrado de loopback en 3 plataformas

**Dónde:** `collector_linux.go:300` (`name == "lo"`), `collector_darwin.go:278` (`current == "lo0"`),
`collector_windows.go:239` (probablemente similar)

**Problema:** Cada platforma hace `if name == "lo" { continue }` con el nombre específico
de su SO. La lógica de filtrado debería vivir en un lugar común o en una constante por
plataforma.

**Severidad:** baja. Pero si alguien agrega Linux en una arquitectura donde loopback
se llame distinto (loN, lo.something), hay que acordarse de cambiarlo en 3 archivos.

---

### D4. Invocación de comandos en 3 versiones

**Dónde:**
- Linux: `exec.Command("uname", ...)`, `exec.Command("lspci", ...)`, `exec.Command("ip", ...)`
- macOS: `exec.Command(name, args...)` envuelto en `runCmd`, más `exec.Command("bash", "-c", script)`
- Windows: `exec.Command("powershell", "-NoProfile", "-Command", script)` envuelto en 4 wrappers

**Problema:** No hay una abstracción común para ejecutar comandos externos. Esto
impide:
1. Testear colección sin comandos reales (no se puede mockear)
2. Aplicar timeouts de forma consistente
3. Registrar comandos ejecutados para auditoría

**Severidad:** alta. Sin esta abstracción, la testabilidad es cero para el core del collector.

**Propuesta:**

```go
// En un nuevo archivo exec.go (sin build tag)
type CommandRunner interface {
    Run(name string, args ...string) (string, error)
    RunJSON(script string, dest interface{}) error
}

type OSCommandRunner struct{} // implementación real

// Cada plataforma recibe un CommandRunner en Run():
func Run(runner CommandRunner) (*Inventory, error)
```

---

### D5. cleanCPUName duplicado en concepto

**Dónde:** `utils.go:8` (compartido 👍), pero `NameClean` se recalcula en cada
plataforma individualmente.

**Problema:** El llamado a `cleanCPUName()` está en cada `getCPU()` de cada plataforma.
La lógica de limpieza está centralizada, pero la invocación está duplicada. Si mañana
cambia el signature de `cleanCPUName`, hay que tocarlo en 3 archivos.

**Severidad:** baja. La función ya está en `utils.go` (bien), la invocación repetida
es ruido más que duplicación real.

---

## 3. KISS — Violaciones

### K1. PowerShell wrappers duplicados (4 funciones para lo mismo)

**Dónde:** `collector_windows.go:317-348`

**Problema:** Cuatro funciones que envuelven `exec.Command("powershell", ...)`:

| Función | Trim | Error handling | Uso |
|---------|------|----------------|-----|
| `ps(script)` | No | Retorna error | 1 llamada (`hostname`) |
| `psRaw(script)` | Sí | Retorna error | 1 llamada (`getSoftware`) |
| `psJSON(script)` | Sí | Silencia error → `"[]"` | ~8 llamadas |
| `psGet(wmiClass, property)` | Sí | Silencia error → `""` | ~8 llamadas |

**Problemas:**
- `psJSON` y `psGet` tragan errores silenciosamente (fallo de WMI → campo vacío sin log)
- `ps` vs `psRaw` se distinguen solo por un `TrimSpace`
- `psJSON` usa `ConvertTo-Json -Depth 2` que trunca objetos anidados

**Propuesta:** Una sola función `runPS(script string, opts ...PSOption)` con opciones
`WithTrim()`, `WithJSON()`, `WithErrorLog()`. O mejor: solo `psJSON` y `psGet` con
error logging obligatorio.

---

### K2. WMI: mezcla de Get-WmiObject (deprecado) y Get-CimInstance

**Dónde:** `collector_windows.go`

**Problema:** `getSystemInfo` usa `Get-WmiObject Win32_ComputerSystem` como fallback
si `Get-CimInstance` falla. `getCPU` hace lo mismo. `getSoftware` usa directamente
registry. La mezcla duplica código (cada función tiene try-CimInstance → fallback-WmiObject)
y usa una API deprecada desde PowerShell 3.0.

**Propuesta:** Usar **solo** `Get-CimInstance` con `-ErrorAction Stop`. Si falla,
loguear y devolver datos parciales. Sin fallback a WMI.

---

### K3. macOS: `bash -c` para system_profiler

**Dónde:** `collector_darwin.go:43`

**Problema:** `runJSON` ejecuta `exec.Command("bash", "-c", script)`. Esto es
innecesario — `exec.Command` puede ejecutar `system_profiler` directamente con args:

```go
cmd := exec.Command("system_profiler", "SPHardwareDataType", "-json")
```

`bash -c` introduce un shell intermedio que:
- Requiere escapado de caracteres especiales
- Es más lento (fork adicional)
- Rompe `LANG`/locale (shell hereda entorno)
- Es vector potencial si el script incluyera input del usuario

**Propuesta:** Eliminar `runJSON`, reemplazar con `exec.Command` directo:

```go
type SPDataType string
const (
    SPHardware SPDataType = "SPHardwareDataType"
    SPMemory   SPDataType = "SPMemoryDataType"
    // etc
)

func runSystemProfiler(dtype SPDataType, dest interface{}) error {
    cmd := exec.Command("system_profiler", string(dtype), "-json")
    out, err := cmd.Output()
    if err != nil {
        return err
    }
    return json.Unmarshal(out, dest)
}
```

---

### K4. macOS: parseo de `ifconfig` con regex

**Dónde:** `collector_darwin.go:268-308`

**Problema:** Análisis línea por línea con 3 regex (`re`, `macRe`, `ipRe`).
El formato `ifconfig` cambia entre versiones de macOS y con el locale. No hay
una opción `-json` en macOS `ifconfig`.

**Propuesta alternativa:** Usar `netstat -I <iface>` o `networksetup -getinfo`
para datos de red, o delegar a `ipconfig` (disponible en macOS también).

---

### K5. Linux: EDID parsing manual

**Dónde:** `collector_linux.go:247-288`

**Problema:** Extracción de bytes sueltos del bloque EDID:

```go
mf := string([]byte{edid[8] + 'A' - 1, edid[9] + 'A' - 1, edid[10] + 'A' - 1})
```

Esto asume EDID v1.3/v1.4 bloque 0 de 128 bytes. Monitores 4K/HDR usan EDID
extendido (256+ bytes). El cálculo de fabricante con `+ 'A' - 1` es frágil si
los bytes vienen corruptos.

**Propuesta:** Introducir una dependencia mínima para EDID parsing, o al menos
validar checksum y versión del bloque antes de parsear.

---

### K6. Sin timeouts en exec.Command

**Dónde:** TODOS los `exec.Command` en las 3 plataformas.

**Problema:** `system_profiler` se congela en MacBooks con GPU Radeon (~5% de
los casos). `lspci` puede colgarse en kernels buggy. `powershell` puede colgar
si WMI está corrupto. 0 (cero) llamadas tienen `context.WithTimeout`.

**Propuesta:** Definir una constante de timeout por comando (ej: 10s para lspci,
30s para system_profiler, 30s para PowerShell):

```go
func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    cmd := exec.CommandContext(ctx, name, args...)
    return cmd.Output()
}
```

---

### K7. main.go: argument parsing artesanal

**Dónde:** `cmd/iodat/main.go:24-43`

**Problema:** Flag parsing manual con `range os.Args`. Aunque Go 1.26 tiene
`flag` en stdlib, no se usa. Además los flags solo soportan:
- `--stdout` / `-s`
- `--help` / `-h`
- Un path como argumento posicional

No hay validación de que `outputDir` exista (solo se verifica con `os.Stat`).
No hay flag `--version`. No hay subcomandos.

**Propuesta:** Usar `flag.Func` o un parser mínimo. O agregar un flag `--output`
explícito y tratar paths posicionales como destino.

---

## 4. LEAN — Violaciones

### L1. Tests ausentes en el core

**Dónde:** Todos los archivos `*_test.go`

**Estadística actual:**

| Archivo | Funciones probadas | LoC test | Ratio |
|---------|-------------------|----------|-------|
| `utils_test.go` | 1 función | 25 | ✅ |
| `collector_linux_test.go` | 2 helpers | 42 | ⚠️ solo helpers |
| `collector_windows_test.go` | 4 helpers | 99 | ⚠️ solo helpers |
| `collector_darwin_test.go` | 1 helper | 25 | ⚠️ solo helpers |

**Cero** tests prueban `getCPU()`, `getRAM()`, `getStorage()`, `getNetwork()`,
`getGPU()`, `getMonitors()`, `getSoftware()`, `getSystemInfo()`, `getMotherboard()`.

Es decir, 0% del collector real está testeado.

**Propuesta:** Tests parametrizados (table-driven) con archivos fixture:
- `/testdata/linux/proc_cpuinfo` con contenido simulado
- `/testdata/linux/sys_class_dmi/` con archivos simulados
- `/testdata/windows/powershell_outputs/*.json`
- `/testdata/darwin/system_profiler_outputs/*.json`

```go
// Ejemplo en linux_test.go
func TestGetCPU(t *testing.T) {
    // Usar testutil.WithFakeFS(map[string]string{
    //     "/proc/cpuinfo": readFixture("linux/proc_cpuinfo_8cores"),
    // }) para injectar datos falsos
}
```

---

### L2. output.go: dos serializaciones del JSON

**Dónde:** `output.go:25-40`

**Problema:** `GenerateOutput` serializa el JSON dos veces:

```go
data, _ := json.MarshalIndent(inv, "", "  ")   // primera vez (placeholder)
hash := sha256.Sum256(data)
inv.CollectorHash = hashStr
data, _ = json.MarshalIndent(inv, "", "  ")   // segunda vez (hash real)
```

La solución correcta es serializar una vez con el hash calculado sobre el
contenido, no modificar el struct entre serializaciones. Esto también tiene el
efecto secundario de que el hash no incluye el campo `collector_hash` (se calcula
antes de incluirlo).

**Propuesta:** Separar el contenido del hash:

```go
body, _ := json.MarshalIndent(inv, "", "  ")
hash := sha256.Sum256(body)
inv.CollectorHash = hex.EncodeToString(hash[:])
body, _ = json.MarshalIndent(inv, "", "  ")  // una sola re-serialización
```

O mejor: no incluir `collector_hash` dentro del JSON que se hashea (calcular
hash antes de asignar el campo, como ya se hace — está bien).

---

### L3. collectorHash no cubre todos los campos

**Dónde:** `output.go`

**Problema:** El hash SHA-256 se calcula sobre `json.MarshalIndent(inv, "", "  ")`.
Esto incluye el `CollectorHash = "pending"` en la primera iteración. El hash
resultante cubre el contenido PERO incluye un placeholder, no el hash final.

El flujo correcto debería ser:
1. Serializar todo EXCEPTO `collector_hash`
2. Calcular hash sobre ese contenido
3. Asignar hash al campo
4. Re-serializar completo

**Propuesta:** Marcar `CollectorHash` con `json:"-"` y exponerlo aparte, o
calcular el hash sobre un subconjunto de campos.

---

### L4. 4 wrappers de PowerShell que hacen casi lo mismo

**Dónde:** `collector_windows.go:317-348`

Ya cubierto en K1. Es duplicación LEAN porque podrían ser **una** función con
composición. 4 × 10 líneas = 40 líneas que se reducen a ~15.

---

### L5. readFile en Linux, no en Windows ni macOS

**Dónde:** `collector_linux.go:33` (`readFile`)

**Problema:** Linux usa `os.ReadFile` envuelto. macOS no lo necesita (usa comandos).
Pero si alguien en el futuro agrega lectura directa en Windows o macOS, va a
re-implementar lo mismo.

**Propuesta:** Mover `readFile` a `utils.go` como función compartida, con manejo
de errores mejorado (log + string vacío en vez de silencio total).

---

## 5. SOLID — Violaciones

### S1. Single Responsibility: collector hace demasiado

**Dónde:** `pkg/collector/`

**Paquete único que hace:**
- Recolectar datos del sistema (9 categorías × 3 plataformas)
- Parsear EDID binario
- Parsear output de comandos
- Serializar JSON
- Calcular hashes
- Escribir archivos a disco
- Leer archivos del disco (output.go: `ReadFromFile`)

**Propuesta:** Separar en sub-paquetes:

```
pkg/
├── collector/           # Orchestrador (bajo build tag)
│   ├── collect.go       # Run() con inyección de dependencias
│   ├── collect_linux.go
│   ├── collect_darwin.go
│   └── collect_windows.go
├── inventory/           # Modelo de datos (sin build tag)
│   ├── types.go         # Inventory, SystemInfo, CPUInfo, etc.
│   └── validate.go      # Validación de campos obligatorios
├── output/              # Serialización y escritura
│   ├── json.go           # JSON output + hash
│   └── json_test.go
├── sysinfo/             # Lógica de obtención por SO
│   ├── cpu.go           # getCPU() compartido
│   ├── cpu_linux.go     # Implementación Linux
│   ├── cpu_darwin.go
│   ├── cpu_windows.go
│   ├── disk.go
│   ├── disk_linux.go
│   ├── ...
│   └── executor.go      # CommandRunner interface
└── testutil/            # Helpers de test cross-platform
    ├── fakes.go          # FakeFS, FakeCommandRunner
    └── fixtures.go       # Carga de fixtures
```

---

### S2. Open-Closed: imposible extender sin tocar collector

**Dónde:** `collector_*_test.go`

**Problema:** No hay interfaces para las funciones de colección. Agregar una
categoría nueva (ej: batería, temperatura, TPM) requiere:
1. Agregar el tipo en `types.go`
2. Agregar `getBattery() BatteryInfo` en 3 archivos
3. Llamarla desde `Run()` en 3 archivos
4. Agregar output en `output.go`

Con interfaces, sería:

```go
type Collector interface {
    Name() string
    Collect(ctx context.Context, runner CommandRunner) (interface{}, error)
}
```

Y registrar collectors por plataforma. Nuevos collectors se agregan sin tocar
el código existente.

---

### S3. Liskov: Windows devuelve error, Linux y macOS no

**Dónde:** Firma de `Run()` en las 3 plataformas.

**Firmas actuales:**

| Plataforma | Firma | ¿Retorna error real? |
|------------|-------|---------------------|
| Linux | `Run() (*Inventory, error)` | ❌ Siempre nil |
| macOS | `Run() (*Inventory, error)` | ❌ Siempre nil |
| Windows | `Run() (*Inventory, error)` | ✅ Errores parciales |

**Problema:** La interfaz promete `error` pero Linux y macOS nunca lo usan.
Windows lo usa para errores parciales. Un consumidor del API no sabe si confiar
en `err` o no. Es una violación de Liskov porque los subtipos no cumplen el
contrato de la misma forma.

**Propuesta:** Unificar el comportamiento. O todas devuelven error cuando algo
falla, o ninguna. Recomiendo que todas acumulen errores parciales como Windows:

```go
type PartialErrors []string
func (e PartialErrors) Error() string { return fmt.Sprintf("%d errores parciales", len(e)) }
func (e *PartialErrors) Add(format string, args ...interface{}) { ... }
```

---

### S4. Interface Segregation: CommandRunner mezcla comandos y JSON

**Dónde:** Si se implementa `CommandRunner` como se propuso en D4.

**Propuesta:** Segregar:

```go
type Commander interface {
    Run(name string, args ...string) (string, error)
}

type JSONProvider interface {
    RunJSON(cmd string, dest interface{}) error
}
```

Así no todos los collectors necesitan el método `RunJSON`. Un collector de CPU
usa `Commander`, solo los que parsean JSON usan `JSONProvider`.

---

### S5. Dependency Inversion: collector depende directamente de os/exec

**Dónde:** Todos los `exec.Command` en las 3 plataformas.

**Problema:** `collector.Run()` crea sus propios comandos. No hay forma de
testear la colección real sin ejecutar comandos reales. La dependencia concreta
`os/exec` está acoplada directamente en la lógica de negocio.

**Propuesta:** `Run()` recibe un `CommandRunner` externo. En producción se pasa
`OSCommandRunner{}`, en test se pasa un `FakeCommandRunner` que devuelve data
precargada desde fixtures.

---

## 6. Archivos afectados por hallazgo

| Archivo | Líneas | Hallazgos |
|---------|--------|-----------|
| `collector_linux.go` | 356 | D1, D2, D3, D4, K4, K5, K6, S3 |
| `collector_windows.go` | 417 | D1, D3, D4, K1, K2, K6, S3 |
| `collector_darwin.go` | 332 | D1, D2, D3, D4, K3, K4, K6, S3 |
| `output.go` | 97 | L2, L3 |
| `types.go` | 103 | S2 (faltan interfaces) |
| `utils.go` | 21 | D1, D5 (parcial) |
| `utils_test.go` | 25 | L1 (poco) |
| `collector_*_test.go` | 166 total | L1 (solo helpers) |
| `cmd/iodat/main.go` | 67 | K7 |

---

## 7. Propuesta de estructura futura

```
iodat/
├── cmd/iodat/main.go              # Punto de entrada (~50 líneas)
├── pkg/
│   ├── inventory/
│   │   ├── types.go               # Tipos de datos puros
│   │   ├── validate.go            # Validación cross-platform
│   │   └── types_test.go
│   ├── collector/
│   │   ├── run.go                 # Run(inv *inventory.Inventory, runner CommandRunner)
│   │   ├── run_linux.go
│   │   ├── run_darwin.go
│   │   └── run_windows.go
│   ├── sysinfo/
│   │   ├── executor.go            # CommandRunner interface
│   │   ├── executor_prod.go       # OSCommandRunner real
│   │   ├── executor_test.go
│   │   ├── cpu/
│   │   │   ├── cpu.go             # getCPU() shared signature
│   │   │   ├── cpu_linux.go
│   │   │   ├── cpu_darwin.go
│   │   │   └── cpu_windows.go
│   │   ├── disk/
│   │   ├── ram/
│   │   ├── network/
│   │   ├── gpu/
│   │   ├── monitor/
│   │   └── software/              # Solo Windows
│   ├── output/
│   │   ├── json.go                # JSON output + hash
│   │   └── json_test.go
│   └── testutil/
│       ├── fakesys.go             # FakeCommandRunner
│       ├── fixtures.go            # Carga de fixtures desde testdata/
│       └── testdata/              # Datos simulados por SO
│           ├── linux/
│           ├── darwin/
│           └── windows/
├── docs/autocritica-*.md
└── Makefile
```

---

## 8. Recomendaciones de prioridad

| Prioridad | Ref | Esfuerzo | Impacto | Tarea |
|-----------|-----|----------|---------|-------|
| 🔴 P0 | D4 + S5 | 1 día | 🔥 Testability | Abstraer `CommandRunner` con interface |
| 🔴 P0 | K6 | 1 día | 🔥 Estabilidad | Agregar timeouts a exec.Command |
| 🔴 P0 | L1 | 2 días | 🔥 Calidad | Tests de integración con fixtures |
| 🟡 P1 | D1 | 0.5 día | DRY | Unificar helpers de parseo en utils.go |
| 🟡 P1 | S1 | 1 día | SOLID | Separar en sub-paquetes (inventory, output) |
| 🟡 P1 | K1 | 0.5 día | KISS | Simplificar wrappers de PowerShell |
| 🟡 P1 | S3 | 0.5 día | SOLID | Unificar error handling en Run() |
| 🟢 P2 | D2 | 0.5 día | DRY | Unificar parseo de tamaños a ByteSize |
| 🟢 P2 | K2 | 1 día | KISS | Migrar de Get-WmiObject a Get-CimInstance |
| 🟢 P2 | K3 | 0.5 día | KISS | Eliminar bash -c en macOS |
| 🟢 P2 | D3 | 0.25 día | DRY | Constante compartida para loopback |
| 🔵 P3 | S2 | 2 días | SOLID | Sistema de plugins para nuevos collectors |
| 🔵 P3 | K5 | 1 día | KISS | EDID parsing con validación |
| 🔵 P3 | K7 | 0.5 día | KISS | Migrar a flag.Parse() |
| 🔵 P3 | L2/L3 | 0.5 día | LEAN | Hash correcto del JSON |
| 🔵 P3 | K4 | 0.5 día | KISS | macOS network sin regex |
| ⚪ P4 | S2 (full) | 3 días | SOLID | Sistema completo de Collector interface |

### Dependencias entre tareas

```
D4 + S5 (CommandRunner) ←── bloquea ──→ L1 (tests)
       ↓
  S1 (reorganización) ←── pueden ser paralelas con ──→ D1, K1, K6
       ↓
  S2 (sistema de plugins) ←── requiere ──→ S1 completo
  K2, K3, K4, K5, K7      ←── independientes entre sí
```

### Recomendación de secuencia

1. **Worktree 1:** D4 + S5 (CommandRunner + timeouts) — base para todo lo demás
2. **Worktree 2:** D1 + D2 (helpers unificados) — paralelo al anterior
3. **Worktree 3:** K1 + K2 (PowerShell) — independiente
4. **Worktree 4:** K3 + K4 (macOS) — independiente
5. **Worktree 5:** L2 + L3 (hash) + K7 (flags) — independiente
6. **Worktree 6:** S1 + S2 + S3 (arquitectura) — después de 1-5
7. **Worktree 7:** L1 (tests con fixtures) — después de 1-5

> Cada worktree puede crear una rama `refactor/<nombre>` desde `develop`,
> trabajarse en paralelo y mergearse de vuelta.
