//go:build darwin

package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ionet-cl/iodat/pkg/inventory"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// runWithTimeout executes a command with a context timeout and returns
// trimmed stdout. Returns empty string if the command fails or times out.
func runWithTimeout(runner CommandRunner, timeout time.Duration, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := runner.Run(ctx, name, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Run recolecta todo el inventario en macOS.
// Compatible con macOS 12 (Monterey) en adelante.
func Run(runner CommandRunner, _ FileSystem) (*inventory.Inventory, error) {
	inv := &inventory.Inventory{
		CollectorVersion: "1.0.0",
	}
	var errs PartialErrors

	inv.Hostname = runWithTimeout(runner, CmdTimeoutFast, "hostname")
	if inv.Hostname == "" {
		errs.Add("hostname: no se pudo obtener")
	}

	inv.System = getSystemInfo(runner)
	if inv.System.Model == "" {
		errs.Add("system: info de hardware incompleta")
	}

	inv.CPU = getCPU(runner)
	if inv.CPU.Name == "" {
		errs.Add("cpu: no se pudo detectar")
	}

	inv.RAM = getRAM(runner)
	if inv.RAM.TotalGB == 0 {
		errs.Add("ram: no se pudo detectar memoria")
	}

	inv.Storage = getStorage(runner)
	if len(inv.Storage) == 0 {
		errs.Add("storage: no se detectaron discos")
	}

	inv.Motherboard = getMotherboard(runner)
	inv.GPU = getGPU(runner)
	if len(inv.GPU) == 0 {
		errs.Add("gpu: no se detectaron adaptadores gráficos")
	}

	inv.Monitors = getMonitors(runner)
	if len(inv.Monitors) == 0 {
		errs.Add("monitors: no se detectaron monitores")
	}

	inv.Network = getNetwork(runner)
	if len(inv.Network) == 0 {
		errs.Add("network: no se detectaron interfaces")
	}

	return inv, errs.Err()
}

func getSystemInfo(runner CommandRunner) inventory.SystemInfo {
	si := inventory.SystemInfo{}
	sp := runWithTimeout(runner, CmdTimeoutSlow, "system_profiler", "SPHardwareDataType", "-json")

	var hw struct {
		SPHardwareDataType []struct {
			SerialNumber string `json:"serial_number"`
			MachineName  string `json:"machine_name"`
			MachineModel string `json:"machine_model"`
			OSVersion    string `json:"os_version"`
		} `json:"SPHardwareDataType"`
	}
	if err := json.Unmarshal([]byte(sp), &hw); err == nil && len(hw.SPHardwareDataType) > 0 {
		d := hw.SPHardwareDataType[0]
		si.Manufacturer = "Apple"
		si.Model = strings.TrimSpace(d.MachineModel)
		si.SerialNumber = strings.TrimSpace(d.SerialNumber)
	}

	sw := runWithTimeout(runner, CmdTimeoutSlow, "system_profiler", "SPSoftwareDataType", "-json")
	var swData struct {
		SPSoftwareDataType []struct {
			OSVersion     string `json:"os_version"`
			KernelVersion string `json:"kernel_version"`
		} `json:"SPSoftwareDataType"`
	}
	if err := json.Unmarshal([]byte(sw), &swData); err == nil && len(swData.SPSoftwareDataType) > 0 {
		si.OS = "macOS"
		si.OSVersion = strings.TrimSpace(swData.SPSoftwareDataType[0].OSVersion)
	}
	si.OSArchitecture = runWithTimeout(runner, CmdTimeoutFast, "uname", "-m")

	return si
}

func getCPU(runner CommandRunner) inventory.CPUInfo {
	cpu := inventory.CPUInfo{}
	cpu.Name = runWithTimeout(runner, CmdTimeoutFast, "sysctl", "-n", "machdep.cpu.brand_string")
	cpu.NameClean = cleanCPUName(cpu.Name)

	cores := runWithTimeout(runner, CmdTimeoutFast, "sysctl", "-n", "machdep.cpu.core_count")
	cpu.Cores, _ = strconv.Atoi(cores)
	logical := runWithTimeout(runner, CmdTimeoutFast, "sysctl", "-n", "machdep.cpu.thread_count")
	cpu.LogicalProcessors, _ = strconv.Atoi(logical)
	mhz := runWithTimeout(runner, CmdTimeoutFast, "sysctl", "-n", "hw.cpufrequency_max")
	if mhz != "" {
		if freq, err := strconv.Atoi(mhz); err == nil {
			cpu.MaxClockMHz = freq / (1000 * 1000) // hw.cpufrequency_max está en Hz
		}
	}

	return cpu
}

func getRAM(runner CommandRunner) inventory.RAMInfo {
	ram := inventory.RAMInfo{}
	memStr := runWithTimeout(runner, CmdTimeoutFast, "sysctl", "-n", "hw.memsize")
	memBytes, _ := strconv.ParseFloat(memStr, 64)
	ram.TotalGB = int(memBytes / (1024 * 1024 * 1024))
	if ram.TotalGB > 0 {
		ram.Formatted = fmt.Sprintf("%dGB", ram.TotalGB)
	}

	// Slots de RAM via system_profiler
	sp := runWithTimeout(runner, CmdTimeoutSlow, "system_profiler", "SPMemoryDataType", "-json")
	var mem struct {
		SPMemoryDataType []struct {
			Items []struct {
				Name  string `json:"_name"`
				Size  string `json:"dimm_size"`
				Type  string `json:"dimm_type"`
				Speed string `json:"dimm_speed"`
			} `json:"_items"`
		} `json:"SPMemoryDataType"`
	}
	if err := json.Unmarshal([]byte(sp), &mem); err == nil && len(mem.SPMemoryDataType) > 0 {
		for _, slot := range mem.SPMemoryDataType[0].Items {
			sizeGB := MustGB(strings.TrimSpace(slot.Size))
			speed, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(slot.Speed), " MHz"))
			ram.Slots = append(ram.Slots, inventory.RAMSlot{
				BankLabel: strings.TrimSpace(slot.Name),
				SizeGB:    sizeGB,
				SpeedMHz:  speed,
				Type:      strings.TrimSpace(slot.Type),
			})
		}
	}

	return ram
}

func getStorage(runner CommandRunner) []inventory.StorageInfo {
	var result []inventory.StorageInfo

	sp := runWithTimeout(runner, CmdTimeoutSlow, "system_profiler", "SPStorageDataType", "-json")
	var storage struct {
		SPStorageDataType []struct {
			ItemName      string `json:"_name"`
			MountPoint    string `json:"mount_point"`
			Size          string `json:"size"`
			FileSystem    string `json:"file_system"`
			PhysicalDrive string `json:"physical_drive"`
			MediumType    string `json:"medium_type"`
		} `json:"SPStorageDataType"`
	}
	if err := json.Unmarshal([]byte(sp), &storage); err == nil {
		for _, s := range storage.SPStorageDataType {
			sizeGB := MustGB(s.Size)
			model := strings.TrimSpace(s.ItemName)
			if model == "" {
				model = strings.TrimSpace(s.PhysicalDrive)
			}

			diskType := ""
			mt := strings.ToLower(s.MediumType)
			if strings.Contains(mt, "ssd") || strings.Contains(mt, "solid") {
				diskType = "SSD"
			} else if strings.Contains(mt, "hdd") || strings.Contains(mt, "rotational") {
				diskType = "HDD"
			}

			result = append(result, inventory.StorageInfo{
				Model:  model,
				SizeGB: sizeGB,
				Type:   diskType,
			})
		}
	}

	return result
}

func getMotherboard(runner CommandRunner) inventory.MotherboardInfo {
	mb := inventory.MotherboardInfo{}
	mb.Manufacturer = "Apple"
	mb.Product = runWithTimeout(runner, CmdTimeoutFast, "sysctl", "-n", "hw.model")

	// Serial del sistema (si no viene de SPHardwareDataType)
	if serial := runWithTimeout(runner, CmdTimeoutMedium, "ioreg", "-l"); serial != "" {
		re := regexp.MustCompile(`"IOPlatformSerialNumber"\s*=\s*"([^"]+)"`)
		if m := re.FindStringSubmatch(serial); m != nil {
			mb.SerialNumber = m[1]
		}
	}

	mb.BIOSVersion = runWithTimeout(runner, CmdTimeoutFast, "sysctl", "-n", "kern.osversion")
	return mb
}

func getGPU(runner CommandRunner) []inventory.GPUInfo {
	var result []inventory.GPUInfo

	sp := runWithTimeout(runner, CmdTimeoutSlow, "system_profiler", "SPDisplaysDataType", "-json")
	var disp struct {
		SPDisplaysDataType []struct {
			ChipsetModel string `json:"sppci_model"`
			VRAM         string `json:"spdisplays_vram"`
			Displays     []struct {
				Name       string `json:"_name"`
				Resolution string `json:"_spdisplays_resolution"`
			} `json:"spdisplays_ndrvs"`
		} `json:"SPDisplaysDataType"`
	}
	if err := json.Unmarshal([]byte(sp), &disp); err == nil {
		for _, d := range disp.SPDisplaysDataType {
			gpu := inventory.GPUInfo{
				Name: strings.TrimSpace(d.ChipsetModel),
			}
			// Parsear VRAM: "4096 MB" → 4
			if d.VRAM != "" {
				vram := strings.TrimSpace(d.VRAM)
				vram = strings.TrimSuffix(vram, " MB")
				if mb, err := strconv.Atoi(vram); err == nil {
					gpu.MemoryGB = mb / 1024
				}
			}
			if gpu.Name != "" {
				result = append(result, gpu)
			}
		}
	}

	return result
}

func getMonitors(runner CommandRunner) []inventory.MonitorInfo {
	var result []inventory.MonitorInfo

	sp := runWithTimeout(runner, CmdTimeoutSlow, "system_profiler", "SPDisplaysDataType", "-json")
	var disp struct {
		SPDisplaysDataType []struct {
			Displays []struct {
				Name       string `json:"_name"`
				Resolution string `json:"_spdisplays_resolution"`
				DisplayID  string `json:"_spdisplays_display_id"`
			} `json:"spdisplays_ndrvs"`
		} `json:"SPDisplaysDataType"`
	}
	if err := json.Unmarshal([]byte(sp), &disp); err == nil {
		for _, d := range disp.SPDisplaysDataType {
			for _, display := range d.Displays {
				mi := inventory.MonitorInfo{
					Model:      strings.TrimSpace(display.Name),
					Resolution: strings.TrimSpace(display.Resolution),
				}
				if display.DisplayID != "" {
					mi.SerialNumber = strings.TrimSpace(display.DisplayID)
				}
				if mi.Model != "" {
					result = append(result, mi)
				}
			}
		}
	}

	return result
}

func getNetwork(runner CommandRunner) []inventory.NetworkInfo {
	var result []inventory.NetworkInfo

	// Listar interfaces con ifconfig -l (formato estable, espacio-separado)
	ifaces := runWithTimeout(runner, CmdTimeoutMedium, "ifconfig", "-l")
	if ifaces == "" {
		return result
	}

	for _, name := range strings.Fields(ifaces) {
		if name == "lo0" {
			continue
		}

		ni := inventory.NetworkInfo{Name: name}

		// Obtener detalle por interfaz
		out := runWithTimeout(runner, CmdTimeoutMedium, "ifconfig", name)
		if out == "" {
			result = append(result, ni)
			continue
		}

		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)

			// MAC address: línea que comienza con "ether "
			if strings.HasPrefix(line, "ether ") {
				if parts := strings.Fields(line); len(parts) >= 2 {
					ni.MACAddress = parts[1]
				}
			}

			// IPv4: línea que comienza con "inet "
			if strings.HasPrefix(line, "inet ") {
				if parts := strings.Fields(line); len(parts) >= 2 {
					ni.IPAddress = parts[1]
				}
			}
		}

		// Velocidad via ifconfig <name> media
		speed := runWithTimeout(runner, CmdTimeoutMedium, "ifconfig", name, "media")
		if speed != "" {
			for _, word := range strings.Fields(speed) {
				if strings.HasSuffix(word, "baseT") {
					speedStr := strings.TrimSuffix(word, "baseT")
					if s, err := strconv.ParseInt(speedStr, 10, 64); err == nil {
						ni.Speed = s
						break
					}
				}
			}
		}

		result = append(result, ni)
	}

	return result
}
