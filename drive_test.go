package godrivelist

import (
	"testing"
)

func TestDriveList(t *testing.T) {
	drives, err := List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(drives) == 0 {
		t.Error("List() returned no drives, expected at least one (system drive)")
	}

	// Test that at least one drive has mountpoints
	hasMountpoints := false
	for _, drive := range drives {
		if len(drive.Mountpoints) > 0 {
			hasMountpoints = true
			break
		}
	}
	if !hasMountpoints {
		t.Error("No drives with mountpoints found, expected at least one")
	}

	// Test that each drive has valid fields
	for i, drive := range drives {
		if drive.Device == "" {
			t.Errorf("Drive[%d] has empty Device field", i)
		}
		if drive.DisplayName == "" {
			t.Errorf("Drive[%d] has empty DisplayName field", i)
		}
		if drive.Size <= 0 {
			t.Errorf("Drive[%d] has invalid Size: %d", i, drive.Size)
		}
		if drive.Raw == "" {
			t.Errorf("Drive[%d] has empty Raw field", i)
		}
	}

	// Test that at least one system drive exists
	hasSystemDrive := false
	for _, drive := range drives {
		if drive.System {
			hasSystemDrive = true
			break
		}
	}
	if !hasSystemDrive {
		t.Error("No system drive found, expected at least one")
	}
}