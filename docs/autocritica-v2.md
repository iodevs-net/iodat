# ioDat — Lo que duele (post-refactor)

> Documento generado el 2026-05-12 después del refactor DRY+KISS+LEAN+SOLID.
> 7 worktrees ejecutados, 7 commits en develop.
> Esto es lo que queda sin resolver.

---

## 1. Solo Linux tiene tests de integración

**Problema:** Los tests de `getGPU`, `getNetwork`, `getSystemInfo` con `FakeCommandRunner` solo corren en Linux porque los archivos `*_test.go` tienen build tags (`//go:build linux`). macOS y Windows no tienen un solo test de integración.

**Riesgo:** Un cambio en el parseo de `system_profiler` en macOS o de WMI en Windows no se detecta hasta que alguien ejecuta el binario en esa plataforma.

**Propuesta:**

Agregar fixtures cross-platform y tests que corran en cualquier SO:

```
pkg/collector/testdata/
├── darwin/
│   ├── system_profiler_SPHardwareDataType.json
│   ├── system_profiler_SPMemoryDataType.json
│   ├── system_profiler_SPStorageDataType.json
│   ├── system_profiler_SPDisplaysDataType.json
│   └── sysctl.txt
├── windows/
│   ├── powershell_ComputerSystem.json
│   ├── powershell_Processor.json
│   ├── powershell_PhysicalMemory.json
│   ├── powershell_DiskDrive.json
│   ├── powershell_VideoController.json
│   └── powershell_Uninstall.json
└── linux/
    ├── lspci_mm.txt
    └── ip_addr_eno1.json   ← ya existe
```

Los tests deben usar `FakeCommandRunner` con estas fixtures para verificar el parsing.
Se ejecutan manualmente en cada SO o via CI matrix.

**Esfuerzo:** 1-2 días por SO.
**Prioridad:** Alta — sin esto, la confianza en macOS/Windows es nula.

---

## 2. Las lecturas de archivo no están abstraídas

**Problema:** `getCPU()`, `getRAM()`, `getMotherboard()`, `getMonitors()` en Linux leen archivos directamente con `os.ReadFile()` -> `readFile()`. No hay forma de mockear `/proc/cpuinfo`, `/proc/meminfo`, `/sys/class/dmi/id/*` en tests.

**Impacto:** 4 de 8 funciones del collector Linux no tienen tests unitarios reales. Dependen del hardware de la máquina donde corre el test.

**Propuesta:**

```go
// En executor.go o un nuevo archivo:
type FileSystem interface {
    ReadFile(path string) (string, error)
    ReadDir(path string) ([]os.DirEntry, error)
}

type OSFileSystem struct{}
func (OSFileSystem) ReadFile(path string) (string, error) {
    data, err := os.ReadFile(path)
    return string(data), err
}
```

Y en `collector_linux.go`:

```go
func Run(runner CommandRunner, fs FileSystem) (*Inventory, error)
```

Las funciones `getCPU`, `getRAM`, etc. reciben `fs` y leen archivos a través de la interfaz.

**Esfuerzo:** 1 día (similar a CommandRunner).
**Prioridad:** Media-alta — sin esto, los tests de Linux quedan cojos.

---

## 3. collector es un monolito de ~1850 líneas

**Problema:** El paquete `pkg/collector/` contiene toda la lógica: tipos, colección, output, parsing, ejecución de comandos. No hay separación por responsabilidad.

**Impacto:** A medida que crece, es difícil encontrar código, las importaciones circulares son probables, y el paquete pierde cohesión.

**Propuesta (S1 original):**

```
pkg/
├── inventory/       # Tipos de datos puros
├── collector/       # Orquestación (Run, build tags)
├── output/          # JSON + hash
├── sysinfo/         # Lógica por SO (cpu, disk, ram...)
└── testutil/        # Fakes + fixtures
```

Detalle completo en el PRD original (sección S1).

**Esfuerzo:** 2-3 días. Requiere mover archivos, actualizar imports, verificar que los build tags sigan funcionando.
**Prioridad:** Media — relevante si se agregan nuevas categorías (batería, TPM, temperatura).

---

## 4. EDID parsing en Linux es artesanal

**Problema:** `getMonitors()` en Linux extrae bytes sueltos del bloque EDID sin validar checksum:

```go
mf := string([]byte{edid[8] + 'A' - 1, edid[9] + 'A' - 1, edid[10] + 'A' - 1})
```

**Riesgo:** EDID corrupto o extendido (>128 bytes, común en monitores 4K/HDR) produce datos basura o panic por index out of bounds.

**Propuesta:**

Validar antes de parsear:
1. Verificar que `len(edid) >= 128`
2. Verificar checksum del bloque 0 (suma de bytes 0-127 debe ser 0 mod 256)
3. Opcional: parsear bloque EDID extendido si existe

```go
func validateEDID(edid []byte) bool {
    if len(edid) < 128 {
        return false
    }
    var sum int
    for _, b := range edid[:128] {
        sum += int(b)
    }
    return sum%256 == 0
}
```

**Esfuerzo:** 0.5 días.
**Prioridad:** Baja — solo afecta monitores 4K/HDR en Linux.

---

## 5. Sin validación cross-platform real

**Problema:** No se ha ejecutado el binario compilado en macOS ni Windows después del refactor. Los cambios en `collector_darwin.go` y `collector_windows.go` son correctos estructuralmente, pero nadie los ha probado contra un sistema real.

**Riesgo:** Un detalle como `ifconfig -l` no existir en alguna versión de macOS, o `ConvertTo-Json -Depth 2` truncar datos en PowerShell 5.1, pasaría desapercibido.

**Propuesta:**

| Plataforma | Qué probar | Quién |
|-----------|-----------|-------|
| macOS 14+ | `go build`, `./iodat --stdout`, verificar JSON completo | Developer con Mac |
| macOS 12-13 | Igual (versiones anteriores) | Developer con Mac antiguo |
| Windows 10 | `go build`, `./iodat --stdout` en PowerShell | Developer con Windows |
| Windows 11 | Igual | Developer con Windows |
| Windows Server Core | `./iodat --stdout` (sin PowerShell completo) | DevOps |

**Esfuerzo:** 1 hora por plataforma si se tiene acceso.
**Prioridad:** Alta — sin esto, los cambios en macOS/Windows son teóricos.

---

## Prioridades

| Prioridad | Tema | Esfuerzo | Por qué |
|-----------|------|----------|---------|
| 🔴 P0 | Tests en macOS y Windows (#1) | 1-2 días | Sin tests no hay confianza |
| 🔴 P0 | Validación cross-platform real (#5) | ~3 horas | Lo teórico no basta |
| 🟡 P1 | Abstraer FileSystem (#2) | 1 día | Tests de Linux cojos |
| 🟢 P2 | Separar collector en paquetes (#3) | 2-3 días | Prevención, no urgencia |
| 🔵 P3 | EDID con checksum (#4) | 0.5 días | Caso borde poco común |

---

## Resumen

El refactor dejó ioDat en un estado sólido pero incompleto:

- **Lo que duele de verdad:** no sabemos si macOS y Windows siguen funcionando. Los tests solo corren en Linux.
- **Lo que duele a medias:** las lecturas de archivo no se pueden mockear. Si alguien cambia `/proc/cpuinfo`, los tests no lo detectan.
- **Lo que no duele hoy:** el monolito de 1850 líneas y el EDID artesanal. Dolerán cuando el proyecto crezca.

La prioridad uno es **probar en macOS y Windows**, ya sea con tests automatizados o ejecución manual. Sin eso, el refactor en esas plataformas es humo.
