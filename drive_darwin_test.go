//go:build darwin

package godrivelist

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDarwinDriveList(t *testing.T) {
	// Check if diskutil is available
	_, err := exec.LookPath("diskutil")
	if err != nil {
		t.Skip("diskutil not found, skipping macOS-specific tests")
	}

	drives, err := List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Get root device
	out, err := exec.Command("mount").Output()
	if err != nil {
		t.Skip("Could not run mount command, skipping root device test")
	}

	// Find the root device in mount output
	var rootDevice string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasSuffix(line, "on / (") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				rootDevice = fields[0]
			}
			break
		}
	}

	if rootDevice == "" {
		t.Skip("Could not determine root device, skipping root device test")
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
		if !strings.HasPrefix(drive.Raw, "/dev/r") {
			t.Errorf("Invalid raw device path format: %s", drive.Raw)
		}
	}

	// Test that at least one internal drive exists
	hasInternalDrive := false
	for _, drive := range drives {
		if drive.System {
			hasInternalDrive = true
			break
		}
	}
	if !hasInternalDrive {
		t.Error("No internal drive found")
	}
}
