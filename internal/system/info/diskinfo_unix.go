//go:build !windows

package info

import "syscall"

func diskInfo() (string, string) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return formatUnknown("statfs", err, ""), formatUnknown("statfs", err, "")
	}

	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	return formatBytes(total), formatBytes(available)
}
