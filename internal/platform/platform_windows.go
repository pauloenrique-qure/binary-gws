//go:build windows

package platform

import "golang.org/x/sys/windows"

const verNTWorkstation = 0x1

func isWindowsServer() bool {
	info := windows.RtlGetVersion()
	return info.ProductType != verNTWorkstation
}
