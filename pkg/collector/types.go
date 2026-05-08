// Package collector recolecta información del sistema y genera un
// JSON estructurado compatible con ioDesk-3 InventoryImportService.
package collector

// Inventory representa la información completa de un equipo.
type Inventory struct {
	CollectorVersion string           `json:"collector_version"`
	CollectorHash    string           `json:"collector_hash"`
	Hostname         string           `json:"hostname"`
	System           SystemInfo       `json:"system"`
	CPU              CPUInfo          `json:"cpu"`
	RAM              RAMInfo          `json:"ram"`
	Storage          []StorageInfo    `json:"storage"`
	Motherboard      MotherboardInfo  `json:"motherboard"`
	GPU              []GPUInfo        `json:"gpu"`
	Monitors         []MonitorInfo    `json:"monitors"`
	Network          []NetworkInfo    `json:"network"`
	Software         []SoftwareInfo   `json:"installed_software"`
}

// SystemInfo representa el fabricante, modelo y OS.
type SystemInfo struct {
	Manufacturer    string `json:"manufacturer"`
	Model           string `json:"model"`
	SerialNumber    string `json:"serial_number"`
	OS              string `json:"os"`
	OSVersion       string `json:"os_version"`
	OSArchitecture  string `json:"os_architecture"`
}

// CPUInfo representa el procesador.
type CPUInfo struct {
	Name              string `json:"name"`
	NameClean         string `json:"name_clean"`
	Cores             int    `json:"cores"`
	LogicalProcessors int    `json:"logical_processors"`
	MaxClockMHz       int    `json:"max_clock_mhz"`
}

// RAMInfo representa la memoria RAM.
type RAMInfo struct {
	TotalGB   int         `json:"total_gb"`
	Formatted string      `json:"formatted"`
	Slots     []RAMSlot   `json:"slots"`
}

// RAMSlot representa un módulo de memoria.
type RAMSlot struct {
	BankLabel string `json:"bank_label"`
	SizeGB    int    `json:"size_gb"`
	SpeedMHz  int    `json:"speed_mhz"`
	Type      string `json:"type"`
}

// StorageInfo representa un disco.
type StorageInfo struct {
	Model        string `json:"model"`
	SerialNumber string `json:"serial_number"`
	SizeGB       int    `json:"size_gb"`
	Interface    string `json:"interface"`
	Type         string `json:"type"`
}

// MotherboardInfo representa la placa madre.
type MotherboardInfo struct {
	Manufacturer string `json:"manufacturer"`
	Product      string `json:"product"`
	SerialNumber string `json:"serial_number"`
	BIOSVersion  string `json:"bios_version"`
	BIOSDate     string `json:"bios_date"`
}

// GPUInfo representa una tarjeta gráfica.
type GPUInfo struct {
	Name          string `json:"name"`
	MemoryGB      int    `json:"memory_gb"`
	DriverVersion string `json:"driver_version"`
}

// MonitorInfo representa un monitor.
type MonitorInfo struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	SerialNumber string `json:"serial_number"`
	Resolution   string `json:"resolution"`
}

// NetworkInfo representa un adaptador de red.
type NetworkInfo struct {
	Name       string `json:"name"`
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	DHCPEnabled bool  `json:"dhcp_enabled"`
	Speed      int64  `json:"speed"`
}

// SoftwareInfo representa un programa instalado.
type SoftwareInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Publisher   string `json:"publisher"`
	InstallDate string `json:"install_date"`
}
