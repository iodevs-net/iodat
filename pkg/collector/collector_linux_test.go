//go:build linux

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

// fakeLinuxFiles returns a FakeFileSystem pre-loaded with realistic
// Linux /proc and /sys fixture data.
func fakeLinuxFiles() *FakeFileSystem {
	fs := &FakeFileSystem{
		Files: map[string]string{
			"/proc/sys/kernel/hostname":        "test-host",
			"/sys/class/dmi/id/sys_vendor":     "HP\n",
			"/sys/class/dmi/id/product_name":   "ProDesk 600 G5\n",
			"/sys/class/dmi/id/product_serial": "ABC123\n",
			"/etc/os-release":                  "PRETTY_NAME=\"Ubuntu 22.04 LTS\"\nVERSION_ID=\"22.04\"\n",
			"/proc/cpuinfo":                    fixtureCPUInfo(),
			"/proc/meminfo":                    "MemTotal:       16266708 kB\nMemFree:         8234567 kB\n",
			"/sys/class/dmi/id/board_vendor":   "HP\n",
			"/sys/class/dmi/id/board_name":     "8653A\n",
			"/sys/class/dmi/id/board_serial":   "ABC123BOARD\n",
			"/sys/class/dmi/id/bios_version":   "U42 v2.15\n",
			"/sys/class/dmi/id/bios_date":      "04/15/2023\n",
			"/sys/block/sda/device/model":      "ST500DM002-1BD142\n",
			"/sys/block/sda/size":              "976773168\n",
			"/sys/block/sda/device/serial":     "Z1ABCDEF\n",
			"/sys/block/sda/queue/rotational":  "1\n",
			"/sys/class/net/eno1/address":      "04:0e:3c:31:6e:07\n",
			"/sys/class/net/eno1/speed":        "1000\n",
		},
		Dirs: map[string][]string{
			"/sys/block":        {"sda", "sr0"},
			"/sys/class/drm":    {},
			"/sys/class/net":    {"lo", "eno1"},
			"/sys/class/drm/":   {},
			"/sys/class/dmi/id": {},
		},
	}
	return fs
}

func fixtureCPUInfo() string {
	return `processor	: 0
model name	: Intel(R) Core(TM) i5-8500 CPU @ 3.00GHz
cpu MHz		: 800.000
physical id	: 0

processor	: 1
physical id	: 0

processor	: 2
physical id	: 0

processor	: 3
physical id	: 0

processor	: 4
physical id	: 0

processor	: 5
model name	: Intel(R) Core(TM) i5-8500 CPU @ 3.00GHz
cpu MHz		: 3700.000
physical id	: 0
`
}

func TestGetCPU_WithFakeFS(t *testing.T) {
	fs := fakeLinuxFiles()
	cpu := getCPU(fs)

	if cpu.Name != "Intel(R) Core(TM) i5-8500 CPU @ 3.00GHz" {
		t.Errorf("Name = %q, want %q", cpu.Name, "Intel(R) Core(TM) i5-8500 CPU @ 3.00GHz")
	}
	if cpu.NameClean != "Intel Core i5-8500" {
		t.Errorf("NameClean = %q, want %q", cpu.NameClean, "Intel Core i5-8500")
	}
	if cpu.Cores != 1 {
		t.Errorf("Cores = %d, want 1 (single physical id)", cpu.Cores)
	}
	if cpu.LogicalProcessors != 6 {
		t.Errorf("LogicalProcessors = %d, want 6", cpu.LogicalProcessors)
	}
	if cpu.MaxClockMHz != 3700 {
		t.Errorf("MaxClockMHz = %d, want %d", cpu.MaxClockMHz, 3700)
	}
}

func TestGetRAM_WithFakeFS(t *testing.T) {
	fs := fakeLinuxFiles()
	ram := getRAM(fs)

	if ram.TotalGB != 15 {
		t.Errorf("TotalGB = %d, want %d", ram.TotalGB, 15)
	}
	if ram.Formatted != "15GB" {
		t.Errorf("Formatted = %q, want %q", ram.Formatted, "15GB")
	}
}

func TestGetStorage_WithFakeFS(t *testing.T) {
	fs := fakeLinuxFiles()
	storage := getStorage(fs)

	if len(storage) != 1 {
		t.Fatalf("len(Storage) = %d, want 1 (sda only, sr0 skipped)", len(storage))
	}
	if storage[0].Model != "ST500DM002-1BD142" {
		t.Errorf("Model = %q, want %q", storage[0].Model, "ST500DM002-1BD142")
	}
	if storage[0].SizeGB != 500 {
		t.Errorf("SizeGB = %d, want %d", storage[0].SizeGB, 500)
	}
	if storage[0].Type != "HDD" {
		t.Errorf("Type = %q, want %q", storage[0].Type, "HDD")
	}
}

func TestGetMotherboard_WithFakeFS(t *testing.T) {
	fs := fakeLinuxFiles()
	mb := getMotherboard(fs)

	if mb.Manufacturer != "HP" {
		t.Errorf("Manufacturer = %q, want %q", mb.Manufacturer, "HP")
	}
	if mb.Product != "8653A" {
		t.Errorf("Product = %q, want %q", mb.Product, "8653A")
	}
	if mb.SerialNumber != "ABC123BOARD" {
		t.Errorf("SerialNumber = %q, want %q", mb.SerialNumber, "ABC123BOARD")
	}
	if mb.BIOSVersion != "U42 v2.15" {
		t.Errorf("BIOSVersion = %q, want %q", mb.BIOSVersion, "U42 v2.15")
	}
}

func TestGetSystemInfo_WithFakeFS(t *testing.T) {
	fs := fakeLinuxFiles()
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"uname -r": "6.8.0-45-generic",
			"uname -m": "x86_64",
		},
	}

	si := getSystemInfo(fs, fake)

	if si.Manufacturer != "HP" {
		t.Errorf("Manufacturer = %q, want %q", si.Manufacturer, "HP")
	}
	if si.Model != "ProDesk 600 G5" {
		t.Errorf("Model = %q, want %q", si.Model, "ProDesk 600 G5")
	}
	if si.OS != "Ubuntu 22.04 LTS" {
		t.Errorf("OS = %q, want %q", si.OS, "Ubuntu 22.04 LTS")
	}
	if si.OSVersion != "6.8.0-45-generic" {
		t.Errorf("OSVersion = %q, want %q", si.OSVersion, "6.8.0-45-generic")
	}
	if si.OSArchitecture != "x86_64" {
		t.Errorf("OSArchitecture = %q, want %q", si.OSArchitecture, "x86_64")
	}
}

func TestRun_WithFakeFS(t *testing.T) {
	fs := fakeLinuxFiles()
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			"uname -r":          "6.8.0-45-generic",
			"uname -m":          "x86_64",
			"lspci -mm":         readFixture("linux", "lspci_mm.txt"),
			"ip addr show lo":   "1: lo: ... inet 127.0.0.1/8 ...",
			"ip addr show eno1": "2: eno1: ... inet 192.168.1.100/24 brd 192.168.1.255 scope global eno1",
		},
	}

	inv, err := Run(fake, fs)
	if err != nil {
		t.Logf("Run() partial errors: %v", err)
	}
	if inv == nil {
		t.Fatal("Run() returned nil inventory")
	}

	if inv.Hostname != "test-host" {
		t.Errorf("Hostname = %q, want %q", inv.Hostname, "test-host")
	}
	if inv.System.Manufacturer != "HP" {
		t.Errorf("Manufacturer = %q, want %q", inv.System.Manufacturer, "HP")
	}
	if inv.CPU.Name != "Intel(R) Core(TM) i5-8500 CPU @ 3.00GHz" {
		t.Errorf("CPU.Name = %q, want %q", inv.CPU.Name, "Intel(R) Core(TM) i5-8500 CPU @ 3.00GHz")
	}
	if inv.RAM.TotalGB != 15 {
		t.Errorf("RAM.TotalGB = %d, want %d", inv.RAM.TotalGB, 15)
	}
	if len(inv.Storage) != 1 {
		t.Errorf("len(Storage) = %d, want 1", len(inv.Storage))
	}
	if len(inv.Network) != 1 {
		t.Errorf("len(Network) = %d, want 1 (eno1 only, lo skipped)", len(inv.Network))
	}
	if inv.Network[0].MACAddress != "04:0e:3c:31:6e:07" {
		t.Errorf("Network[0].MAC = %q, want %q", inv.Network[0].MACAddress, "04:0e:3c:31:6e:07")
	}
	if inv.Network[0].IPAddress != "192.168.1.100" {
		t.Errorf("Network[0].IP = %q, want %q", inv.Network[0].IPAddress, "192.168.1.100")
	}
}
