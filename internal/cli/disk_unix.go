//go:build !windows

package cli

import "syscall"

func getAvailableDiskSpace(path string) (float64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	// Available bytes = available blocks * block size
	avail := float64(stat.Bavail) * float64(stat.Bsize)
	return avail / (1 << 30), nil // Convert to GB
}
