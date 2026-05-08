package collector

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// GenerateOutput genera el archivo JSON con hash embebido.
// outputDir: directorio destino (vacío = directorio actual).
// Retorna la ruta del archivo generado y el hash SHA-256.
func GenerateOutput(inv *Inventory, outputDir string) (string, string, error) {
	// Embed hash (placeholder inicial)
	inv.CollectorHash = "pending"

	// Serializar a JSON
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("error serializando: %w", err)
	}

	// Calcular hash del contenido
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	// Re-serializar con hash real
	inv.CollectorHash = hashStr
	data, err = json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("error re-serializando: %w", err)
	}

	// Generar nombre de archivo
	hostname := inv.Hostname
	if hostname == "" || hostname == "DESCONOCIDO" {
		hostname = "unknown"
	}
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("ioDat_%s_%s.json", hostname, timestamp)

	// Determinar ruta de salida
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
func PrintOutput(inv *Inventory) error {
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializando: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// ReadFromFile lee un archivo JSON de inventario y lo parsea.
func ReadFromFile(path string) (*Inventory, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error abriendo archivo: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo: %w", err)
	}

	var inv Inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return nil, fmt.Errorf("error parseando JSON: %w", err)
	}

	return &inv, nil
}
