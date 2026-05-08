//go:build windows

package collector

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Run recolecta todo el inventario en Windows usando PowerShell + WMI.
// Compatible con Windows 10/11, Server 2016+.
func Run() (*Inventory, error) {
	inv := &Inventory{
		CollectorVersion: "1.0.0",
	}

	var errs []string
	collect := func(label string, fn func() error) {
		if err := fn(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", label, err))
		}
	}

	inv.Hostname = getHostname()
	collect("System", func() error { var err error; inv.System, err = getSystemInfo(); return err })
	collect("CPU", func() error { var err error; inv.CPU, err = getCPU(); return err })
	collect("RAM", func() error { var err error; inv.RAM, err = getRAM(); return err })
	collect("Storage", func() error { var err error; inv.Storage, err = getStorage(); return err })
	collect("Motherboard", func() error { var err error; inv.Motherboard, err = getMotherboard(); return err })
	collect("GPU", func() error { var err error; inv.GPU, err = getGPU(); return err })
	collect("Monitors", func() error { var err error; inv.Monitors, err = getMonitors(); return err })
	collect("Network", func() error { var err error; inv.Network, err = getNetwork(); return err })
	collect("Software", func() error { var err error; inv.Software, err = getSoftware(); return err })

	if len(errs) > 0 {
		return inv, fmt.Errorf("errores parciales (%d): %s", len(errs), strings.Join(errs, "; "))
	}
	return inv, nil
}

func getHostname() string {
	out, err := ps("hostname")
	if err != nil {
		return "DESCONOCIDO"
	}
	return strings.TrimSpace(out)
}

func getSystemInfo() (SystemInfo, error) {
	si := SystemInfo{}
	jsonStr := psJSON(`Get-CimInstance Win32_ComputerSystem | Select-Object Manufacturer,Model,SystemType`)
	if err := json.Unmarshal([]byte(jsonStr), &si); err == nil {
		si.Manufacturer = strings.TrimSpace(si.Manufacturer)
		si.Model = strings.TrimSpace(si.Model)
	} else {
		si.Manufacturer = psGet("Get-WmiObject Win32_ComputerSystem", "Manufacturer")
		si.Model = psGet("Get-WmiObject Win32_ComputerSystem", "Model")
	}

	si.SerialNumber = strings.TrimSpace(psGet("Get-CimInstance Win32_BIOS", "SerialNumber"))

	osJSON := psJSON(`Get-CimInstance Win32_OperatingSystem | Select-Object Caption,Version,OSArchitecture`)
	var osInfo struct {
		Caption        string `json:"Caption"`
		Version        string `json:"Version"`
		OSArchitecture string `json:"OSArchitecture"`
	}
	if err := json.Unmarshal([]byte(osJSON), &osInfo); err == nil {
		si.OS = strings.TrimSpace(osInfo.Caption)
		si.OSVersion = strings.TrimSpace(osInfo.Version)
		si.OSArchitecture = strings.TrimSpace(osInfo.OSArchitecture)
	}

	return si, nil
}

func getCPU() (CPUInfo, error) {
	cpu := CPUInfo{}
	jsonStr := psJSON(`Get-CimInstance Win32_Processor | Select-Object Name,NumberOfCores,NumberOfLogicalProcessors,MaxClockSpeed`)
	var cpus []struct {
		Name                     string `json:"Name"`
		NumberOfCores            int    `json:"NumberOfCores"`
		NumberOfLogicalProcessors int   `json:"NumberOfLogicalProcessors"`
		MaxClockSpeed            int    `json:"MaxClockSpeed"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &cpus); err == nil && len(cpus) > 0 {
		cpu.Name = strings.TrimSpace(cpus[0].Name)
		cpu.Cores = cpus[0].NumberOfCores
		cpu.LogicalProcessors = cpus[0].NumberOfLogicalProcessors
		cpu.MaxClockMHz = cpus[0].MaxClockSpeed
	} else {
		cpu.Name = psGet("Get-WmiObject Win32_Processor", "Name")
		cpu.Cores = parseInt(psGet("Get-WmiObject Win32_Processor", "NumberOfCores"))
		cpu.LogicalProcessors = parseInt(psGet("Get-WmiObject Win32_Processor", "NumberOfLogicalProcessors"))
		cpu.MaxClockMHz = parseInt(psGet("Get-WmiObject Win32_Processor", "MaxClockSpeed"))
	}

	cpu.NameClean = cleanCPUName(cpu.Name)
	return cpu, nil
}

func getRAM() (RAMInfo, error) {
	ram := RAMInfo{}

	totalBytesStr := psGet("Get-CimInstance Win32_ComputerSystem", "TotalPhysicalMemory")
	totalBytes := parseFloat(totalBytesStr)
	ram.TotalGB = int(totalBytes / (1024 * 1024 * 1024))
	if ram.TotalGB <= 0 {
		ram.TotalGB = int(totalBytes / (1000 * 1000 * 1000))
	}
	if ram.TotalGB > 0 {
		ram.Formatted = fmt.Sprintf("%dGB", ram.TotalGB)
	}

	jsonStr := psJSON(`Get-CimInstance Win32_PhysicalMemory | Select-Object BankLabel,Capacity,Speed,FormFactor,@{N='TypeDetail';E={$_.SMBIOSMemoryType}}`)
	var slots []struct {
		BankLabel  string `json:"BankLabel"`
		Capacity   string `json:"Capacity"`
		Speed      int    `json:"Speed"`
		TypeDetail int    `json:"TypeDetail"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &slots); err == nil {
		for _, s := range slots {
			capBytes := parseFloat(s.Capacity)
			ram.Slots = append(ram.Slots, RAMSlot{
				BankLabel: strings.TrimSpace(s.BankLabel),
				SizeGB:    int(capBytes / (1024 * 1024 * 1024)),
				SpeedMHz:  s.Speed,
				Type:      memoryType(s.TypeDetail),
			})
		}
	}

	return ram, nil
}

func getStorage() ([]StorageInfo, error) {
	jsonStr := psJSON(`Get-CimInstance Win32_DiskDrive | Select-Object Model,SerialNumber,Size,InterfaceType,MediaType | Where-Object { $_.Size -gt 0 }`)
	var disks []struct {
		Model         string `json:"Model"`
		SerialNumber  string `json:"SerialNumber"`
		Size          string `json:"Size"`
		InterfaceType string `json:"InterfaceType"`
		MediaType     string `json:"MediaType"`
	}
	var result []StorageInfo
	if err := json.Unmarshal([]byte(jsonStr), &disks); err == nil {
		for _, d := range disks {
			sizeBytes := parseFloat(d.Size)
			result = append(result, StorageInfo{
				Model:        strings.TrimSpace(d.Model),
				SerialNumber: strings.TrimSpace(d.SerialNumber),
				SizeGB:       int(sizeBytes / (1000 * 1000 * 1000)),
				Interface:    strings.TrimSpace(d.InterfaceType),
				Type:         diskType(d.MediaType),
			})
		}
	}
	return result, nil
}

func getMotherboard() (MotherboardInfo, error) {
	mb := MotherboardInfo{}
	jsonStr := psJSON(`Get-CimInstance Win32_BaseBoard | Select-Object Manufacturer,Product,SerialNumber`)
	var boards []struct {
		Manufacturer string `json:"Manufacturer"`
		Product      string `json:"Product"`
		SerialNumber string `json:"SerialNumber"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &boards); err == nil && len(boards) > 0 {
		mb.Manufacturer = strings.TrimSpace(boards[0].Manufacturer)
		mb.Product = strings.TrimSpace(boards[0].Product)
		mb.SerialNumber = strings.TrimSpace(boards[0].SerialNumber)
	}

	biosJSON := psJSON(`Get-CimInstance Win32_BIOS | Select-Object SMBIOSBIOSVersion,ReleaseDate`)
	var bioses []struct {
		SMBIOSBIOSVersion string `json:"SMBIOSBIOSVersion"`
		ReleaseDate       string `json:"ReleaseDate"`
	}
	if err := json.Unmarshal([]byte(biosJSON), &bioses); err == nil && len(bioses) > 0 {
		mb.BIOSVersion = strings.TrimSpace(bioses[0].SMBIOSBIOSVersion)
		mb.BIOSDate = strings.TrimSpace(bioses[0].ReleaseDate)
	}

	return mb, nil
}

func getGPU() ([]GPUInfo, error) {
	jsonStr := psJSON(`Get-CimInstance Win32_VideoController | Select-Object Name,AdapterRAM,DriverVersion`)
	var cards []struct {
		Name          string `json:"Name"`
		AdapterRAM    string `json:"AdapterRAM"`
		DriverVersion string `json:"DriverVersion"`
	}
	var result []GPUInfo
	if err := json.Unmarshal([]byte(jsonStr), &cards); err == nil {
		for _, c := range cards {
			ramBytes := parseFloat(c.AdapterRAM)
			result = append(result, GPUInfo{
				Name:          strings.TrimSpace(c.Name),
				MemoryGB:      int(ramBytes / (1024 * 1024 * 1024)),
				DriverVersion: strings.TrimSpace(c.DriverVersion),
			})
		}
	}
	return result, nil
}

func getMonitors() ([]MonitorInfo, error) {
	jsonStr := psJSON(`Get-CimInstance WmiMonitorID -Namespace root/wmi | Select-Object ManufacturerName,ProductName,SerialNumberID,UserFriendlyName 2>$null`)
	var monitors []struct {
		ManufacturerName []uint16 `json:"ManufacturerName"`
		ProductName      []uint16 `json:"ProductName"`
		SerialNumberID   []uint16 `json:"SerialNumberID"`
	}
	var result []MonitorInfo
	if err := json.Unmarshal([]byte(jsonStr), &monitors); err == nil {
		for _, m := range monitors {
			result = append(result, MonitorInfo{
				Manufacturer: decodeUint16(m.ManufacturerName),
				Model:        decodeUint16(m.ProductName),
				SerialNumber: decodeUint16(m.SerialNumberID),
			})
		}
	}
	return result, nil
}

func getNetwork() ([]NetworkInfo, error) {
	jsonStr := psJSON(`Get-CimInstance Win32_NetworkAdapter | Where-Object { $_.PhysicalAdapter -eq $true -and $_.MACAddress -ne $null } | Select-Object Name,MACAddress,Speed,DhcpEnabled`)
	var adapters []struct {
		Name        string `json:"Name"`
		MACAddress  string `json:"MACAddress"`
		Speed       int64  `json:"Speed"`
		DhcpEnabled bool   `json:"DhcpEnabled"`
	}
	var result []NetworkInfo
	if err := json.Unmarshal([]byte(jsonStr), &adapters); err == nil {
		for _, a := range adapters {
			result = append(result, NetworkInfo{
				Name:        strings.TrimSpace(a.Name),
				MACAddress:  strings.TrimSpace(a.MACAddress),
				Speed:       a.Speed,
				DHCPEnabled: a.DhcpEnabled,
			})
		}
	}

	ipJSON := psJSON(`Get-CimInstance Win32_NetworkAdapterConfiguration | Where-Object { $_.IPAddress -ne $null } | Select-Object Index,IPAddress,@{N='DHCPEnabled';E={$_.DHCPEnabled}}`)
	var ips []struct {
		Index       int      `json:"Index"`
		IPAddress   []string `json:"IPAddress"`
		DHCPEnabled bool     `json:"DHCPEnabled"`
	}
	if err := json.Unmarshal([]byte(ipJSON), &ips); err == nil {
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

func getSoftware() ([]SoftwareInfo, error) {
	script := `
$paths = @(
	"HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*",
	"HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*"
)
Get-ItemProperty $paths 2>$null | Where-Object { $_.DisplayName -ne $null } | 
Select-Object @{N='Name';E={$_.DisplayName}}, @{N='Version';E={$_.DisplayVersion}}, @{N='Publisher';E={$_.Publisher}}, @{N='InstallDate';E={$_.InstallDate}} | 
ConvertTo-Json -Compress
`
	out, err := psRaw(script)
	if err != nil {
		return nil, err
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

// ── Helpers ──────────────────────────────────────

func ps(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func psRaw(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func psJSON(script string) string {
	fullScript := script + " | ConvertTo-Json -Compress -Depth 2"
	cmd := exec.Command("powershell", "-NoProfile", "-Command", fullScript)
	out, err := cmd.Output()
	if err != nil {
		return "[]"
	}
	return strings.TrimSpace(string(out))
}

func psGet(wmiClass, property string) string {
	script := fmt.Sprintf("(%s).%s", wmiClass, property)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

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
