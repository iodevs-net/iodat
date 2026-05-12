# ioDat — Colector de inventario para ioDesk-3

Herramienta multiplataforma de línea de comandos para el levantamiento técnico de equipos informáticos. Genera un archivo JSON estructurado con las especificaciones del equipo, listo para importar en ioDesk-3.

**Aprobado para entornos regulados:** estudios jurídicos, banca, AGF, bolsa de valores, salud y gobierno. Ver [Security & Compliance](#security--compliance).

---

## Requisitos

| SO | Versión mínima | Dependencias |
|----|---------------|-------------|
| Windows | 10 / Server 2016 | PowerShell 5.1+ (incluido) |
| Linux | Kernel 4.x+ | `lspci` (pciutils, opcional) |
| macOS | 12 Monterey+ | `system_profiler` (incluido) |

**Cero dependencias externas.** El binario es autocontenido. No requiere runtime, framework ni paquetes de terceros. Compila contra la biblioteca estándar de Go 1.26.

---

## Instalación

Descargar el binario para tu plataforma desde la [página de releases](https://github.com/ionet-cl/iodat/releases).

Cada release incluye un archivo `SHA256SUMS` con los hashes de todos los binarios. Verificar integridad:

```bash
# Linux / macOS
sha256sum -c SHA256SUMS 2>&1 | grep OK

# Windows (PowerShell)
Get-FileHash iodat-v1.0.0-windows-amd64.exe -Algorithm SHA256
# Comparar contra SHA256SUMS
```

No requiere instalación ni permisos de administrador. Ejecutar directamente.

---

## Uso

```bash
# Generar archivo JSON en el directorio actual
./iodat

# Imprimir a stdout (útil para piping)
./iodat --stdout

# Guardar en un directorio específico
./iodat /ruta/destino/

# Mostrar ayuda
./iodat --help
```

### Ejemplo de salida

```json
{
  "collector_version": "1.0.0",
  "collector_hash": "e3b0c44298fc1c149afbf4c8996fb924...",
  "hostname": "PC-CONTABILIDAD",
  "system": {
    "manufacturer": "Dell",
    "model": "OptiPlex 7080",
    "serial_number": "ABC123",
    "os": "Microsoft Windows 11 Pro",
    "os_version": "10.0.22631",
    "os_architecture": "64-bit"
  },
  "cpu": {
    "name": "Intel Core i7-10700",
    "name_clean": "Intel Core i7-10700",
    "cores": 8,
    "logical_processors": 16,
    "max_clock_mhz": 2900
  },
  "ram": {
    "total_gb": 16,
    "formatted": "16GB",
    "slots": [
      {"bank_label": "DIMM A1", "size_gb": 8, "speed_mhz": 3200, "type": "DDR4"}
    ]
  }
}
```

---

## Security & Compliance

ioDat está diseñado para operar en entornos con requisitos regulatorios estrictos: **Ley 19.628** (protección de datos personales, Chile), **NCG 461** (ciberseguridad CMF), **ISO 27001**, **PCI-DSS**, **HIPAA**.

### Principios de diseño seguro

| Principio | Implementación |
|-----------|---------------|
| **Zero network** | No realiza conexiones de red salientes ni entrantes. No depende de servicios externos. |
| **Read-only** | Solo lee archivos del sistema (`/proc`, `/sys`, WMI, `system_profiler`). Jamás escribe fuera del directorio de salida. |
| **No shell eval** | PowerShell se invoca con `-NoProfile`. Sin `eval()`, sin `exec()`, sin interpretación dinámica de código. |
| **Least privilege** | No requiere permisos de administrador/root. Funciona con permisos de usuario estándar. |
| **Zero dependencies** | Binario estático autocontenido. Sin DLLs, sin runtimes, sin paquetes npm/pip/gems. |
| **Integridad verificable** | Cada release incluye `SHA256SUMS`. El JSON generado incluye un hash SHA-256 del contenido. |
| **Código auditable** | ~800 líneas de Go estándar. Repositorio público. Sin ofuscación. |
| **Sin telemetría** | No recolecta estadísticas de uso, no envía datos a IONET ni a terceros. |

### Datos recolectados y su clasificación

| Dato | Clasificación | Fundamento |
|------|--------------|------------|
| Hostname | Identificador técnico | Nombre de red del equipo |
| N° de serie | Identificador de hardware | Grabado por el fabricante en BIOS/EFI |
| MAC address | Identificador de red | Visible en cualquier red local |
| IP address | Datos de red | Dirección actual del equipo |
| Software instalado | Inventario de activos | Top 200 aplicaciones (solo Windows) |
| CPU, RAM, discos | Especificaciones técnicas | Sin información personal |

**Ningún dato recolectado constituye dato personal** según la definición de la Ley 19.628. ioDat no accede a: documentos del usuario, historial de navegación, contraseñas, cookies, correos electrónicos, archivos personales, ni claves de registro fuera de `HKLM\Software\...\Uninstall`.

### Verificación independiente

Cualquier equipo de seguridad o compliance puede verificar estas afirmaciones:

```bash
# 1. Verificar que no hay dependencias externas
go version -m iodat | grep dep  # vacío

# 2. Verificar que no hay llamadas de red en el código fuente
grep -r 'http\.' pkg/ cmd/       # vacío
grep -r 'net\.Dial' pkg/ cmd/    # vacío
grep -r 'tls\.' pkg/ cmd/        # vacío

# 3. Verificar que solo lee archivos del sistema
grep -rn 'os\.ReadFile\|os\.Open\|exec\.Command' pkg/collector/
# Solo rutas bajo /proc, /sys, system_profiler, powershell

# 4. Verificar hash del binario descargado
sha256sum iodat-v1.0.0-linux-amd64
# Debe coincidir con SHA256SUMS del release
```

---

## Datos recolectados

| Categoría | Campos | Windows | Linux | macOS |
|-----------|--------|---------|-------|-------|
| Sistema | Fabricante, modelo, serial, OS | WMI | /sys/class/dmi | system_profiler |
| CPU | Modelo, cores, frecuencia | WMI | /proc/cpuinfo | sysctl |
| RAM | Total y slots individuales | WMI | /proc/meminfo | sysctl + system_profiler |
| Almacenamiento | Discos, tamaño, tipo (SSD/HDD) | WMI | /sys/block | system_profiler |
| Placa madre | Fabricante, BIOS | WMI | /sys/class/dmi | sysctl |
| GPU | Modelo, VRAM, driver | WMI | lspci + /sys/class/drm | system_profiler |
| Monitores | Modelo, resolución | WMI | EDID (/sys/class/drm) | system_profiler |
| Red | MAC, IP, velocidad | WMI | /sys/class/net | ifconfig |
| Software | Programas instalados (top 200) | Registry Uninstall | — | — |

---

## Compatibilidad con ioDesk-3

El JSON generado es compatible con la vista `InventarioImportView` de ioDesk-3.

1. Ejecutar `iodat` en el equipo del cliente
2. En ioDesk-3: Cliente → Inventario → Cargar datos
3. Subir el archivo `.json` generado

---

## Desarrollo

```bash
make build    # Compilar plataforma actual
make all      # Compilar Windows + Linux + macOS (amd64/arm64)
make test     # Ejecutar tests unitarios
make verify   # Generar SHA256SUMS de todos los binarios
make sbom     # Listar dependencias (Software Bill of Materials)
make run      # Ejecutar localmente
```

---

## Licencia

ioDat se distribuye bajo licencia **MIT**. Ver [LICENSE](LICENSE).
Eres libre de usar, modificar y distribuir este software
siempre que mantengas el aviso de copyright original.

---

## Créditos

Desarrollado por **Leonardo Vergara**
  — [lvergara@iodevs.net](mailto:lvergara@iodevs.net)
  — [leonardovergaramarin@gmail.com](mailto:leonardovergaramarin@gmail.com)
  — [iodevs.net](https://iodevs.net)
para **IONET Ltda.** — [ionet.cl](https://ionet.cl)

---

## Reporte de vulnerabilidades

Ver [SECURITY.md](SECURITY.md). Vulnerabilidades críticas: correo a soporte@ionet.cl. No abrir issue público para vulnerabilidades de seguridad.
