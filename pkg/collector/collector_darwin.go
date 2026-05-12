//go:build darwin

package collector

import (
	"context"
	"encoding/json"
	"fmt"
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
func Run(runner CommandRunner) (*Inventory, error) {
	inv := &Inventory{
		CollectorVersion: "1.0.0",
	}

	inv.Hostname = runWithTimeout(runner, CmdTimeoutFast, "hostname")
	inv.System = getSystemInfo(runner)
	inv.CPU = getCPU(runner)
	inv.RAM = getRAM(runner)
	inv.Storage = getStorage(runner)
	inv.Motherboard = getMotherboard(runner)
	inv.GPU = getGPU(runner)
	inv.Monitors = getMonitors(runner)
	inv.Network = getNetwork(runner)

	return inv, nil
}

func getSystemInfo(runner CommandRunner) SystemInfo {
	si := SystemInfo{}
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

func getCPU(runner CommandRunner) CPUInfo {
	cpu := CPUInfo{}
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

func getRAM(runner CommandRunner) RAMInfo {
	ram := RAMInfo{}
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
				Name     string `json:"_name"`
				Size     string `json:"dimm_size"`
				Type     string `json:"dimm_type"`
				Speed    string `json:"dimm_speed"`
			} `json:"_items"`
		} `json:"SPMemoryDataType"`
	}
	if err := json.Unmarshal([]byte(sp), &mem); err == nil && len(mem.SPMemoryDataType) > 0 {
		for _, slot := range mem.SPMemoryDataType[0].Items {
			sizeGB := parseSizeToGB(strings.TrimSpace(slot.Size))
			speed, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(slot.Speed), " MHz"))
			ram.Slots = append(ram.Slots, RAMSlot{
				BankLabel: strings.TrimSpace(slot.Name),
				SizeGB:    sizeGB,
				SpeedMHz:  speed,
				Type:      strings.TrimSpace(slot.Type),
			})
		}
	}

	return ram
}

func getStorage(runner CommandRunner) []StorageInfo {
	var result []StorageInfo

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
			sizeGB := parseSizeToGB(s.Size)
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

			result = append(result, StorageInfo{
				Model:  model,
				SizeGB: sizeGB,
				Type:   diskType,
			})
		}
	}

	return result
}

func getMotherboard(runner CommandRunner) MotherboardInfo {
	mb := MotherboardInfo{}
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

func getGPU(runner CommandRunner) []GPUInfo {
	var result []GPUInfo

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
			gpu := GPUInfo{
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

func getMonitors(runner CommandRunner) []MonitorInfo {
	var result []MonitorInfo

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
				mi := MonitorInfo{
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

func getNetwork(runner CommandRunner) []NetworkInfo {
	var result []NetworkInfo

	out := runWithTimeout(runner, CmdTimeoutMedium, "ifconfig")
	if out == "" {
		return result
	}

	re := regexp.MustCompile(`^([a-z0-9]+):.*`)
	macRe := regexp.MustCompile(`ether\s+([0-9a-f:]+)`)
	ipRe := regexp.MustCompile(`inet\s+(\d+\.\d+\.\d+\.\d+)`)

	var current string
	for _, line := range strings.Split(string(out), "\n") {
		if m := re.FindStringSubmatch(line); m != nil {
			current = m[1]
			if current == "lo0" {
				continue
			}
			result = append(result, NetworkInfo{Name: current})
		} else if m := macRe.FindStringSubmatch(line); m != nil && len(result) > 0 {
			result[len(result)-1].MACAddress = m[1]
		} else if m := ipRe.FindStringSubmatch(line); m != nil && len(result) > 0 {
			result[len(result)-1].IPAddress = m[1]
		}
	}

	// Enriquecer con velocidad desde ifconfig
	for i := range result {
		speed := runWithTimeout(runner, CmdTimeoutMedium, "ifconfig", result[i].Name, "media")
		if speed != "" {
			re := regexp.MustCompile(`(\d+)baseT`)
			if m := re.FindStringSubmatch(speed); m != nil {
				if s, err := strconv.ParseInt(m[1], 10, 64); err == nil {
					result[i].Speed = s
				}
			}
		}
	}

	return result
}

func parseSizeToGB(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	re := regexp.MustCompile(`([\d.]+)\s*(TB|GB|KB|MB)`)
	m := re.FindStringSubmatch(s)
	if len(m) < 3 {
		return 0
	}
	val, _ := strconv.ParseFloat(m[1], 64)
	switch m[2] {
	case "TB":
		return int(val * 1000)
	case "KB":
		return int(val / (1000 * 1000))
	case "MB":
		return int(val / 1000)
	default:
		return int(val)
	}
}
