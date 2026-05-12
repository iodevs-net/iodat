package collector

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ionet-cl/iodat/pkg/inventory"
)

// cleanCPUName elimina marcas registradas, sufijos de velocidad y la palabra "CPU"
// del nombre del procesador, dejando solo el nombre comercial limpio.
func cleanCPUName(name string) string {
	if name == "" {
		return ""
	}
	// (R), (r), (TM), (tm)
	name = regexp.MustCompile(`\s*\([Rr]\)`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`\s*\([Tt][Mm]\)`).ReplaceAllString(name, "")
	// " CPU " o "CPU " o " CPU"
	name = regexp.MustCompile(`\s*CPU\s*`).ReplaceAllString(name, " ")
	// "@ 2.90GHz"
	name = regexp.MustCompile(`\s+@.*`).ReplaceAllString(name, "")
	// Colapsar espacios múltiples
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(name, " "))
}

// ── Parsers numéricos (unificados) ───────────────

// ParseInt parses a string to int. Returns 0 on failure.
func ParseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// ParseInt64 parses a string to int64. Returns 0 on failure.
func ParseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// ParseFloat64 parses a string to float64. Returns 0 on failure.
func ParseFloat64(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// ── Conversiones de tamaño a ByteSize ────────────

// FromBlocks convierte un contador de bloques de 512 bytes (como los que
// reporta /sys/block/*/size en Linux) a ByteSize.
func FromBlocks(blocks int64) inventory.ByteSize {
	return inventory.ByteSize(blocks) * 512
}

// FromBytes crea un ByteSize a partir de un entero de bytes en crudo.
func FromBytes(bytes int64) inventory.ByteSize {
	return inventory.ByteSize(bytes)
}

// ParseByteSize parsea strings con unidad como "500.24 GB", "1 TB", "256 MB"
// y retorna un ByteSize. Soporta GB, TB, MB, KB.
func ParseByteSize(s string) (inventory.ByteSize, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}
	re := regexp.MustCompile(`([\d.]+)\s*(TB|GB|KB|MB)`)
	m := re.FindStringSubmatch(s)
	if len(m) < 3 {
		return 0, fmt.Errorf("cannot parse size: %q", s)
	}
	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in %q: %w", s, err)
	}
	switch m[2] {
	case "TB":
		return inventory.ByteSize(val * float64(inventory.TB)), nil
	case "GB":
		return inventory.ByteSize(val * float64(inventory.GB)), nil
	case "MB":
		return inventory.ByteSize(val * float64(inventory.MB)), nil
	case "KB":
		return inventory.ByteSize(val * float64(inventory.KB)), nil
	}
	return 0, fmt.Errorf("unknown unit: %s", m[2])
}

// MustGB parses a size string and returns the integer GB value.
// Returns 0 if the string cannot be parsed.
// Convenience helper for callers that don't need error handling.
func MustGB(s string) int {
	bs, err := ParseByteSize(s)
	if err != nil {
		return 0
	}
	return bs.GB()
}
