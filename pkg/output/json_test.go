package output

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ionet-cl/iodat/pkg/inventory"
)

func TestGenerateOutputHashIntegrity(t *testing.T) {
	inv := &inventory.Inventory{
		CollectorVersion: "1.0.0",
		Hostname:         "test-pc",
		System: inventory.SystemInfo{
			Manufacturer: "Test",
			Model:        "TestModel",
			OS:           "TestOS",
		},
		CPU: inventory.CPUInfo{
			Name:  "Test CPU",
			Cores: 4,
		},
	}

	tmpDir := t.TempDir()
	path, hash, err := GenerateOutput(inv, tmpDir)
	if err != nil {
		t.Fatalf("GenerateOutput() error: %v", err)
	}

	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}

	var parsed inventory.Inventory
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("generated file is not valid JSON: %v", err)
	}

	if parsed.CollectorHash != hash {
		t.Errorf("embedded hash = %q, want %q", parsed.CollectorHash, hash)
	}

	if !strings.Contains(filepath.Base(path), "test-pc") {
		t.Errorf("filename does not contain hostname: %s", path)
	}

	if filepath.Ext(path) != ".json" {
		t.Errorf("file extension = %q, want .json", filepath.Ext(path))
	}
}

func TestGenerateOutputHostnameFallback(t *testing.T) {
	inv := &inventory.Inventory{
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

func TestPrintOutputNoHash(t *testing.T) {
	inv := &inventory.Inventory{
		CollectorVersion: "1.0.0",
		Hostname:         "test-pc",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintOutput(inv)

	w.Close()
	os.Stdout = old
	buf := make([]byte, 4096)
	for {
		n, _ := r.Read(buf)
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
	orig := &inventory.Inventory{
		CollectorVersion: "1.0.0",
		Hostname:         "roundtrip-test",
		System: inventory.SystemInfo{
			Manufacturer: "Round",
			Model:        "Trip",
			SerialNumber: "SN-001",
		},
		CPU: inventory.CPUInfo{
			Name:  "RoundTrip CPU",
			Cores: 8,
		},
		RAM: inventory.RAMInfo{
			TotalGB:   16,
			Formatted: "16GB",
		},
	}

	tmpDir := t.TempDir()
	path, hash, err := GenerateOutput(orig, tmpDir)
	if err != nil {
		t.Fatalf("GenerateOutput: %v", err)
	}

	parsed, err := ReadFromFile(path)
	if err != nil {
		t.Fatalf("ReadFromFile: %v", err)
	}

	if parsed.CollectorVersion != orig.CollectorVersion {
		t.Errorf("CollectorVersion = %q, want %q", parsed.CollectorVersion, orig.CollectorVersion)
	}
	if parsed.Hostname != orig.Hostname {
		t.Errorf("Hostname = %q, want %q", parsed.Hostname, orig.Hostname)
	}
	if parsed.CollectorHash != hash {
		t.Errorf("CollectorHash = %q, want %q", parsed.CollectorHash, hash)
	}

	parsed.CollectorHash = ""
	data, _ := json.MarshalIndent(parsed, "", "  ")
	computedHash := sha256.Sum256(data)
	computedHex := hex.EncodeToString(computedHash[:])

	if computedHex != hash {
		t.Errorf("hash integrity check failed: computed %q, want %q", computedHex, hash)
	}
}
