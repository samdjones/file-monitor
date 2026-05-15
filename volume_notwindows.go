//go:build !windows

package main

import "path/filepath"

func getVolumeLabel(mountPoint string) string {
	return filepath.Base(mountPoint)
}
