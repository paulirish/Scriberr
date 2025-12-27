//go:build !darwin && !windows

package cli

import (
	"os"
	"time"
)

func getFileCreationTime(info os.FileInfo) time.Time {
	return info.ModTime()
}
