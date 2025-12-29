//go:build !darwin && !windows

package dropzone

import (
	"os"
	"time"
)

func getFileCreationTimeOS(info os.FileInfo) time.Time {
	return info.ModTime()
}
