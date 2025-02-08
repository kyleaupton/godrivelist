//go:build linux

package godrivelist

import (
	"os/exec"
	"strings"
	"testing"
)

func TestLinuxDriveList(t *testing.T) {
	// Check if lsblk is available
	_, err := exec.LookPath("lsblk")
	if err != nil {
		t.Skip("lsblk not found, skipping Linux-specific tests")
	}

	drives, err := List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Get root device from /proc/mounts
	out, err := exec.Command("grep", "/ ", "/proc/mounts").Output()
	if err != nil {
		t.Skip("Could not read /proc/mounts, skipping root device test")
	}

	rootDevice := strings.Fields(string(out))[0]
	if !strings.HasPrefix(rootDevice, "/dev/") {
		t.Skip("Root device not in expected format, skipping root device test")
	}

	// Find the root device in our drives list
	foundRoot := false
	for _, drive := range drives {
		if strings.HasPrefix(rootDevice, drive.Device) {
			foundRoot = true
			// Root device should be marked as system
			if !drive.System {
				t.Error("Root device not marked as system drive")
			}
			break
		}
	}
	if !foundRoot {
		t.Error("Root device not found in drives list")
	}

	// Test that device paths are in expected format
	for _, drive := range drives {
		if !strings.HasPrefix(drive.Device, "/dev/") {
			t.Errorf("Invalid device path format: %s", drive.Device)
		}
		if !strings.HasPrefix(drive.Raw, "/dev/") {
			t.Errorf("Invalid raw device path format: %s", drive.Raw)
		}
	}
}
