//go:build darwin

package collector

import (
	"os"
	"path/filepath"
	"testing"
)

func readFixture(so, name string) string {
	data, err := os.ReadFile(filepath.Join("testdata", so, name))
	if err != nil {
		data, err = os.ReadFile(filepath.Join("pkg/collector/testdata", so, name))
		if err != nil {
			return ""
		}
	}
	return string(data)
}

func TestGetSystemInfo_Darwin(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"system_profiler SPHardwareDataType -json": readFixture("darwin", "system_profiler_SPHardwareDataType.json"),
			"system_profiler SPSoftwareDataType -json": readFixture("darwin", "system_profiler_SPSoftwareDataType.json"),
			"uname -m": "arm64",
		},
	}

	si := getSystemInfo(fake)

	if si.Manufacturer != "Apple" {
		t.Errorf("Manufacturer = %q, want %q", si.Manufacturer, "Apple")
	}
	if si.Model != "MacBookPro18,3" {
		t.Errorf("Model = %q, want %q", si.Model, "MacBookPro18,3")
	}
	if si.SerialNumber != "FGHIJKLMNOPQ" {
		t.Errorf("SerialNumber = %q, want %q", si.SerialNumber, "FGHIJKLMNOPQ")
	}
	if si.OS != "macOS" {
		t.Errorf("OS = %q, want %q", si.OS, "macOS")
	}
	if si.OSVersion != "macOS 14.5 (23F79)" {
		t.Errorf("OSVersion = %q, want %q", si.OSVersion, "macOS 14.5 (23F79)")
	}
	if si.OSArchitecture != "arm64" {
		t.Errorf("OSArchitecture = %q, want %q", si.OSArchitecture, "arm64")
	}
}

func TestGetCPU_Darwin(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"sysctl -n machdep.cpu.brand_string": "Apple M1 Pro",
			"sysctl -n machdep.cpu.core_count":   "10",
			"sysctl -n machdep.cpu.thread_count": "10",
			"sysctl -n hw.cpufrequency_max":      "",
		},
	}

	cpu := getCPU(fake)

	if cpu.Name != "Apple M1 Pro" {
		t.Errorf("Name = %q, want %q", cpu.Name, "Apple M1 Pro")
	}
	if cpu.Cores != 10 {
		t.Errorf("Cores = %d, want %d", cpu.Cores, 10)
	}
	if cpu.LogicalProcessors != 10 {
		t.Errorf("LogicalProcessors = %d, want %d", cpu.LogicalProcessors, 10)
	}
	// M1 doesn't report cpufrequency_max, so MaxClockMHz should be 0
	if cpu.MaxClockMHz != 0 {
		t.Errorf("MaxClockMHz = %d, want 0 (M1 doesn't report this)", cpu.MaxClockMHz)
	}
}

func TestGetRAM_Darwin(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"sysctl -n hw.memsize":                   "34359738368",
			"system_profiler SPMemoryDataType -json": readFixture("darwin", "system_profiler_SPMemoryDataType.json"),
		},
	}

	ram := getRAM(fake)

	if ram.TotalGB != 32 {
		t.Errorf("TotalGB = %d, want %d", ram.TotalGB, 32)
	}
	if ram.Formatted != "32GB" {
		t.Errorf("Formatted = %q, want %q", ram.Formatted, "32GB")
	}
	if len(ram.Slots) != 2 {
		t.Fatalf("len(Slots) = %d, want 2", len(ram.Slots))
	}
	if ram.Slots[0].BankLabel != "DIMM1" {
		t.Errorf("Slot[0].BankLabel = %q, want %q", ram.Slots[0].BankLabel, "DIMM1")
	}
	if ram.Slots[0].SizeGB != 16 {
		t.Errorf("Slot[0].SizeGB = %d, want %d", ram.Slots[0].SizeGB, 16)
	}
	if ram.Slots[0].Type != "DDR5" {
		t.Errorf("Slot[0].Type = %q, want %q", ram.Slots[0].Type, "DDR5")
	}
}

func TestGetStorage_Darwin(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"system_profiler SPStorageDataType -json": readFixture("darwin", "system_profiler_SPStorageDataType.json"),
		},
	}

	storage := getStorage(fake)

	if len(storage) != 1 {
		t.Fatalf("len(Storage) = %d, want 1", len(storage))
	}
	if storage[0].Model != "APPLE SSD AP1024R" {
		t.Errorf("Model = %q, want %q", storage[0].Model, "APPLE SSD AP1024R")
	}
	if storage[0].SizeGB != 994 {
		t.Errorf("SizeGB = %d, want %d", storage[0].SizeGB, 994)
	}
	if storage[0].Type != "SSD" {
		t.Errorf("Type = %q, want %q", storage[0].Type, "SSD")
	}
}

func TestGetMotherboard_Darwin(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"sysctl -n hw.model":       "MacBookPro18,3",
			"ioreg -l":                 `  | "IOPlatformSerialNumber" = "FGHIJKLMNOPQ"`,
			"sysctl -n kern.osversion": "23F79",
		},
	}

	mb := getMotherboard(fake)

	if mb.Manufacturer != "Apple" {
		t.Errorf("Manufacturer = %q, want %q", mb.Manufacturer, "Apple")
	}
	if mb.Product != "MacBookPro18,3" {
		t.Errorf("Product = %q, want %q", mb.Product, "MacBookPro18,3")
	}
	if mb.SerialNumber != "FGHIJKLMNOPQ" {
		t.Errorf("SerialNumber = %q, want %q", mb.SerialNumber, "FGHIJKLMNOPQ")
	}
	if mb.BIOSVersion != "23F79" {
		t.Errorf("BIOSVersion = %q, want %q", mb.BIOSVersion, "23F79")
	}
}

func TestGetGPU_Darwin(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"system_profiler SPDisplaysDataType -json": readFixture("darwin", "system_profiler_SPDisplaysDataType.json"),
		},
	}

	gpus := getGPU(fake)

	if len(gpus) != 1 {
		t.Fatalf("len(GPUs) = %d, want 1", len(gpus))
	}
	if gpus[0].Name != "Apple M1 Pro" {
		t.Errorf("Name = %q, want %q", gpus[0].Name, "Apple M1 Pro")
	}
	if gpus[0].MemoryGB != 16 {
		t.Errorf("MemoryGB = %d, want %d", gpus[0].MemoryGB, 16)
	}
}

func TestGetMonitors_Darwin(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"system_profiler SPDisplaysDataType -json": readFixture("darwin", "system_profiler_SPDisplaysDataType.json"),
		},
	}

	monitors := getMonitors(fake)

	if len(monitors) != 1 {
		t.Fatalf("len(Monitors) = %d, want 1", len(monitors))
	}
	if monitors[0].Model != "Built-in Retina Display" {
		t.Errorf("Model = %q, want %q", monitors[0].Model, "Built-in Retina Display")
	}
	if monitors[0].Resolution != "3456x2234" {
		t.Errorf("Resolution = %q, want %q", monitors[0].Resolution, "3456x2234")
	}
	if monitors[0].SerialNumber != "F0123456" {
		t.Errorf("SerialNumber = %q, want %q", monitors[0].SerialNumber, "F0123456")
	}
}

func TestRun_Darwin(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"hostname": "macbook-pro",
			"system_profiler SPHardwareDataType -json": readFixture("darwin", "system_profiler_SPHardwareDataType.json"),
			"system_profiler SPSoftwareDataType -json": readFixture("darwin", "system_profiler_SPSoftwareDataType.json"),
			"uname -m":                                 "arm64",
			"sysctl -n machdep.cpu.brand_string":       "Apple M1 Pro",
			"sysctl -n machdep.cpu.core_count":         "10",
			"sysctl -n machdep.cpu.thread_count":       "10",
			"sysctl -n hw.cpufrequency_max":            "",
			"sysctl -n hw.memsize":                     "34359738368",
			"system_profiler SPMemoryDataType -json":   readFixture("darwin", "system_profiler_SPMemoryDataType.json"),
			"system_profiler SPStorageDataType -json":  readFixture("darwin", "system_profiler_SPStorageDataType.json"),
			"sysctl -n hw.model":                       "MacBookPro18,3",
			"ioreg -l":                                 `  | "IOPlatformSerialNumber" = "FGHIJKLMNOPQ"`,
			"sysctl -n kern.osversion":                 "23F79",
			"system_profiler SPDisplaysDataType -json": readFixture("darwin", "system_profiler_SPDisplaysDataType.json"),
			"ifconfig -l":                              "lo0 en0 en1",
			"ifconfig en0":                             "en0: ... ether 00:11:22:33:44:55 ... inet 10.0.0.100",
			"ifconfig en0 media":                       "autoselect 1000baseT <full-duplex>",
			"ifconfig en1":                             "en1: ... ether aa:bb:cc:dd:ee:ff",
			"ifconfig en1 media":                       "autoselect",
		},
	}

	inv, err := Run(fake)
	if err != nil {
		t.Logf("Run() partial errors: %v", err)
	}
	if inv == nil {
		t.Fatal("Run() returned nil inventory")
	}

	if inv.Hostname != "macbook-pro" {
		t.Errorf("Hostname = %q, want %q", inv.Hostname, "macbook-pro")
	}
	if inv.System.Manufacturer != "Apple" {
		t.Errorf("Manufacturer = %q, want %q", inv.System.Manufacturer, "Apple")
	}
	if inv.System.Model != "MacBookPro18,3" {
		t.Errorf("Model = %q, want %q", inv.System.Model, "MacBookPro18,3")
	}
	if inv.CPU.Name != "Apple M1 Pro" {
		t.Errorf("CPU.Name = %q, want %q", inv.CPU.Name, "Apple M1 Pro")
	}
	if inv.RAM.TotalGB != 32 {
		t.Errorf("RAM.TotalGB = %d, want %d", inv.RAM.TotalGB, 32)
	}
	if len(inv.Storage) != 1 {
		t.Errorf("len(Storage) = %d, want 1", len(inv.Storage))
	}
	if len(inv.GPU) != 1 {
		t.Errorf("len(GPU) = %d, want 1", len(inv.GPU))
	}
	if len(inv.Network) != 2 {
		t.Errorf("len(Network) = %d, want 2 (en0 + en1)", len(inv.Network))
	}
}
