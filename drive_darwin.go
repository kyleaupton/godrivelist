//go:build darwin

package godrivelist

/*
#cgo CFLAGS: -x objective-c -arch arm64
#cgo LDFLAGS: -framework Foundation -framework DiskArbitration -arch arm64
#include "disklist.h"
*/
import "C"
import (
	"unsafe"
)

func list() ([]Drive, error) {
	driveList := C.GetDriveList()
	defer C.FreeDriveList(driveList)

	drives := make([]Drive, driveList.count)
	driveSlice := unsafe.Slice(driveList.drives, driveList.count)

	for i := 0; i < int(driveList.count); i++ {
		driveInfo := driveSlice[i]

		var mountpoints []Mountpoint
		if driveInfo.mountpointsCount > 0 {
			mountpointSlice := unsafe.Slice(driveInfo.mountpoints, driveInfo.mountpointsCount)
			mountpoints = make([]Mountpoint, driveInfo.mountpointsCount)
			for j := 0; j < int(driveInfo.mountpointsCount); j++ {
				mountpoints[j] = Mountpoint{
					Path: C.GoString(mountpointSlice[j]),
				}
			}
		}

		drives[i] = Drive{
			Device:      C.GoString(driveInfo.device),
			DisplayName: C.GoString(driveInfo.displayName),
			Description: C.GoString(driveInfo.description),
			Size:        int64(driveInfo.size),
			Mountpoints: mountpoints,
			Raw:         C.GoString(driveInfo.raw),
			Protected:   bool(driveInfo.protected),
			System:      bool(driveInfo.system),
		}
	}

	return drives, nil
}
