//go:build windows

package main

import "golang.org/x/sys/windows"

func getVolumeLabel(mountPoint string) string {
	rootPath, err := windows.UTF16PtrFromString(mountPoint)
	if err != nil {
		return ""
	}
	var buf [windows.MAX_PATH + 1]uint16
	if err := windows.GetVolumeInformation(rootPath, &buf[0], uint32(len(buf)), nil, nil, nil, nil, 0); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:])
}
