//go:build windows

package godrivelist

import (
	"strings"
	"testing"
)

func TestWindowsDriveList(t *testing.T) {
	drives, err := List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Test that C: drive exists and is marked as system
	foundSystemDrive := false
	for _, drive := range drives {
		if strings.HasPrefix(drive.DisplayName, "C:") {
			foundSystemDrive = true
			if !drive.System {
				t.Error("C: drive not marked as system drive")
			}
			break
		}
	}
	if !foundSystemDrive {
		t.Error("C: drive not found")
	}

	// Test that device paths are in Windows format
	for _, drive := range drives {
		if !strings.HasPrefix(drive.Device, "\\\\.\\") {
			t.Errorf("Invalid device path format: %s", drive.Device)
		}
		if !strings.HasPrefix(drive.Raw, "\\\\.\\") {
			t.Errorf("Invalid raw device path format: %s", drive.Raw)
		}
		// Test that display names are drive letters
		if len(drive.DisplayName) < 2 || !strings.HasSuffix(drive.DisplayName, ":") {
			t.Errorf("Invalid display name format: %s", drive.DisplayName)
		}
	}

	// Test that mountpoints are in Windows format
	for _, drive := range drives {
		for _, mp := range drive.Mountpoints {
			if len(mp.Path) < 2 || !strings.HasSuffix(mp.Path, ":") {
				t.Errorf("Invalid mountpoint format: %s", mp.Path)
			}
		}
	}
}