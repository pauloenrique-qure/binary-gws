//go:build !windows

package platform

func isWindowsServer() bool {
	return false
}
