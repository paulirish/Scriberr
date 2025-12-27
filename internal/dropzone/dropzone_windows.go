//go:build windows

package dropzone

import (
	"os"
	"syscall"
	"time"
)

func getFileCreationTimeOS(info os.FileInfo) time.Time {
	if stat, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, stat.CreationTime.Nanoseconds())
	}
	return info.ModTime()
}
