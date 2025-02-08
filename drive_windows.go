//go:build windows

package godrivelist

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	getLogicalDrives     = kernel32.NewProc("GetLogicalDrives")
	getVolumeInformation = kernel32.NewProc("GetVolumeInformationW")
	getDiskFreeSpaceEx   = kernel32.NewProc("GetDiskFreeSpaceExW")
	getVolumePathName    = kernel32.NewProc("GetVolumePathNameW")
	getDriveType         = kernel32.NewProc("GetDriveTypeW")
)

func list() ([]Drive, error) {
	var drives []Drive

	// Get bitmask of available drives
	mask, _, _ := getLogicalDrives.Call()

	// Iterate through all possible drive letters
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}

		driveLetter := string(rune('A'+i)) + ":\\"
		rootPath := syscall.StringToUTF16Ptr(driveLetter)

		// Get drive type
		driveType, _, _ := getDriveType.Call(uintptr(unsafe.Pointer(rootPath)))

		// Windows drive types
		const (
			DRIVE_UNKNOWN     = 0
			DRIVE_NO_ROOT_DIR = 1
			DRIVE_REMOVABLE   = 2
			DRIVE_FIXED       = 3
			DRIVE_REMOTE      = 4
			DRIVE_CDROM       = 5
			DRIVE_RAMDISK     = 6
		)

		// Skip CD-ROM drives
		if driveType == DRIVE_CDROM {
			continue
		}

		// Get volume information
		var volumeNameBuffer [256]uint16
		var volumeSerialNumber uint32
		var maximumComponentLength uint32
		var fileSystemFlags uint32
		var fileSystemNameBuffer [256]uint16

		getVolumeInformation.Call(
			uintptr(unsafe.Pointer(rootPath)),
			uintptr(unsafe.Pointer(&volumeNameBuffer[0])),
			uintptr(len(volumeNameBuffer)),
			uintptr(unsafe.Pointer(&volumeSerialNumber)),
			uintptr(unsafe.Pointer(&maximumComponentLength)),
			uintptr(unsafe.Pointer(&fileSystemFlags)),
			uintptr(unsafe.Pointer(&fileSystemNameBuffer[0])),
			uintptr(len(fileSystemNameBuffer)),
		)

		// Get disk space information
		var freeBytesAvailable int64
		var totalBytes int64
		var totalFreeBytes int64

		getDiskFreeSpaceEx.Call(
			uintptr(unsafe.Pointer(rootPath)),
			uintptr(unsafe.Pointer(&freeBytesAvailable)),
			uintptr(unsafe.Pointer(&totalBytes)),
			uintptr(unsafe.Pointer(&totalFreeBytes)),
		)

		volumeName := syscall.UTF16ToString(volumeNameBuffer[:])
		if volumeName == "" {
			volumeName = "Local Disk"
		}

		description := fmt.Sprintf("%s (%s)", volumeName, strings.TrimRight(driveLetter, "\\"))

		drive := Drive{
			Device:      fmt.Sprintf("\\\\.\\%s", strings.TrimRight(driveLetter, "\\")),
			DisplayName: strings.TrimRight(driveLetter, "\\"),
			Description: description,
			Size:        totalBytes,
			Mountpoints: []Mountpoint{{Path: strings.TrimRight(driveLetter, "\\")}},
			Raw:         fmt.Sprintf("\\\\.\\%s", strings.TrimRight(driveLetter, "\\")),
			Protected:   (fileSystemFlags & 0x00080000) != 0, // READ_ONLY_VOLUME
			System:      driveType == DRIVE_FIXED,
		}

		drives = append(drives, drive)
	}

	return drives, nil
}
