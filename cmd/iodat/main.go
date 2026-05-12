// ioDat — Colector de inventario técnico para ioDesk-3 (IONET).
// Genera un archivo JSON con las especificaciones del equipo
// para ser importado en ioDesk-3.
//
// Uso:
//   iodat                    → genera ioDat_<hostname>_<fecha>.json
//   iodat --stdout           → imprime JSON a stdout
//   iodat --output /ruta     → guarda en el directorio especificado
//   iodat --version          → muestra la versión

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ionet-cl/iodat/pkg/collector"
	"github.com/ionet-cl/iodat/pkg/output"
)

// version se inyecta en tiempo de compilación via ldflags.
// Ejemplo: go build -ldflags="-X main.version=1.0.0"
var version = "dev"

func main() {
	// Flags
	toStdout := flag.Bool("stdout", false, "Imprime JSON a stdout en vez de archivo")
	outputDir := flag.String("output", "", "Directorio donde guardar el archivo JSON")
	showVersion := flag.Bool("version", false, "Muestra la versión del binario")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ioDat v%s — Colector de inventario para ioDesk-3\n", version)
		fmt.Fprintf(os.Stderr, "\nUso:\n")
		fmt.Fprintf(os.Stderr, "  iodat [flags]\n")
		fmt.Fprintf(os.Stderr, "  iodat [flags] /ruta/directorio\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEjemplos:\n")
		fmt.Fprintf(os.Stderr, "  iodat                   Genera archivo en el directorio actual\n")
		fmt.Fprintf(os.Stderr, "  iodat --stdout          Imprime JSON a stdout\n")
		fmt.Fprintf(os.Stderr, "  iodat --output /tmp     Guarda archivo en /tmp\n")
		fmt.Fprintf(os.Stderr, "  iodat --version         Muestra la versión\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("ioDat v%s\n", version)
		os.Exit(0)
	}

	// Argumento posicional: directorio de salida
	dir := *outputDir
	if dir == "" && flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	// Validar directorio si se especificó
	if dir != "" {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: %q no es un directorio válido\n", dir)
			os.Exit(1)
		}
	}

	// Recolectar
	fmt.Fprintf(os.Stderr, "ioDat: Recolectando información del equipo...\n")
	inv, err := collector.Run(collector.OSCommandRunner{}, collector.OSFileSystem{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ioDat: Advertencia: %v\n", err)
	}

	if *toStdout {
		if err := output.PrintOutput(inv); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		path, hash, err := output.GenerateOutput(inv, dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "ioDat: ¡Listo! Archivo generado:\n")
		fmt.Fprintf(os.Stderr, "  %s\n", path)
		fmt.Fprintf(os.Stderr, "  SHA-256: %s\n", hash)
	}
}
