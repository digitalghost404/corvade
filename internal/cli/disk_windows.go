//go:build windows

package cli

import "fmt"

func getAvailableDiskSpace(path string) (float64, error) {
	return 0, fmt.Errorf("disk space check not implemented on Windows")
}
