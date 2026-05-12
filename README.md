# ioDat — Colector de inventario para ioDesk-3

Herramienta **multiplataforma** para el levantamiento técnico de equipos informáticos.
Genera un JSON listo para importar en ioDesk-3.

**Cero dependencias.** Binario autocontenido. No requiere instalación, red ni permisos de administrador.

```bash
./iodat --stdout                    # Ver resultado en pantalla
./iodat --output ./Levantamientos   # Guardar JSON en carpeta
```

---

## Requisitos

| SO | Versión | Dependencias |
|----|---------|-------------|
| Windows | 10 / Server 2016 | PowerShell 5.1+ |
| Linux | Kernel 4.x+ | `lspci` (opcional) |
| macOS | 12 Monterey+ | `system_profiler` |

---

## Security & Compliance

Diseñado para entornos regulados: **Ley 19.628**, **NCG 461**, **ISO 27001**, **PCI-DSS**, **HIPAA**.

| Principio | Implementación |
|-----------|---------------|
| **Zero network** | Sin conexiones de red |
| **Read-only** | Solo lee `/proc`, `/sys`, WMI, `system_profiler` |
| **No shell eval** | PowerShell con `-NoProfile`, sin `eval()` |
| **Least privilege** | Sin permisos de administrador |
| **Zero dependencies** | Binario Go estático |
| **Sin telemetría** | No envía datos a ningún lado |

**Datos recolectados:** hostname, serial, MAC, IP, CPU, RAM, discos, GPU, monitores, red, software (top 200, solo Windows).  
**Ningún dato personal** según Ley 19.628.

<details>
<summary><strong>Verificación independiente</strong> — comandos para auditores</summary>

```bash
# Sin dependencias externas
go version -m iodat | grep dep

# Sin conexiones de red
grep -r 'http\.\|net\.Dial\|tls\.' pkg/ cmd/

# Solo lee archivos del sistema (rutas fijas)
grep -rn 'os\.ReadFile\|os\.Open\|exec\.Command' pkg/collector/

# Verificar hash del binario
sha256sum iodat-v1.0.0-linux-amd64
```

</details>

<details>
<summary><strong>Datos recolectados por SO</strong> — tabla completa</summary>

| Categoría | Windows | Linux | macOS |
|-----------|---------|-------|-------|
| Sistema | WMI | /sys/class/dmi | system_profiler |
| CPU | WMI | /proc/cpuinfo | sysctl |
| RAM | WMI | /proc/meminfo | sysctl + system_profiler |
| Almacenamiento | WMI | /sys/block | system_profiler |
| Placa madre | WMI | /sys/class/dmi | sysctl |
| GPU | WMI | lspci + /sys/class/drm | system_profiler |
| Monitores | WMI | EDID (/sys/class/drm) | system_profiler |
| Red | WMI | /sys/class/net | ifconfig |
| Software | Registry | — | — |

</details>

<details>
<summary><strong>Instalación</strong> — descargar y verificar</summary>

Descargar desde el [repositorio](https://github.com/iodevs-net/iodat).

```bash
# Verificar integridad
sha256sum -c SHA256SUMS 2>&1 | grep OK
```

</details>

<details>
<summary><strong>Compatibilidad con ioDesk-3</strong></summary>

1. Ejecutar `iodat` en el equipo del cliente
2. ioDesk-3 → Inventario → Cargar datos
3. Subir el archivo `.json` generado

</details>

<details>
<summary><strong>Desarrollo</strong></summary>

```bash
make build    # Compilar plataforma actual
make all      # Compilar Windows + Linux + macOS
make test     # Tests unitarios
make verify   # SHA256SUMS de todos los binarios
make sbom     # Software Bill of Materials
```

</details>

<details>
<summary><strong>Reporte de vulnerabilidades</strong></summary>

Ver [SECURITY.md](SECURITY.md). Correo: **soporte@ionet.cl**.  
No abrir issues públicos para vulnerabilidades de seguridad.

</details>

---

## Licencia

**MIT** — [LICENSE](LICENSE). Usa, modifica y distribuye libremente.

## Créditos

**Leonardo Vergara** — [lvergara@iodevs.net](mailto:lvergara@iodevs.net)
— [iodevs.net](https://iodevs.net)  
para **IONET Ltda.** — [ionet.cl](https://ionet.cl)
