//go:build windows

package info

import (
	"syscall"

	"golang.org/x/sys/windows"
)

func diskInfo() (string, string) {
	rootPath, err := syscall.UTF16PtrFromString(`C:\`)
	if err != nil {
		return formatUnknown("UTF16PtrFromString", err, ""), formatUnknown("UTF16PtrFromString", err, "")
	}

	var freeBytesAvailable uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(
		rootPath,
		&freeBytesAvailable,
		&totalNumberOfBytes,
		&totalNumberOfFreeBytes,
	); err != nil {
		return formatUnknown("GetDiskFreeSpaceEx", err, ""), formatUnknown("GetDiskFreeSpaceEx", err, "")
	}

	return formatBytes(totalNumberOfBytes), formatBytes(totalNumberOfFreeBytes)
}
