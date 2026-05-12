//go:build windows

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

// FakeCommandRunner helpers for common PowerShell patterns
func psKey(script string) string {
	return "powershell -NoProfile -Command " + script
}

func psJSONKey(script string) string {
	return psKey(script + " | ConvertTo-Json -Compress -Depth 2")
}

func psGetKey(wmiClass, property string) string {
	return psKey("(" + wmiClass + ")." + property)
}

func TestGetSystemInfo_Windows(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			psJSONKey("Get-CimInstance Win32_ComputerSystem -ErrorAction Stop | Select-Object Manufacturer,Model,SystemType"): readFixture("windows", "cim_computersystem.json"),
			psGetKey("Get-CimInstance Win32_BIOS", "SerialNumber"):                                                         "ABC123",
			psJSONKey("Get-CimInstance Win32_OperatingSystem | Select-Object Caption,Version,OSArchitecture"):               `[{"Caption":"Microsoft Windows 11 Pro","Version":"10.0.22631","OSArchitecture":"64-bit"}]`,
		},
	}

	si, err := getSystemInfo(fake)
	if err != nil {
		t.Fatalf("getSystemInfo() error: %v", err)
	}

	if si.Manufacturer != "Dell Inc." {
		t.Errorf("Manufacturer = %q, want %q", si.Manufacturer, "Dell Inc.")
	}
	if si.Model != "OptiPlex 7080" {
		t.Errorf("Model = %q, want %q", si.Model, "OptiPlex 7080")
	}
	if si.SerialNumber != "ABC123" {
		t.Errorf("SerialNumber = %q, want %q", si.SerialNumber, "ABC123")
	}
	if si.OS != "Microsoft Windows 11 Pro" {
		t.Errorf("OS = %q, want %q", si.OS, "Microsoft Windows 11 Pro")
	}
	if si.OSVersion != "10.0.22631" {
		t.Errorf("OSVersion = %q, want %q", si.OSVersion, "10.0.22631")
	}
	if si.OSArchitecture != "64-bit" {
		t.Errorf("OSArchitecture = %q, want %q", si.OSArchitecture, "64-bit")
	}
}

func TestGetCPU_Windows(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			psJSONKey("Get-CimInstance Win32_Processor -ErrorAction Stop | Select-Object Name,NumberOfCores,NumberOfLogicalProcessors,MaxClockSpeed"): readFixture("windows", "cim_processor.json"),
		},
	}

	cpu, err := getCPU(fake)
	if err != nil {
		t.Fatalf("getCPU() error: %v", err)
	}

	if cpu.Name != "Intel(R) Core(TM) i7-10700 CPU @ 2.90GHz" {
		t.Errorf("Name = %q, want %q", cpu.Name, "Intel(R) Core(TM) i7-10700 CPU @ 2.90GHz")
	}
	if cpu.Cores != 8 {
		t.Errorf("Cores = %d, want %d", cpu.Cores, 8)
	}
	if cpu.LogicalProcessors != 16 {
		t.Errorf("LogicalProcessors = %d, want %d", cpu.LogicalProcessors, 16)
	}
	if cpu.MaxClockMHz != 2900 {
		t.Errorf("MaxClockMHz = %d, want %d", cpu.MaxClockMHz, 2900)
	}
	// NameClean should strip (R), (TM), "CPU", and @ speed
	if cpu.NameClean != "Intel Core i7-10700" {
		t.Errorf("NameClean = %q, want %q", cpu.NameClean, "Intel Core i7-10700")
	}
}

func TestGetRAM_Windows(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			psGetKey("Get-CimInstance Win32_ComputerSystem", "TotalPhysicalMemory"): "17179869184",
			psJSONKey("Get-CimInstance Win32_PhysicalMemory | Select-Object BankLabel,Capacity,Speed,MemoryType,ConfiguredClockSpeed"): readFixture("windows", "cim_physical_memory.json"),
		},
	}

	ram, err := getRAM(fake)
	if err != nil {
		t.Fatalf("getRAM() error: %v", err)
	}

	if ram.TotalGB != 16 {
		t.Errorf("TotalGB = %d, want %d", ram.TotalGB, 16)
	}
	if ram.Formatted != "16GB" {
		t.Errorf("Formatted = %q, want %q", ram.Formatted, "16GB")
	}
	if len(ram.Slots) != 2 {
		t.Fatalf("len(Slots) = %d, want 2", len(ram.Slots))
	}
	if ram.Slots[0].SizeGB != 8 {
		t.Errorf("Slot[0].SizeGB = %d, want %d", ram.Slots[0].SizeGB, 8)
	}
	if ram.Slots[0].SpeedMHz != 3200 {
		t.Errorf("Slot[0].SpeedMHz = %d, want %d", ram.Slots[0].SpeedMHz, 3200)
	}
	if ram.Slots[0].Type != "DDR4" {
		t.Errorf("Slot[0].Type = %q, want %q", ram.Slots[0].Type, "DDR4")
	}
}

func TestGetGPU_Windows(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			psJSONKey("Get-CimInstance Win32_VideoController | Select-Object Name,AdapterRAM,DriverVersion"): readFixture("windows", "cim_videocontroller.json"),
		},
	}

	gpus, err := getGPU(fake)
	if err != nil {
		t.Fatalf("getGPU() error: %v", err)
	}

	if len(gpus) != 1 {
		t.Fatalf("len(GPUs) = %d, want 1", len(gpus))
	}
	if gpus[0].Name != "NVIDIA GeForce RTX 3070" {
		t.Errorf("Name = %q, want %q", gpus[0].Name, "NVIDIA GeForce RTX 3070")
	}
	if gpus[0].MemoryGB != 8 {
		t.Errorf("MemoryGB = %d, want %d", gpus[0].MemoryGB, 8)
	}
	if gpus[0].DriverVersion != "31.0.15.3742" {
		t.Errorf("DriverVersion = %q, want %q", gpus[0].DriverVersion, "31.0.15.3742")
	}
}

func TestGetStorage_Windows(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			psJSONKey("Get-CimInstance Win32_DiskDrive | Select-Object Model,SerialNumber,Size,InterfaceType,MediaType"): readFixture("windows", "cim_diskdrive.json"),
		},
	}

	drives, err := getStorage(fake)
	if err != nil {
		t.Fatalf("getStorage() error: %v", err)
	}

	if len(drives) != 2 {
		t.Fatalf("len(Drives) = %d, want 2", len(drives))
	}

	// SSD (NVMe)
	if drives[0].Model != "Samsung SSD 970 EVO Plus" {
		t.Errorf("Drive[0].Model = %q, want %q", drives[0].Model, "Samsung SSD 970 EVO Plus")
	}
	if drives[0].Type != "SSD" {
		t.Errorf("Drive[0].Type = %q, want %q", drives[0].Type, "SSD")
	}
	if drives[0].Interface != "NVMe" {
		t.Errorf("Drive[0].Interface = %q, want %q", drives[0].Interface, "NVMe")
	}
	if drives[0].SizeGB != 500 {
		t.Errorf("Drive[0].SizeGB = %d, want %d", drives[0].SizeGB, 500)
	}

	// HDD (SATA)
	if drives[1].Type != "HDD" {
		t.Errorf("Drive[1].Type = %q, want %q", drives[1].Type, "HDD")
	}
	if drives[1].Interface != "SATA" {
		t.Errorf("Drive[1].Interface = %q, want %q", drives[1].Interface, "SATA")
	}
	if drives[1].SizeGB != 2000 {
		t.Errorf("Drive[1].SizeGB = %d, want %d", drives[1].SizeGB, 2000)
	}
}

func TestMemoryType_Windows(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{20, "DDR"},
		{21, "DDR2"},
		{24, "DDR3"},
		{26, "DDR4"},
		{34, "DDR5"},
		{0, ""},
		{99, ""},
	}
	for _, tc := range tests {
		got := memoryType(tc.input)
		if got != tc.expected {
			t.Errorf("memoryType(%d) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestDiskType_Windows(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SSD", "SSD"},
		{"Solid state drive", "SSD"},
		{"HDD", "HDD"},
		{"Fixed hard disk media", "HDD"},
		{"", ""},
		{"NVMe", ""},
	}
	for _, tc := range tests {
		got := diskType(tc.input)
		if got != tc.expected {
			t.Errorf("diskType(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestDecodeUint16_Windows(t *testing.T) {
	tests := []struct {
		input    []uint16
		expected string
	}{
		{[]uint16{68, 69, 76, 76, 0}, "DELL"},
		{[]uint16{}, ""},
		{[]uint16{72, 80, 0}, "HP"},
	}
	for _, tc := range tests {
		got := decodeUint16(tc.input)
		if got != tc.expected {
			t.Errorf("decodeUint16(%v) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestRun_Windows(t *testing.T) {
	fake := &FakeCommandRunner{
		Responses: map[string]string{
			psKey("hostname"): "PC-CONTABILIDAD",
			psJSONKey("Get-CimInstance Win32_ComputerSystem -ErrorAction Stop | Select-Object Manufacturer,Model,SystemType"):           readFixture("windows", "cim_computersystem.json"),
			psGetKey("Get-CimInstance Win32_BIOS", "SerialNumber"):                                                                    "ABC123",
			psJSONKey("Get-CimInstance Win32_OperatingSystem | Select-Object Caption,Version,OSArchitecture"):                          `[{"Caption":"Microsoft Windows 11 Pro","Version":"10.0.22631","OSArchitecture":"64-bit"}]`,
			psJSONKey("Get-CimInstance Win32_Processor -ErrorAction Stop | Select-Object Name,NumberOfCores,NumberOfLogicalProcessors,MaxClockSpeed"): readFixture("windows", "cim_processor.json"),
			psGetKey("Get-CimInstance Win32_ComputerSystem", "TotalPhysicalMemory"):                                                   "17179869184",
			psJSONKey("Get-CimInstance Win32_PhysicalMemory | Select-Object BankLabel,Capacity,Speed,MemoryType,ConfiguredClockSpeed"): readFixture("windows", "cim_physical_memory.json"),
			psJSONKey("Get-CimInstance Win32_DiskDrive | Select-Object Model,SerialNumber,Size,InterfaceType,MediaType"):              readFixture("windows", "cim_diskdrive.json"),
			psJSONKey("Get-CimInstance Win32_BaseBoard | Select-Object Manufacturer,Product,SerialNumber"):                            readFixture("windows", "cim_baseboard.json"),
			psJSONKey("Get-CimInstance Win32_BIOS | Select-Object SMBIOSBIOSVersion,ReleaseDate"):                                     readFixture("windows", "cim_bios.json"),
			psJSONKey("Get-CimInstance Win32_VideoController | Select-Object Name,AdapterRAM,DriverVersion"):                          readFixture("windows", "cim_videocontroller.json"),
			psJSONKey(`Get-CimInstance WmiMonitorID -Namespace root\wmi | Select-Object MonitorManufacturerID,Name,MonitorID,ScreenWidth,ScreenHeight,SerialNumberID`): `[]`,
			psJSONKey("Get-CimInstance Win32_NetworkAdapter | Where-Object { $_.NetEnabled -eq $true } | Select-Object Name,MACAddress,Speed,NetEnabled,DhcpEnabled"): readFixture("windows", "cim_network_adapter.json"),
			psJSONKey(`Get-CimInstance Win32_NetworkAdapterConfiguration | Where-Object { $_.IPAddress -ne $null } | Select-Object Index,IPAddress,@{N='DHCPEnabled';E={$_.DHCPEnabled}}`): readFixture("windows", "cim_net_adapter_config.json"),
		},
	}

	inv, err := Run(fake)
	if err != nil {
		t.Logf("Run() partial errors: %v", err)
	}
	if inv == nil {
		t.Fatal("Run() returned nil inventory")
	}

	if inv.Hostname != "PC-CONTABILIDAD" {
		t.Errorf("Hostname = %q, want %q", inv.Hostname, "PC-CONTABILIDAD")
	}
	if inv.CollectorVersion != "1.0.0" {
		t.Errorf("CollectorVersion = %q, want %q", inv.CollectorVersion, "1.0.0")
	}
	if inv.System.Manufacturer != "Dell Inc." {
		t.Errorf("Manufacturer = %q, want %q", inv.System.Manufacturer, "Dell Inc.")
	}
	if inv.System.Model != "OptiPlex 7080" {
		t.Errorf("Model = %q, want %q", inv.System.Model, "OptiPlex 7080")
	}
	if inv.CPU.Cores != 8 {
		t.Errorf("CPU.Cores = %d, want %d", inv.CPU.Cores, 8)
	}
	if inv.RAM.TotalGB != 16 {
		t.Errorf("RAM.TotalGB = %d, want %d", inv.RAM.TotalGB, 16)
	}
	if len(inv.Storage) != 2 {
		t.Errorf("len(Storage) = %d, want 2", len(inv.Storage))
	}
	if len(inv.GPU) != 1 {
		t.Errorf("len(GPU) = %d, want 1", len(inv.GPU))
	}
}
