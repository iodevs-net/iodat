// ioDat — Colector de inventario técnico para ioDesk-3 (IONET).
// Genera un archivo JSON con las especificaciones del equipo
// para ser importado en ioDesk-3.
//
// Uso:
//   iodat                    → genera ioDat_<hostname>_<fecha>.json
//   iodat --stdout           → imprime JSON a stdout
//   iodat /ruta/directorio   → guarda en el directorio especificado

package main

import (
	"fmt"
	"os"

	"github.com/ionet-cl/iodat/pkg/collector"
)

func main() {
	recolectarAStdout := false
	outputDir := ""

	// Parsear argumentos simples
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--stdout", "-s":
			recolectarAStdout = true
		case "--help", "-h":
			fmt.Println("ioDat v1.0.0 — Colector de inventario para ioDesk-3")
			fmt.Println("")
			fmt.Println("Uso:")
			fmt.Println("  iodat                    Genera archivo JSON en el directorio actual")
			fmt.Println("  iodat --stdout           Imprime JSON a stdout")
			fmt.Println("  iodat /ruta/dir          Guarda archivo en /ruta/dir")
			fmt.Println("  iodat --help             Muestra esta ayuda")
			os.Exit(0)
		default:
			// Si es un directorio, usarlo como output
			if info, err := os.Stat(arg); err == nil && info.IsDir() {
				outputDir = arg
			}
		}
	}

	// Recolectar
	fmt.Fprintf(os.Stderr, "ioDat: Recolectando información del equipo...\n")
	inv, err := collector.Run(collector.OSCommandRunner{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ioDat: Advertencia: %v\n", err)
	}

	if recolectarAStdout {
		if err := collector.PrintOutput(inv); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		path, hash, err := collector.GenerateOutput(inv, outputDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "ioDat: ¡Listo! Archivo generado:\n")
		fmt.Fprintf(os.Stderr, "  %s\n", path)
		fmt.Fprintf(os.Stderr, "  SHA-256: %s\n", hash)
	}
}
