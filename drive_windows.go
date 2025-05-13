// +build windows

package godrivelist

import (
	"fmt"
	"golang.org/x/sys/windows"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// Windows API constants and structs
const (
	GENERIC_READ     = 0x80000000
	FILE_SHARE_READ  = 0x00000001
	FILE_SHARE_WRITE = 0x00000002
	OPEN_EXISTING    = 3
	IOCTL_DISK_GET_DRIVE_GEOMETRY = 0x70000
	IOCTL_STORAGE_GET_DEVICE_NUMBER = 0x2D1080
	DRIVE_REMOVABLE = 2
	DRIVE_FIXED     = 3
	DRIVE_REMOTE    = 4
	DRIVE_CDROM     = 5
	DRIVE_RAMDISK   = 6
)

type DISK_GEOMETRY struct {
	Cylinders         int64
	MediaType         byte
	TracksPerCylinder uint32
	SectorsPerTrack   uint32
	BytesPerSector    uint32
}

type STORAGE_DEVICE_NUMBER struct {
	DeviceType      uint32
	DeviceNumber    uint32
	PartitionNumber uint32
}

// getDriveType returns the type of drive
func getDriveType(path string) uint32 {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getDriveTypeW := kernel32.NewProc("GetDriveTypeW")

	pathPtr, _ := syscall.UTF16PtrFromString(path)
	ret, _, _ := getDriveTypeW.Call(uintptr(unsafe.Pointer(pathPtr)))
	
	return uint32(ret)
}

// getVolumeInformation gets volume information
func getVolumeInformation(driveLetter string) (string, error) {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getVolumeInformationW := kernel32.NewProc("GetVolumeInformationW")

	volumeName := make([]uint16, 256)
	fileSystemName := make([]uint16, 256)
	serialNumber := uint32(0)
	maxComponentLength := uint32(0)
	fileSystemFlags := uint32(0)

	rootPathName, err := syscall.UTF16PtrFromString(driveLetter + "\\")
	if err != nil {
		return "", err
	}

	ret, _, err := getVolumeInformationW.Call(
		uintptr(unsafe.Pointer(rootPathName)),
		uintptr(unsafe.Pointer(&volumeName[0])),
		uintptr(len(volumeName)),
		uintptr(unsafe.Pointer(&serialNumber)),
		uintptr(unsafe.Pointer(&maxComponentLength)),
		uintptr(unsafe.Pointer(&fileSystemFlags)),
		uintptr(unsafe.Pointer(&fileSystemName[0])),
		uintptr(len(fileSystemName)),
	)

	if ret == 0 {
		return "", fmt.Errorf("failed to get volume information")
	}

	// Convert volumeName from UTF16 to string
	i := 0
	for volumeName[i] != 0 {
		i++
	}
	return string(utf16.Decode(volumeName[:i])), nil
}

// getDriveFreeSpace gets the total size of a drive
func getDriveSize(driveLetter string) (int64, error) {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getDiskFreeSpaceExW := kernel32.NewProc("GetDiskFreeSpaceExW")

	var freeBytesAvailable, totalBytes, totalFreeBytes int64

	rootPathName, err := syscall.UTF16PtrFromString(driveLetter + "\\")
	if err != nil {
		return 0, err
	}

	ret, _, err := getDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(rootPathName)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)

	if ret == 0 {
		return 0, fmt.Errorf("failed to get disk free space: %v", err)
	}

	return totalBytes, nil
}

// getDiskGeometry gets disk geometry information
func getDiskGeometry(handle windows.Handle) (DISK_GEOMETRY, error) {
	var geom DISK_GEOMETRY
	var bytesReturned uint32

	err := windows.DeviceIoControl(
		handle,
		IOCTL_DISK_GET_DRIVE_GEOMETRY,
		nil,
		0,
		(*byte)(unsafe.Pointer(&geom)),
		uint32(unsafe.Sizeof(geom)),
		&bytesReturned,
		nil,
	)

	return geom, err
}

// isWindowsSystemDrive checks if the drive is the system drive
func isWindowsSystemDrive(driveLetter string) bool {
	windir, err := windows.GetSystemWindowsDirectory()
	if err != nil {
		return false
	}
	return strings.ToUpper(string(windir[0])) == strings.ToUpper(driveLetter)
}

// list returns all connected drives in the system for Windows
func list() ([]Drive, error) {
	var drives []Drive
	
	// Get available drive letters
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getLogicalDrives := kernel32.NewProc("GetLogicalDrives")
	
	ret, _, _ := getLogicalDrives.Call()
	mask := uint32(ret)
	
	for i := 0; i < 26; i++ {
		if mask&(1<<i) > 0 {
			driveLetter := string('A' + i)
			driveRoot := driveLetter + ":"
			driveType := getDriveType(driveRoot + "\\")
			
			// Skip CD-ROM drives
			if driveType == DRIVE_CDROM {
				continue
			}
			
			volumeName, err := getVolumeInformation(driveRoot)
			if err != nil {
				// Drive might not be ready, skip it
				continue
			}
			
			size, err := getDriveSize(driveRoot)
			if err != nil {
				// Unable to get size, set to 0
				size = 0
			}
			
			// Try to open the physical drive to get more information
			physicalPath := fmt.Sprintf("\\\\.\\%s:", driveLetter)
			physicalPathPtr, _ := syscall.UTF16PtrFromString(physicalPath)
			
			handle, err := windows.CreateFile(
				physicalPathPtr,
				GENERIC_READ,
				FILE_SHARE_READ|FILE_SHARE_WRITE,
				nil,
				OPEN_EXISTING,
				0,
				0,
			)
			
			blockSize := int64(512) // Default block size
			if err == nil {
				// Get disk geometry
				geom, err := getDiskGeometry(handle)
				if err == nil {
					blockSize = int64(geom.BytesPerSector)
				}
				windows.CloseHandle(handle)
			}
			
			isRemovable := driveType == DRIVE_REMOVABLE
			isSystem := isWindowsSystemDrive(driveLetter)
			
			drive := Drive{
				Device:      driveRoot,
				DisplayName: driveRoot,
				Description: volumeName,
				Size:        size,
				Mountpoints: []Mountpoint{
					{
						Path:  driveRoot + "\\",
						Label: volumeName,
					},
				},
				Raw:       fmt.Sprintf("\\\\.\\%s:", driveLetter),
				Protected: false, // Windows doesn't provide this easily
				System:    isSystem,
				Removable: isRemovable,
				ReadOnly:  false, // Would need additional API calls
				BlockSize: blockSize,
			}
			
			drives = append(drives, drive)
		}
	}
	
	return drives, nil
} 