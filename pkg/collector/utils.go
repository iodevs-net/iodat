package collector

import "regexp"
import "strings"

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
