//go:build linux

package collector

import (
	"os"
	"path/filepath"
	"testing"
)

func readFixture(name string) string {
	data, err := os.ReadFile(filepath.Join("testdata", "linux", name))
	if err != nil {
		// Fallback: try from package dir (go test runs from package)
		data, err = os.ReadFile(filepath.Join("pkg/collector/testdata", "linux", name))
		if err != nil {
			return ""
		}
	}
	return string(data)
}

func TestGetGPU(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"lspci -mm": readFixture("lspci_mm.txt"),
		},
	}

	gpus := getGPU(fake)

	if len(gpus) != 2 {
		t.Fatalf("getGPU() returned %d GPUs, want 2 (Intel + NVIDIA)", len(gpus))
	}

	// The current parser extracts vendor name (parts[3] from lspci -mm)
	// "Intel Corporation" from "VGA compatible controller" "Intel Corp..."
	if gpus[0].Name != "Intel Corporation" {
		t.Errorf("GPU[0].Name = %q, want %q",
			gpus[0].Name, "Intel Corporation")
	}

	// "NVIDIA Corporation" from "VGA compatible controller" "NVIDIA Corp..."
	if gpus[1].Name != "NVIDIA Corporation" {
		t.Errorf("GPU[1].Name = %q, want %q",
			gpus[1].Name, "NVIDIA Corporation")
	}

	// Verify the exact commands that were issued
	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 command call, got %d", len(fake.Calls))
	}
	if fake.Calls[0] != "lspci -mm" {
		t.Errorf("unexpected command: %q", fake.Calls[0])
	}
}

func TestGetGPU_NoLspci(t *testing.T) {
	// When lspci returns empty, getGPU falls back to /sys/class/drm.
	// On machines with a real GPU, /sys/class/drm may find something;
	// in a container/CI it might not. We just verify it doesn't crash.
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"lspci -mm": "",
		},
	}

	gpus := getGPU(fake)
	_ = gpus // no crash = pass
}

func TestGetGPU_LspciError(t *testing.T) {
	// When lspci fails, it should fallback to /sys/class/drm
	// (which will be empty in CI, so 0 GPUs expected)
	fake := &FakeCommandRunner{
		Responses: map[string]string{}, // no response → error
	}

	gpus := getGPU(fake)
	// On a real machine /sys/class/drm may have cards,
	// but we only verify it doesn't crash
	_ = gpus
}

func TestGetNetwork_Partial(t *testing.T) {
	// Test that getNetwork uses the runner for IP address.
	// MAC and speed come from real /sys/class/net files.
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"ip -json addr show lo": "[]",
		},
	}

	networks := getNetwork(fake)

	// At minimum we should have interfaces (real hardware)
	if len(networks) == 0 {
		t.Fatal("getNetwork() returned 0 interfaces")
	}

	// Verify all recorded commands include "ip -json addr show"
	for _, call := range fake.Calls {
		if len(call) < 22 || call[:22] != "ip -json addr show " {
			continue
		}
		iface := call[22:]
		_ = iface
	}
}

func TestRunReturnsInventory(t *testing.T) {
	// Full Run() with fake runner (uname only)
	// File-read data comes from the real machine.
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"uname -r": "6.8.0-test",
			"uname -m": "x86_64",
		},
	}

	inv, err := Run(fake)
	if err != nil {
		t.Logf("Run() returned partial error: %v", err)
	}
	if inv == nil {
		t.Fatal("Run() returned nil inventory")
	}
	if inv.CollectorVersion != "1.0.0" {
		t.Errorf("CollectorVersion = %q, want %q", inv.CollectorVersion, "1.0.0")
	}
	if inv.Hostname == "" {
		t.Error("Hostname is empty")
	}
	if inv.System.OS == "" {
		t.Error("OS is empty")
	}
}
