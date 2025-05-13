// +build darwin

package godrivelist

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreFoundation -framework DiskArbitration -framework IOKit
#include "darwin/disklist.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// handleApfs processes APFS disks to handle virtual volumes correctly
func handleApfs(disks []Drive) {
	var apfs []Drive
	var other []Drive

	// Separate APFS disks from others
	for _, disk := range disks {
		if disk.Description == "AppleAPFSMedia" {
			apfs = append(apfs, disk)
		} else {
			other = append(other, disk)
		}
	}

	// Match APFS virtual disks to their physical drives
	for i, disk := range apfs {
		for j, source := range other {
			if source.DevicePath == disk.DevicePath && !source.IsVirtual {
				// Add virtual disk mountpoints to the physical disk
				other[j].Mountpoints = append(other[j].Mountpoints, disk.Mountpoints...)
				// Mark the virtual disk
				apfs[i].IsVirtual = true
				break
			}
		}
	}

	// Update the disks in place
	// First copy the other disks
	result := make([]Drive, 0, len(other)+len(apfs))
	result = append(result, other...)
	// Then add the apfs disks
	result = append(result, apfs...)

	// Copy back to the original slice (not very efficient but works for our purpose)
	copy(disks, result)
}

// list returns all connected drives in the system for macOS
func list() ([]Drive, error) {
	// Call the C function to get the drive list
	cDriveList := C.GetDriveList()
	defer C.FreeDriveList(cDriveList)

	// Check for errors
	if cDriveList.error != nil {
		errorMsg := C.GoString(cDriveList.error)
		return nil, fmt.Errorf("drive list error: %s", errorMsg)
	}

	// Convert C drive array to Go
	var drives []Drive
	driveSlice := (*[1 << 30]C.drive_t)(unsafe.Pointer(cDriveList.drives))[:cDriveList.count:cDriveList.count]

	for _, cDrive := range driveSlice {
		drive := Drive{
			Device:      C.GoString(cDrive.device),
			DisplayName: C.GoString(cDrive.display_name),
			Description: C.GoString(cDrive.description),
			Size:        int64(cDrive.size),
			Raw:         C.GoString(cDrive.raw),
			Protected:   bool(cDrive.protected),
			System:      bool(cDrive.system),
			Removable:   bool(cDrive.removable),
			IsVirtual:   bool(cDrive.virtual_drive),
			BlockSize:   int64(cDrive.block_size),
			DevicePath:  C.GoString(cDrive.device), // Using device path as device for now
		}

		// Convert mountpoints
		if cDrive.mountpoints != nil && cDrive.mountpoints_count > 0 {
			mountpointSlice := (*[1 << 30]C.mountpoint_t)(unsafe.Pointer(cDrive.mountpoints))[:cDrive.mountpoints_count:cDrive.mountpoints_count]
			drive.Mountpoints = make([]Mountpoint, cDrive.mountpoints_count)

			for i, cMountpoint := range mountpointSlice {
				drive.Mountpoints[i] = Mountpoint{
					Path: C.GoString(cMountpoint.path),
				}
				if cMountpoint.label != nil {
					drive.Mountpoints[i].Label = C.GoString(cMountpoint.label)
				}
			}
		} else {
			drive.Mountpoints = []Mountpoint{}
		}

		drives = append(drives, drive)
	}

	// Handle APFS virtual disks
	handleApfs(drives)

	return drives, nil
} 