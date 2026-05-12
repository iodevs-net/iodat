package collector

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateOutputHashIntegrity(t *testing.T) {
	// Create a minimal inventory
	inv := &Inventory{
		CollectorVersion: "1.0.0",
		Hostname:         "test-pc",
		System: SystemInfo{
			Manufacturer: "Test",
			Model:        "TestModel",
			OS:           "TestOS",
		},
		CPU: CPUInfo{
			Name:  "Test CPU",
			Cores: 4,
		},
	}

	// Generate output to temp dir
	tmpDir := t.TempDir()
	path, hash, err := GenerateOutput(inv, tmpDir)
	if err != nil {
		t.Fatalf("GenerateOutput() error: %v", err)
	}

	// Verify hash is 64 hex chars (SHA-256)
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}

	// Read the file back
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}

	// Verify it's valid JSON
	var parsed Inventory
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("generated file is not valid JSON: %v", err)
	}

	// The embedded hash should match the parameter
	if parsed.CollectorHash != hash {
		t.Errorf("embedded hash = %q, want %q", parsed.CollectorHash, hash)
	}

	// Verify hostname is in filename
	if !strings.Contains(filepath.Base(path), "test-pc") {
		t.Errorf("filename does not contain hostname: %s", path)
	}

	// Verify extension is .json
	if filepath.Ext(path) != ".json" {
		t.Errorf("file extension = %q, want .json", filepath.Ext(path))
	}
}

func TestGenerateOutputHostnameFallback(t *testing.T) {
	inv := &Inventory{
		CollectorVersion: "1.0.0",
		Hostname:         "",
	}

	tmpDir := t.TempDir()
	path, _, err := GenerateOutput(inv, tmpDir)
	if err != nil {
		t.Fatalf("GenerateOutput() error: %v", err)
	}

	if !strings.Contains(filepath.Base(path), "unknown") {
		t.Errorf("expected 'unknown' in filename, got: %s", filepath.Base(path))
	}
}

func TestGenerateOutputDefaultDir(t *testing.T) {
	inv := &Inventory{
		CollectorVersion: "1.0.0",
		Hostname:         "test-pc",
	}

	// With empty outputDir, should create in current directory
	path, _, err := GenerateOutput(inv, "")
	if err != nil {
		t.Fatalf("GenerateOutput() error: %v", err)
	}

	// Cleanup
	defer os.Remove(path)

	// Should be in current dir
	dir := filepath.Dir(path)
	cwd, _ := os.Getwd()
	if dir != cwd && dir != "." {
		t.Errorf("file created in %q, want current dir %q", dir, cwd)
	}
}

func TestPrintOutputNoHash(t *testing.T) {
	inv := &Inventory{
		CollectorVersion: "1.0.0",
		Hostname:         "test-pc",
	}

	// Redirect stdout to avoid polluting test output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintOutput(inv)

	w.Close()
	os.Stdout = old
	// Drain the pipe
	readBuf := make([]byte, 4096)
	for {
		n, _ := r.Read(readBuf)
		if n == 0 {
			break
		}
	}

	if err != nil {
		t.Fatalf("PrintOutput() error: %v", err)
	}

	if inv.CollectorHash != "" {
		t.Errorf("after PrintOutput, CollectorHash = %q, want ''", inv.CollectorHash)
	}
}

func TestReadFromFileRoundTrip(t *testing.T) {
	orig := &Inventory{
		CollectorVersion: "1.0.0",
		Hostname:         "roundtrip-test",
		System: SystemInfo{
			Manufacturer: "Round",
			Model:        "Trip",
			SerialNumber: "SN-001",
		},
		CPU: CPUInfo{
			Name:  "RoundTrip CPU",
			Cores: 8,
		},
		RAM: RAMInfo{
			TotalGB:  16,
			Formatted: "16GB",
		},
	}

	// Write
	tmpDir := t.TempDir()
	path, hash, err := GenerateOutput(orig, tmpDir)
	if err != nil {
		t.Fatalf("GenerateOutput: %v", err)
	}

	// Read back
	parsed, err := ReadFromFile(path)
	if err != nil {
		t.Fatalf("ReadFromFile: %v", err)
	}

	// Compare key fields
	if parsed.CollectorVersion != orig.CollectorVersion {
		t.Errorf("CollectorVersion = %q, want %q", parsed.CollectorVersion, orig.CollectorVersion)
	}
	if parsed.Hostname != orig.Hostname {
		t.Errorf("Hostname = %q, want %q", parsed.Hostname, orig.Hostname)
	}
	if parsed.System.Manufacturer != orig.System.Manufacturer {
		t.Errorf("Manufacturer = %q, want %q", parsed.System.Manufacturer, orig.System.Manufacturer)
	}
	if parsed.CPU.Cores != orig.CPU.Cores {
		t.Errorf("CPU.Cores = %d, want %d", parsed.CPU.Cores, orig.CPU.Cores)
	}

	// Hash should be preserved
	if parsed.CollectorHash != hash {
		t.Errorf("CollectorHash = %q, want %q", parsed.CollectorHash, hash)
	}

	// Verify hash integrity: compute hash of content with collector_hash=""
	parsed.CollectorHash = ""
	data, _ := json.MarshalIndent(parsed, "", "  ")
	computedHash := sha256.Sum256(data)
	computedHex := hex.EncodeToString(computedHash[:])

	if computedHex != hash {
		t.Errorf("hash integrity check failed: computed %q, want %q", computedHex, hash)
		t.Log("Set collector_hash to empty, re-marshal, and compute SHA-256")
	}
}
