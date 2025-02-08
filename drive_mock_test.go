package godrivelist

import (
	"testing"
)

// mockDrive creates a mock Drive for testing
func mockDrive(device string, isSystem bool) Drive {
	return Drive{
		Device:      device,
		DisplayName: device,
		Description: "Mock Drive",
		Size:        1000204886016, // 1TB
		Mountpoints: []Mountpoint{{Path: "/mock"}},
		Raw:         device,
		Protected:   false,
		System:      isSystem,
	}
}

func TestMockDrive(t *testing.T) {
	// Create mock drives
	drive1 := mockDrive("/dev/mock1", true)
	drive2 := mockDrive("/dev/mock2", false)

	// Test system drive
	if !drive1.System {
		t.Error("Mock system drive not marked as system")
	}

	// Test non-system drive
	if drive2.System {
		t.Error("Mock non-system drive incorrectly marked as system")
	}

	// Test drive properties
	if drive1.Size <= 0 {
		t.Error("Mock drive has invalid size")
	}

	if len(drive1.Mountpoints) == 0 {
		t.Error("Mock drive has no mountpoints")
	}

	if drive1.Device == "" || drive1.Raw == "" {
		t.Error("Mock drive has empty device paths")
	}
}
