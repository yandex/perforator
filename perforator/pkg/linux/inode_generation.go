package linux

import (
	"os"

	"golang.org/x/sys/unix"
)

//nolint:st1003
const FS_IOC_GETVERSION = 0x80087601

func GetInodeGeneration(file *os.File) (int, error) {
	return unix.IoctlGetInt(int(file.Fd()), FS_IOC_GETVERSION)
}
