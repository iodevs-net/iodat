//go:build linux

package collector

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Run recolecta todo el inventario en Linux.
func Run() (*Inventory, error) {
	inv := &Inventory{
		CollectorVersion: "1.0.0",
	}

	inv.Hostname = readFile("/proc/sys/kernel/hostname")
	inv.System = getSystemInfo()
	inv.CPU = getCPU()
	inv.RAM = getRAM()
	inv.Storage = getStorage()
	inv.Motherboard = getMotherboard()
	inv.GPU = getGPU()
	inv.Monitors = getMonitors()
	inv.Network = getNetwork()

	return inv, nil
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func getSystemInfo() SystemInfo {
	si := SystemInfo{}
	si.Manufacturer = readFile("/sys/class/dmi/id/sys_vendor")
	si.Model = readFile("/sys/class/dmi/id/product_name")
	si.SerialNumber = readFile("/sys/class/dmi/id/product_serial")

	// OS info
	osRelease, _ := os.ReadFile("/etc/os-release")
	for _, line := range strings.Split(string(osRelease), "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			si.OS = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}
	if si.OS == "" {
		si.OS = readFile("/etc/hostname") // fallback
	}

	// UName
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		si.OSVersion = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("uname", "-m").Output(); err == nil {
		si.OSArchitecture = strings.TrimSpace(string(out))
	}

	return si
}

func getCPU() CPUInfo {
	cpu := CPUInfo{}
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return cpu
	}

	lines := strings.Split(string(data), "\n")
	cores := make(map[string]bool)
	logical := 0
	maxMHz := 0.0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "model name":
			if cpu.Name == "" {
				cpu.Name = val
			}
		case "cpu MHz":
			if mhz, err := strconv.ParseFloat(val, 64); err == nil && mhz > maxMHz {
				maxMHz = mhz
			}
		case "physical id":
			cores[val] = true
		case "processor":
			logical++
		}
	}

	cpu.Cores = len(cores)
	cpu.LogicalProcessors = logical
	cpu.MaxClockMHz = int(maxMHz)
	cpu.NameClean = cleanCPUName(cpu.Name)

	return cpu
}

func getRAM() RAMInfo {
	ram := RAMInfo{}
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return ram
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				kb, _ := strconv.ParseFloat(parts[1], 64)
				ram.TotalGB = int(kb / (1024 * 1024))
				ram.Formatted = fmt.Sprintf("%dGB", ram.TotalGB)
			}
			break
		}
	}

	return ram
}

func getStorage() []StorageInfo {
	var result []StorageInfo

	// Leer discos desde /sys/block
	devices, err := os.ReadDir("/sys/block")
	if err != nil {
		return result
	}

	for _, dev := range devices {
		name := dev.Name()
		// Saltar loop, ram, sr (CD-ROM)
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "sr") {
			continue
		}

		s := StorageInfo{
			Model: readFile(fmt.Sprintf("/sys/block/%s/device/model", name)),
			SizeGB: parseBlocksToGB(readFile(fmt.Sprintf("/sys/block/%s/size", name))),
		}

		// Serial
		serial := readFile(fmt.Sprintf("/sys/block/%s/device/serial", name))
		wwwn := readFile(fmt.Sprintf("/sys/block/%s/device/wwwn", name))
		s.SerialNumber = serial
		if s.SerialNumber == "" {
			s.SerialNumber = wwwn
		}

		// Tipo (SSD/HDD)
		rotational := readFile(fmt.Sprintf("/sys/block/%s/queue/rotational", name))
		if rotational == "0" {
			s.Type = "SSD"
		} else {
			s.Type = "HDD"
		}

		// Interface
		if strings.HasPrefix(name, "nvme") {
			s.Interface = "NVMe"
		} else if strings.HasPrefix(name, "sd") {
			s.Interface = "SATA"
		} else if strings.HasPrefix(name, "vd") {
			s.Interface = "VirtIO"
		}

		if s.Model != "" || s.SizeGB > 0 {
			result = append(result, s)
		}
	}

	return result
}

func getMotherboard() MotherboardInfo {
	mb := MotherboardInfo{}
	mb.Manufacturer = readFile("/sys/class/dmi/id/board_vendor")
	mb.Product = readFile("/sys/class/dmi/id/board_name")
	mb.SerialNumber = readFile("/sys/class/dmi/id/board_serial")
	mb.BIOSVersion = readFile("/sys/class/dmi/id/bios_version")
	mb.BIOSDate = readFile("/sys/class/dmi/id/bios_date")
	return mb
}

func getGPU() []GPUInfo {
	var result []GPUInfo

	// Método 1: lspci (estándar en la mayoría de distros)
	if out, err := exec.Command("lspci", "-mm").Output(); err == nil {
		re := regexp.MustCompile(`(?i)(VGA|3D|Display).*\[(.*?)\]`)
		for _, line := range strings.Split(string(out), "\n") {
			if re.MatchString(line) {
				parts := strings.Split(line, "\"")
				if len(parts) >= 5 {
					result = append(result, GPUInfo{
						Name: strings.TrimSpace(parts[3]),
					})
				}
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	// Método 2: /sys/class/drm (no requiere lspci)
	devices, err := os.ReadDir("/sys/class/drm")
	if err == nil {
		for _, dev := range devices {
			name := dev.Name()
			if !strings.HasPrefix(name, "card") || strings.Contains(name, "-") {
				continue
			}
			// Leer vendor del dispositivo
			vendorPath := fmt.Sprintf("/sys/class/drm/%s/device/vendor", name)
			devicePath := fmt.Sprintf("/sys/class/drm/%s/device/device", name)
			vendor := readFile(vendorPath)
			devID := readFile(devicePath)
			if vendor != "" || devID != "" {
				result = append(result, GPUInfo{
					Name: fmt.Sprintf("GPU %s:%s", strings.TrimPrefix(vendor, "0x"), strings.TrimPrefix(devID, "0x")),
				})
			}
		}
	}

	return result
}

func getMonitors() []MonitorInfo {
	// EDID parsing from /sys/class/drm/
	var result []MonitorInfo
	devices, err := os.ReadDir("/sys/class/drm/")
	if err != nil {
		return result
	}

	for _, dev := range devices {
		name := dev.Name()
		if !strings.Contains(name, "-") {
			continue
		}
		edidPath := fmt.Sprintf("/sys/class/drm/%s/edid", name)
		edid, err := os.ReadFile(edidPath)
		if err != nil {
			continue
		}
		if len(edid) < 128 {
			continue
		}

		mi := MonitorInfo{}
		// Manufacturer ID (bytes 8-9 en EDID)
		if len(edid) > 10 {
			mf := string([]byte{edid[8] + 'A' - 1, edid[9] + 'A' - 1, edid[10] + 'A' - 1})
			mi.Manufacturer = mf
		}
		// Product code (bytes 10-11)
		if len(edid) > 12 {
			mi.Model = fmt.Sprintf("Monitor %02x%02x", edid[11], edid[10])
		}
		// Serial (bytes 12-15)
		if len(edid) > 16 {
			mi.SerialNumber = fmt.Sprintf("%02x%02x%02x%02x", edid[12], edid[13], edid[14], edid[15])
		}

		if mi.Model != "" {
			result = append(result, mi)
		}
	}

	return result
}

func getNetwork() []NetworkInfo {
	var result []NetworkInfo

	// /sys/class/net
	interfaces, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return result
	}

	for _, iface := range interfaces {
		name := iface.Name()
		if name == "lo" {
			continue
		}

		n := NetworkInfo{
			Name:       name,
			MACAddress: readFile(fmt.Sprintf("/sys/class/net/%s/address", name)),
		}

		// Speed
		speedStr := readFile(fmt.Sprintf("/sys/class/net/%s/speed", name))
		n.Speed = parseInt64(speedStr)

		// IP via ip addr
		out, _ := exec.Command("ip", "-json", "addr", "show", name).Output()
		if len(out) > 0 {
			// Simple parsing: buscar inet
			re := regexp.MustCompile(`inet\s+(\d+\.\d+\.\d+\.\d+)`)
			matches := re.FindStringSubmatch(string(out))
			if len(matches) > 1 {
				n.IPAddress = matches[1]
			}
		}

		result = append(result, n)
	}

	return result
}

func parseBlocksToGB(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	blocks, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	// Un block = 512 bytes
	return int(blocks * 512 / (1000 * 1000 * 1000))
}

func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
