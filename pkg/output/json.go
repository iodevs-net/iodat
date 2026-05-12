// Package output serializes inventory data to JSON with integrity hashing.
package output

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ionet-cl/iodat/pkg/inventory"
)

// GenerateOutput genera el archivo JSON con hash embebido.
// outputDir: directorio destino (vacío = directorio actual).
// Retorna la ruta del archivo generado y el hash SHA-256.
//
// El hash se calcula sobre el contenido JSON con collector_hash="".
// Luego se reemplaza por el hash real en el archivo final.
// Para verificar: reemplazar "collector_hash":"<valor>" por
// "collector_hash":"" en el JSON y comparar el hash.
func GenerateOutput(inv *inventory.Inventory, outputDir string) (string, string, error) {
	inv.CollectorHash = ""
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("error serializando: %w", err)
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	inv.CollectorHash = hashStr
	data, err = json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("error re-serializando: %w", err)
	}

	hostname := inv.Hostname
	if hostname == "" || hostname == "DESCONOCIDO" {
		hostname = "unknown"
	}
	filename := fmt.Sprintf("ioDat_%s.json", hostname)

	outputPath := filename
	if outputDir != "" {
		outputPath = filepath.Join(outputDir, filename)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return "", "", fmt.Errorf("error creando archivo: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return "", "", fmt.Errorf("error escribiendo: %w", err)
	}
	if _, err := f.WriteString("\n"); err != nil {
		return "", "", fmt.Errorf("error escribiendo newline: %w", err)
	}

	return outputPath, hashStr, nil
}

// PrintOutput escribe el JSON a stdout (útil para piping).
// collector_hash aparece vacío en stdout ya que el hash solo es
// significativo en el archivo generado por GenerateOutput.
func PrintOutput(inv *inventory.Inventory) error {
	inv.CollectorHash = ""
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializando: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// ReadFromFile lee un archivo JSON de inventario y lo parsea.
func ReadFromFile(path string) (*inventory.Inventory, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error abriendo archivo: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo: %w", err)
	}

	var inv inventory.Inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return nil, fmt.Errorf("error parseando JSON: %w", err)
	}

	return &inv, nil
}
