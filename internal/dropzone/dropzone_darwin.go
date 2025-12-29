//go:build darwin

package dropzone

import (
	"os"
	"syscall"
	"time"
)

func getFileCreationTimeOS(info os.FileInfo) time.Time {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
	}
	return info.ModTime()
}
