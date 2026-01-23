package platform

import (
	"os"
	"runtime"
	"strings"
)

const (
	PlatformRaspberryPi = "raspberry_pi"
	PlatformUbuntu      = "ubuntu"
	PlatformWindows     = "windows"
	PlatformVM          = "vm"
	PlatformLinux       = "linux"
)

type Info struct {
	Platform       string
	OS             string
	Arch           string
	Virtualization bool
}

func Detect(override string) *Info {
	if override != "" {
		return &Info{
			Platform:       override,
			OS:             runtime.GOOS,
			Arch:           runtime.GOARCH,
			Virtualization: false,
		}
	}

	info := &Info{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	info.Virtualization = detectVirtualization()

	if info.Virtualization {
		info.Platform = PlatformVM
		return info
	}

	switch info.OS {
	case "windows":
		info.Platform = PlatformWindows
	case "linux":
		if info.Arch == "arm64" && isRaspberryPi() {
			info.Platform = PlatformRaspberryPi
		} else if isUbuntu() {
			info.Platform = PlatformUbuntu
		} else {
			info.Platform = PlatformLinux
		}
	default:
		info.Platform = PlatformLinux
	}

	return info
}

func detectVirtualization() bool {
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/sys/class/dmi/id/product_name")
		if err == nil {
			product := strings.ToLower(string(data))
			if strings.Contains(product, "virtualbox") ||
				strings.Contains(product, "vmware") ||
				strings.Contains(product, "kvm") ||
				strings.Contains(product, "qemu") ||
				strings.Contains(product, "virtual") {
				return true
			}
		}

		data, err = os.ReadFile("/proc/cpuinfo")
		if err == nil {
			cpuinfo := strings.ToLower(string(data))
			if strings.Contains(cpuinfo, "hypervisor") {
				return true
			}
		}
	}

	if runtime.GOOS == "windows" {
		data, err := os.ReadFile("C:\\Windows\\System32\\drivers\\vmmouse.sys")
		if err == nil && len(data) > 0 {
			return true
		}
		data, err = os.ReadFile("C:\\Windows\\System32\\drivers\\vmhgfs.sys")
		if err == nil && len(data) > 0 {
			return true
		}
	}

	return false
}

func isRaspberryPi() bool {
	data, err := os.ReadFile("/proc/device-tree/model")
	if err == nil {
		model := strings.ToLower(string(data))
		if strings.Contains(model, "raspberry") {
			return true
		}
	}

	data, err = os.ReadFile("/sys/firmware/devicetree/base/model")
	if err == nil {
		model := strings.ToLower(string(data))
		if strings.Contains(model, "raspberry") {
			return true
		}
	}

	return false
}

func isUbuntu() bool {
	data, err := os.ReadFile("/etc/os-release")
	if err == nil {
		osRelease := strings.ToLower(string(data))
		if strings.Contains(osRelease, "ubuntu") {
			return true
		}
	}

	return false
}
