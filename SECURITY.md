# Política de Seguridad — ioDat

## Versiones soportadas

| Versión | Soporte |
|---------|---------|
| 1.x.x   | Activo (actual) |

Solo la versión más reciente recibe parches de seguridad.

---

## Postura de seguridad

ioDat está diseñado bajo el principio de **mínimo privilegio absoluto**:

1. **No realiza conexiones de red** de ningún tipo (TCP, UDP, HTTP, DNS, WebSocket, ICMP).
2. **No escribe archivos** fuera del directorio de salida especificado por el usuario.
3. **No ejecuta código dinámico**: sin `eval()`, sin `exec()`, sin carga de plugins.
4. **PowerShell con `-NoProfile`**: evita ejecución de scripts de perfil potencialmente maliciosos.
5. **Cero dependencias externas**: binario Go estático, sin DLLs, sin runtimes.

### Superficie de ataque

| Vector | Riesgo | Mitigación |
|--------|--------|------------|
| Ejecución de comandos del sistema | Bajo | Solo binarios nativos del SO (`lspci`, `system_profiler`, `powershell.exe`). Sin interpolación de strings del usuario. |
| Lectura de archivos | Nulo | Solo rutas fijas documentadas (`/proc`, `/sys`, `/sys/class`, WMI). Sin path traversal. |
| Desbordamiento de buffer | Nulo | Go es memory-safe. Sin `unsafe`, sin CGo. |
| Race conditions | Nulo | Código secuencial puro. Sin goroutines, sin concurrencia. |
| Inyección de código | Nulo | Sin interpretación de input del usuario. Args solo controlan flags y directorio de salida. |
| Suplantación de binario | Bajo | SHA256SUMS firmados en cada release. Verificación recomendada antes de ejecutar. |

---

## Reporte de vulnerabilidades

Si descubres una vulnerabilidad de seguridad en ioDat:

1. **No abras un issue público.**
2. Envía un correo a **soporte@ionet.cl** con:
   - Descripción detallada del hallazgo
   - Pasos para reproducir
   - Impacto potencial
   - Sugerencia de mitigación (opcional)


### Tiempos de respuesta

| Severidad | Respuesta inicial | Parche |
|-----------|------------------|--------|
| Crítica | 24 horas | 72 horas |
| Alta | 48 horas | 7 días |
| Media | 1 semana | 30 días |
| Baja | 2 semanas | Próximo release |

### Proceso de divulgación

1. El reporte se acusa recibo en 24 horas.
2. IONET investiga y reproduce.
3. Se desarrolla un parche en rama privada.
4. Se notifica al reportante para validar.
5. Se publica el fix con CVE (si aplica) y crédito al reportante.

---

## Verificación de integridad

Cada release de ioDat incluye:

- **SHA256SUMS**: hashes SHA-256 de todos los binarios
- **Código fuente auditado**: tag de git firmado

```bash
# Verificar release
curl -LO https://github.com/ionet-cl/iodat/releases/download/v1.0.0/SHA256SUMS
curl -LO https://github.com/ionet-cl/iodat/releases/download/v1.0.0/iodat-v1.0.0-linux-amd64
sha256sum -c SHA256SUMS 2>&1 | grep iodat

# Verificar código fuente
git clone https://github.com/ionet-cl/iodat.git
cd iodat
git verify-tag v1.0.0
```
