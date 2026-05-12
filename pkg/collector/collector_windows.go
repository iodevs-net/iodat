//go:build windows

package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Run recolecta todo el inventario en Windows usando PowerShell + WMI.
// Compatible con Windows 10/11, Server 2016+.
func Run(runner CommandRunner) (*Inventory, error) {
	inv := &Inventory{
		CollectorVersion: "1.0.0",
	}

	var errs []string
	collect := func(label string, fn func() error) {
		if err := fn(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", label, err))
		}
	}

	inv.Hostname = getHostname(runner)
	collect("System", func() error { var err error; inv.System, err = getSystemInfo(runner); return err })
	collect("CPU", func() error { var err error; inv.CPU, err = getCPU(runner); return err })
	collect("RAM", func() error { var err error; inv.RAM, err = getRAM(runner); return err })
	collect("Storage", func() error { var err error; inv.Storage, err = getStorage(runner); return err })
	collect("Motherboard", func() error { var err error; inv.Motherboard, err = getMotherboard(runner); return err })
	collect("GPU", func() error { var err error; inv.GPU, err = getGPU(runner); return err })
	collect("Monitors", func() error { var err error; inv.Monitors, err = getMonitors(runner); return err })
	collect("Network", func() error { var err error; inv.Network, err = getNetwork(runner); return err })
	collect("Software", func() error { var err error; inv.Software, err = getSoftware(runner); return err })

	if len(errs) > 0 {
		return inv, fmt.Errorf("errores parciales (%d): %s", len(errs), strings.Join(errs, "; "))
	}
	return inv, nil
}

// ── Helpers de ejecución ─────────────────────────

// runWithTimeout ejecuta un comando de PowerShell con timeout y retorna stdout
// recortado. Retorna string vacío si falla o expira.
func runWithTimeout(runner CommandRunner, timeout time.Duration, script string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := runner.Run(ctx, "powershell", "-NoProfile", "-Command", script)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// runJSON ejecuta un script de PowerShell, canaliza a ConvertTo-Json y parsea
// el resultado en dest. Retorna error si el comando falla o el JSON es inválido.
func runJSON(runner CommandRunner, timeout time.Duration, script string, dest interface{}) error {
	fullScript := script + " | ConvertTo-Json -Compress -Depth 2"
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := runner.Run(ctx, "powershell", "-NoProfile", "-Command", fullScript)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(strings.TrimSpace(string(out))), dest)
}

// psGet obtiene una propiedad específica de una clase WMI vía PowerShell.
func psGet(runner CommandRunner, wmiClass, property string) string {
	script := fmt.Sprintf("(%s).%s", wmiClass, property)
	return runWithTimeout(runner, CmdTimeoutSlow, script)
}

// ── Hostname ─────────────────────────────────────

func getHostname(runner CommandRunner) string {
	out := runWithTimeout(runner, CmdTimeoutFast, "hostname")
	if out == "" {
		return "DESCONOCIDO"
	}
	return out
}

// ── System ───────────────────────────────────────

func getSystemInfo(runner CommandRunner) (SystemInfo, error) {
	si := SystemInfo{}

	var cs []struct {
		Manufacturer string `json:"Manufacturer"`
		Model        string `json:"Model"`
		SystemType   string `json:"SystemType"`
	}
	if err := runJSON(runner, CmdTimeoutSlow, `Get-CimInstance Win32_ComputerSystem | Select-Object Manufacturer,Model,SystemType`, &cs); err == nil && len(cs) > 0 {
		si.Manufacturer = strings.TrimSpace(cs[0].Manufacturer)
		si.Model = strings.TrimSpace(cs[0].Model)
	} else {
		// fallback a Get-WmiObject
		si.Manufacturer = strings.TrimSpace(psGet(runner, "Get-WmiObject Win32_ComputerSystem", "Manufacturer"))
		si.Model = strings.TrimSpace(psGet(runner, "Get-WmiObject Win32_ComputerSystem", "Model"))
	}

	si.SerialNumber = strings.TrimSpace(psGet(runner, "Get-CimInstance Win32_BIOS", "SerialNumber"))

	var osInfo []struct {
		Caption        string `json:"Caption"`
		Version        string `json:"Version"`
		OSArchitecture string `json:"OSArchitecture"`
	}
	if err := runJSON(runner, CmdTimeoutSlow, `Get-CimInstance Win32_OperatingSystem | Select-Object Caption,Version,OSArchitecture`, &osInfo); err == nil && len(osInfo) > 0 {
		si.OS = strings.TrimSpace(osInfo[0].Caption)
		si.OSVersion = strings.TrimSpace(osInfo[0].Version)
		si.OSArchitecture = strings.TrimSpace(osInfo[0].OSArchitecture)
	}

	return si, nil
}

// ── CPU ──────────────────────────────────────────

func getCPU(runner CommandRunner) (CPUInfo, error) {
	cpu := CPUInfo{}

	var cpus []struct {
		Name                     string `json:"Name"`
		NumberOfCores            int    `json:"NumberOfCores"`
		NumberOfLogicalProcessors int    `json:"NumberOfLogicalProcessors"`
		MaxClockSpeed            int    `json:"MaxClockSpeed"`
	}
	if err := runJSON(runner, CmdTimeoutSlow, `Get-CimInstance Win32_Processor | Select-Object Name,NumberOfCores,NumberOfLogicalProcessors,MaxClockSpeed`, &cpus); err == nil && len(cpus) > 0 {
		cpu.Name = strings.TrimSpace(cpus[0].Name)
		cpu.Cores = cpus[0].NumberOfCores
		cpu.LogicalProcessors = cpus[0].NumberOfLogicalProcessors
		cpu.MaxClockMHz = cpus[0].MaxClockSpeed
	} else {
		cpu.Name = psGet(runner, "Get-WmiObject Win32_Processor", "Name")
		cpu.Cores = parseInt(psGet(runner, "Get-WmiObject Win32_Processor", "NumberOfCores"))
		cpu.LogicalProcessors = parseInt(psGet(runner, "Get-WmiObject Win32_Processor", "NumberOfLogicalProcessors"))
		cpu.MaxClockMHz = parseInt(psGet(runner, "Get-WmiObject Win32_Processor", "MaxClockSpeed"))
	}

	cpu.NameClean = cleanCPUName(cpu.Name)
	return cpu, nil
}

// ── RAM ──────────────────────────────────────────

func getRAM(runner CommandRunner) (RAMInfo, error) {
	ram := RAMInfo{}

	totalBytesStr := psGet(runner, "Get-CimInstance Win32_ComputerSystem", "TotalPhysicalMemory")
	totalBytes := parseFloat(totalBytesStr)
	ram.TotalGB = int(totalBytes / (1024 * 1024 * 1024))
	if ram.TotalGB > 0 {
		ram.Formatted = fmt.Sprintf("%dGB", ram.TotalGB)
	}

	var slots []struct {
		BankLabel       string `json:"BankLabel"`
		Capacity        int64  `json:"Capacity"`
		Speed           int    `json:"Speed"`
		MemoryType      int    `json:"MemoryType"`
		ConfiguredClock int    `json:"ConfiguredClockSpeed"`
	}
	if err := runJSON(runner, CmdTimeoutSlow, `Get-CimInstance Win32_PhysicalMemory | Select-Object BankLabel,Capacity,Speed,MemoryType,ConfiguredClockSpeed`, &slots); err == nil {
		for _, slot := range slots {
			ram.Slots = append(ram.Slots, RAMSlot{
				BankLabel: strings.TrimSpace(slot.BankLabel),
				SizeGB:    int(slot.Capacity / (1024 * 1024 * 1024)),
				SpeedMHz:  slot.Speed,
				Type:      memoryType(slot.MemoryType),
			})
		}
	}

	return ram, nil
}

// ── Storage ──────────────────────────────────────

func getStorage(runner CommandRunner) ([]StorageInfo, error) {
	var drives []struct {
		Model        string `json:"Model"`
		SerialNumber string `json:"SerialNumber"`
		Size         string `json:"Size"`
		InterfaceType string `json:"InterfaceType"`
		MediaType    string `json:"MediaType"`
	}
	if err := runJSON(runner, CmdTimeoutSlow, `Get-CimInstance Win32_DiskDrive | Select-Object Model,SerialNumber,Size,InterfaceType,MediaType`, &drives); err != nil {
		return nil, err
	}

	var result []StorageInfo
	for _, d := range drives {
		sizeGB := int(parseFloat(d.Size) / (1000 * 1000 * 1000))
		result = append(result, StorageInfo{
			Model:        strings.TrimSpace(decodeUint16([]uint16(runeArray(d.Model)))),
			SerialNumber: strings.TrimSpace(d.SerialNumber),
			SizeGB:       sizeGB,
			Interface:    strings.TrimSpace(d.InterfaceType),
			Type:         diskType(d.MediaType),
		})
	}
	return result, nil
}

// ── Motherboard ──────────────────────────────────

func getMotherboard(runner CommandRunner) (MotherboardInfo, error) {
	mb := MotherboardInfo{}

	var boards []struct {
		Manufacturer string `json:"Manufacturer"`
		Product      string `json:"Product"`
		SerialNumber string `json:"SerialNumber"`
	}
	if err := runJSON(runner, CmdTimeoutSlow, `Get-CimInstance Win32_BaseBoard | Select-Object Manufacturer,Product,SerialNumber`, &boards); err == nil && len(boards) > 0 {
		mb.Manufacturer = strings.TrimSpace(decodeUint16([]uint16(runeArray(boards[0].Manufacturer))))
		mb.Product = strings.TrimSpace(decodeUint16([]uint16(runeArray(boards[0].Product))))
		mb.SerialNumber = strings.TrimSpace(boards[0].SerialNumber)
	}

	var bios []struct {
		SMBIOSBIOSVersion string `json:"SMBIOSBIOSVersion"`
		ReleaseDate       string `json:"ReleaseDate"`
	}
	if err := runJSON(runner, CmdTimeoutSlow, `Get-CimInstance Win32_BIOS | Select-Object SMBIOSBIOSVersion,ReleaseDate`, &bios); err == nil && len(bios) > 0 {
		mb.BIOSVersion = strings.TrimSpace(bios[0].SMBIOSBIOSVersion)
		mb.BIOSDate = strings.TrimSpace(bios[0].ReleaseDate)
	}

	return mb, nil
}

// ── GPU ──────────────────────────────────────────

func getGPU(runner CommandRunner) ([]GPUInfo, error) {
	var adapters []struct {
		Name              string `json:"Name"`
		AdapterRAM        int64  `json:"AdapterRAM"`
		DriverVersion     string `json:"DriverVersion"`
	}
	if err := runJSON(runner, CmdTimeoutSlow, `Get-CimInstance Win32_VideoController | Select-Object Name,AdapterRAM,DriverVersion`, &adapters); err != nil {
		return nil, err
	}

	var result []GPUInfo
	for _, a := range adapters {
		gpu := GPUInfo{
			Name:          strings.TrimSpace(a.Name),
			DriverVersion: strings.TrimSpace(a.DriverVersion),
		}
		if a.AdapterRAM > 0 {
			gpu.MemoryGB = int(a.AdapterRAM / (1024 * 1024 * 1024))
		}
		result = append(result, gpu)
	}
	return result, nil
}

// ── Monitors ─────────────────────────────────────

func getMonitors(runner CommandRunner) ([]MonitorInfo, error) {
	var monitors []struct {
		MonitorManufacturerID uint16 `json:"MonitorManufacturerID"`
		Name                 string `json:"Name"`
		MonitorID            string `json:"MonitorID"`
		ScreenWidth          uint32 `json:"ScreenWidth"`
		ScreenHeight         uint32 `json:"ScreenHeight"`
		SerialNumberID       string `json:"SerialNumberID"`
	}
	if err := runJSON(runner, CmdTimeoutSlow, `Get-CimInstance WmiMonitorID -Namespace root\wmi | Select-Object MonitorManufacturerID,Name,MonitorID,ScreenWidth,ScreenHeight,SerialNumberID`, &monitors); err != nil {
		return nil, err
	}

	var result []MonitorInfo
	for _, m := range monitors {
		mi := MonitorInfo{
			Manufacturer: decodeUint16([]uint16{m.MonitorManufacturerID}),
			Model:        decodeUint16(m.Name[:]),
			SerialNumber: decodeUint16([]uint16(runeArray(m.SerialNumberID))),
		}
		if m.ScreenWidth > 0 && m.ScreenHeight > 0 {
			mi.Resolution = fmt.Sprintf("%dx%d", m.ScreenWidth, m.ScreenHeight)
		}
		if mi.Model != "" {
			result = append(result, mi)
		}
	}
	return result, nil
}

// ── Network ──────────────────────────────────────

func getNetwork(runner CommandRunner) ([]NetworkInfo, error) {
	var adapters []struct {
		Name        string `json:"Name"`
		MACAddress  string `json:"MACAddress"`
		Speed       int64  `json:"Speed"`
		NetEnabled  bool   `json:"NetEnabled"`
		DhcpEnabled bool   `json:"DhcpEnabled"`
	}
	if err := runJSON(runner, CmdTimeoutSlow, `Get-CimInstance Win32_NetworkAdapter | Where-Object { $_.NetEnabled -eq $true } | Select-Object Name,MACAddress,Speed,NetEnabled,DhcpEnabled`, &adapters); err != nil {
		return nil, err
	}

	var result []NetworkInfo
	for _, a := range adapters {
		result = append(result, NetworkInfo{
			Name:       strings.TrimSpace(a.Name),
			MACAddress: strings.TrimSpace(a.MACAddress),
			Speed:      a.Speed,
			DHCPEnabled: a.DhcpEnabled,
		})
	}

	var ips []struct {
		Index       int      `json:"Index"`
		IPAddress   []string `json:"IPAddress"`
		DHCPEnabled bool     `json:"DHCPEnabled"`
	}
	if err := runJSON(runner, CmdTimeoutSlow, `Get-CimInstance Win32_NetworkAdapterConfiguration | Where-Object { $_.IPAddress -ne $null } | Select-Object Index,IPAddress,@{N='DHCPEnabled';E={$_.DHCPEnabled}}`, &ips); err == nil {
		for i := range result {
			for _, ip := range ips {
				for _, addr := range ip.IPAddress {
					if !strings.Contains(addr, ":") {
						result[i].IPAddress = addr
						result[i].DHCPEnabled = ip.DHCPEnabled
						break
					}
				}
			}
		}
	}

	return result, nil
}

// ── Software ─────────────────────────────────────

func getSoftware(runner CommandRunner) ([]SoftwareInfo, error) {
	script := `
$paths = @(
	"HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*",
	"HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*"
)
Get-ItemProperty $paths 2>$null | Where-Object { $_.DisplayName -ne $null } | 
Select-Object @{N='Name';E={$_.DisplayName}}, @{N='Version';E={$_.DisplayVersion}}, @{N='Publisher';E={$_.Publisher}}, @{N='InstallDate';E={$_.InstallDate}} | 
ConvertTo-Json -Compress
`
	out := runWithTimeout(runner, CmdTimeoutSlow, script)
	if out == "" {
		return nil, nil
	}

	var software []struct {
		Name        string `json:"Name"`
		Version     string `json:"Version"`
		Publisher   string `json:"Publisher"`
		InstallDate string `json:"InstallDate"`
	}
	var result []SoftwareInfo
	if err := json.Unmarshal([]byte(out), &software); err == nil {
		maxSW := 200
		if len(software) > maxSW {
			software = software[:maxSW]
		}
		for _, s := range software {
			result = append(result, SoftwareInfo{
				Name:        strings.TrimSpace(s.Name),
				Version:     strings.TrimSpace(s.Version),
				Publisher:   strings.TrimSpace(s.Publisher),
				InstallDate: strings.TrimSpace(s.InstallDate),
			})
		}
	}
	return result, nil
}

// ── Helpers de parseo ────────────────────────────

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func memoryType(typeDetail int) string {
	types := map[int]string{
		20: "DDR",
		21: "DDR2",
		24: "DDR3",
		26: "DDR4",
		34: "DDR5",
	}
	if t, ok := types[typeDetail]; ok {
		return t
	}
	return ""
}

func diskType(mediaType string) string {
	mt := strings.ToLower(mediaType)
	if strings.Contains(mt, "ssd") || strings.Contains(mt, "solid") {
		return "SSD"
	}
	if strings.Contains(mt, "hdd") || strings.Contains(mt, "fixed") {
		return "HDD"
	}
	return ""
}

func decodeUint16(data []uint16) string {
	if len(data) == 0 {
		return ""
	}
	runes := make([]rune, len(data))
	for i, v := range data {
		if v == 0 {
			runes = runes[:i]
			break
		}
		runes[i] = rune(v)
	}
	return strings.TrimSpace(string(runes))
}

// runeArray convierte un string en un slice de runas (necesario para WMI).
func runeArray(s string) []rune {
	return []rune(s)
}
